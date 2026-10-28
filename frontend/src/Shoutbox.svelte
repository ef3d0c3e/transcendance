<script>
	import { onMount, tick } from 'svelte';
	import ChatBubble from '../icons/ChatBubble.svelte';
	import { relativeTime, fullDate } from './relative-time.js';
	import { ShoutboxConnection } from './shoutbox-socket.js';

	let { dataset } = $props();

	// Same as the server, counted in characters
	const MAX_LENGTH = 500;
	// Messages kept in memory
	const MAX_MESSAGES = 200;
	// Give up waiting for the server's answer after this long
	const PENDING_TIMEOUT_MS = 10_000;
	// Scrolled this close to the bottom follows new messages
	const STICK_THRESHOLD_PX = 48;
	// Relative times are updated this often while the panel is open
	const CLOCK_MS = 30_000;

	let messages = $state([]);
	let unread = $state(0);
	// `idle` until first opened, then `connecting`, `open` or `closed`
	let status = $state('idle');
	let closedReason = $state('');
	// Sent by the server, the browser's is used until then
	let locale = $state(undefined);
	// Read by relative times so they update
	let now = $state(Date.now());
	let open = $state(false);
	let compact = $state(false);

	let draft = $state('');
	let pending = $state(false);
	let error = $state('');
	let limitedUntil = $state(0);

	let root = $state(undefined);
	let trigger = $state(undefined);
	let panel = $state(undefined);
	let list = $state(undefined);
	let input = $state(undefined);

	let connection;
	let pendingTimer;
	let limitTimer;

	const length = $derived([...draft.trim()].length);
	const tooLong = $derived(length > MAX_LENGTH);
	const limited = $derived(limitedUntil > 0);
	const canSend = $derived(status === 'open' && !pending && !limited && length > 0 && !tooLong);

	// `current` is only there so Svelte re-renders the time when `now` changes
	function ago(date, current) {
		return current && relativeTime(date, locale);
	}

	function nearBottom() {
		return !list || list.scrollHeight - list.scrollTop - list.clientHeight < STICK_THRESHOLD_PX;
	}

	async function scrollToBottom() {
		await tick();
		if (list) {
			list.scrollTop = list.scrollHeight;
		}
	}

	// Add messages the client doesn't have yet, the server can repeat some after reconnecting
	function addMessages(incoming) {
		const lastId = messages.at(-1)?.id ?? 0;
		const fresh = incoming.filter((message) => message.id > lastId);
		if (fresh.length === 0) {
			return;
		}
		const follow = nearBottom() || fresh.some((message) => message.username === dataset.username);
		messages = [...messages, ...fresh].slice(-MAX_MESSAGES);
		if (!open) {
			unread += fresh.filter((message) => message.username !== dataset.username).length;
		}
		if (follow) {
			scrollToBottom();
		}
	}

	function settle() {
		pending = false;
		clearTimeout(pendingTimer);
	}

	function rateLimited(retryAfter) {
		clearTimeout(limitTimer);
		limitedUntil = Date.now() + retryAfter;
		limitTimer = setTimeout(() => {
			limitedUntil = 0;
			if (error === 'rate-limited') {
				error = '';
			}
		}, retryAfter);
	}

	function handleEvent(event) {
		switch (event.type) {
			case 'history': {
				locale = event.locale;
				addMessages(event.messages ?? []);
				break;
			}
			case 'message': {
				addMessages([event.message]);
				break;
			}
			case 'ack': {
				settle();
				draft = '';
				error = '';
				break;
			}
			case 'error': {
				settle();
				error = event.error;
				if (event.error === 'rate-limited') {
					rateLimited(event.retryAfter ?? 2000);
				}
				break;
			}
		}
	}

	function handleStatus(next, reason = '') {
		status = next;
		closedReason = reason;
		// An answer won't come on a new connection
		if (next !== 'open') {
			settle();
		}
	}

	function send(event) {
		event.preventDefault();
		if (!canSend) {
			return;
		}
		if (!connection.send(draft)) {
			return;
		}
		pending = true;
		error = '';
		pendingTimer = setTimeout(() => {
			pending = false;
		}, PENDING_TIMEOUT_MS);
	}

	function handleInputKeydown(event) {
		// Enter sends, shift+enter is ignored as messages are a single line
		if (event.key === 'Enter') {
			send(event);
		}
	}

	// Move the panel out of the compact header's side menu, where it can't be shown
	function portal(node, enabled) {
		if (enabled) {
			document.body.append(node);
		}
		return {
			destroy() {
				node.remove();
			},
		};
	}

	async function toggle() {
		open = !open;
		if (!open) {
			return;
		}
		compact = Boolean(root?.closest('header.is-compact'));
		if (compact) {
			const navToggle = document.querySelector('#nav-toggle');
			if (navToggle?.checked) {
				navToggle.checked = false;
				navToggle.dispatchEvent(new Event('change'));
			}
		}
		unread = 0;
		connection ??= new ShoutboxConnection({ onEvent: handleEvent, onStatus: handleStatus });
		await scrollToBottom();
		input?.focus();
	}

	function close() {
		open = false;
		trigger?.focus();
	}

	const errorLabels = $derived({
		empty: dataset.labelErrorEmpty,
		'too-long': dataset.labelErrorTooLong,
		'rate-limited': dataset.labelErrorRateLimited,
		internal: dataset.labelErrorInternal,
	});

	const closedLabels = $derived({
		unauthorized: dataset.labelClosedUnauthorized,
		'too-many': dataset.labelClosedTooMany,
		error: dataset.labelClosedError,
	});

	$effect(() => {
		if (!open) {
			return;
		}
		now = Date.now();
		const clock = setInterval(() => {
			now = Date.now();
		}, CLOCK_MS);
		return () => clearInterval(clock);
	});

	onMount(() => {
		function handleOutsidePointer(event) {
			if (open && !root?.contains(event.target) && !panel?.contains(event.target)) {
				open = false;
			}
		}

		function handleKeydown(event) {
			if (event.key === 'Escape' && open) {
				close();
			}
		}

		document.addEventListener('pointerdown', handleOutsidePointer);
		document.addEventListener('keydown', handleKeydown);
		return () => {
			document.removeEventListener('pointerdown', handleOutsidePointer);
			document.removeEventListener('keydown', handleKeydown);
			clearTimeout(pendingTimer);
			clearTimeout(limitTimer);
			connection?.close();
		};
	});
</script>

<div class="shout-root" bind:this={root}>
	<button
		bind:this={trigger}
		type="button"
		class="shout-trigger"
		aria-label={unread > 0 ? `${dataset.labelTitle}, ${unread} ${dataset.labelUnread}` : dataset.labelTitle}
		aria-haspopup="dialog"
		aria-expanded={open}
		aria-controls="shoutbox-panel"
		onclick={toggle}
	>
		<ChatBubble />
		<span class="shout-trigger-label">{dataset.labelTitle}</span>
		{#if unread > 0}
			<span class="shout-badge" aria-hidden="true">{unread > 9 ? '9+' : unread}</span>
		{/if}
	</button>

	{#if open}
		<div
			id="shoutbox-panel"
			class="shout-panel"
			class:is-floating={compact}
			role="dialog"
			aria-label={dataset.labelTitle}
			bind:this={panel}
			use:portal={compact}
		>
			<div class="shout-head">
				<span class="shout-title">{dataset.labelTitle}</span>
				<span class="shout-status" data-status={status} title={status === 'open' ? dataset.labelConnected : dataset.labelConnecting}>
					<span class="shout-status-dot" aria-hidden="true"></span>
					<span class="shout-status-text">{status === 'open' ? dataset.labelConnected : dataset.labelConnecting}</span>
				</span>
				<button type="button" class="shout-close" aria-label={dataset.labelClose} onclick={close}>
					<svg width="18" height="18" viewBox="0 0 24 24" aria-hidden="true">
						<path d="M6 6l12 12M18 6L6 18" stroke="currentColor" stroke-width="2" stroke-linecap="round" fill="none" />
					</svg>
				</button>
			</div>

			<ol class="shout-list" bind:this={list} aria-live="polite" aria-relevant="additions" aria-busy={status !== 'open'}>
				{#each messages as message (message.id)}
					<li class="shout-message" class:is-self={message.username === dataset.username}>
						<span class="shout-meta">
							<a class="shout-author" href="/profile/{encodeURIComponent(message.username)}">{message.username}</a>
							<time datetime={message.createdAt} title={fullDate(message.createdAt, locale)}>
								{ago(message.createdAt, now)}
							</time>
						</span>
						<span class="shout-body">{message.body}</span>
					</li>
				{:else}
					{#if status === 'open'}
						<li class="shout-empty">{dataset.labelEmpty}</li>
					{:else if status !== 'closed'}
						<li class="shout-empty">{dataset.labelConnecting}</li>
					{/if}
				{/each}
			</ol>

			{#if status === 'closed'}
				<p class="section-error shout-alert" role="alert">{closedLabels[closedReason] ?? dataset.labelClosedError}</p>
			{:else if error}
				<p class="section-error shout-alert" role="alert">{errorLabels[error] ?? dataset.labelErrorInternal}</p>
			{/if}

			<form class="shout-form" onsubmit={send}>
				<input
					bind:this={input}
					bind:value={draft}
					type="text"
					class="shout-input"
					placeholder={dataset.labelPlaceholder}
					aria-label={dataset.labelPlaceholder}
					aria-invalid={tooLong}
					autocomplete="off"
					disabled={status === 'closed'}
					onkeydown={handleInputKeydown}
				/>
				{#if length > MAX_LENGTH - 50}
					<span class="shout-count" class:is-over={tooLong} aria-live="polite">{MAX_LENGTH - length}</span>
				{/if}
				<button type="submit" class="shout-send" disabled={!canSend} aria-busy={pending}>
					{dataset.labelSend}
				</button>
			</form>
		</div>
	{/if}
</div>

<style>
	.shout-root {
		position: relative;
		display: inline-flex;
	}

	/* Same as the notification bell */
	.shout-trigger {
		position: relative;
		display: inline-flex;
		align-items: center;
		justify-content: center;

		min-height: 2.5rem;
		min-width: 2.5rem;
		padding: 0.45rem;

		color: var(--color-text);
		background: var(--color-bg-4);
		border-radius: 0.5rem;

		&:hover {
			background: color-mix(in srgb, var(--color-bg-4), black 10%);
		}

		&:active {
			transform: translateY(1px);
			background: color-mix(in srgb, var(--color-bg-4), black 20%);
		}

		&[aria-expanded='true'] {
			background: color-mix(in srgb, var(--color-bg-4), black 12%);
		}

		:global(svg) {
			width: 1.35rem;
			height: 1.35rem;
		}
	}

	.shout-badge {
		position: absolute;
		top: -0.35rem;
		right: -0.35rem;

		min-width: 1.25rem;
		height: 1.25rem;
		padding: 0 0.3rem;

		color: var(--color-text-alt);
		background: var(--color-accent-secondary);
		border: 2px solid var(--color-bg-2);
		border-radius: 999px;

		font-size: 0.7rem;
		font-weight: 700;
		line-height: calc(1.25rem - 4px);
		font-variant-numeric: tabular-nums;
		text-align: center;
	}

	/* Only labelled in the compact header's side menu */
	.shout-trigger-label {
		display: none;
	}

	:global(header.is-compact) .shout-root {
		flex-direction: column;
		align-items: stretch;
	}

	:global(header.is-compact) .shout-trigger {
		gap: 0.5rem;
		padding: 0.45rem 0.9rem;
		font-weight: 600;

		.shout-badge {
			position: static;
		}

		.shout-trigger-label {
			display: inline;
		}
	}

	.shout-panel {
		position: absolute;
		z-index: 1000;
		top: calc(100% + 0.5rem);
		right: 0;

		display: flex;
		flex-direction: column;

		width: min(24rem, calc(100vw - 2rem));
		height: min(32rem, calc(100dvh - 6rem));

		color: var(--color-text);
		background: var(--color-bg-2);
		border: 1px solid var(--color-border-1);
		border-radius: 0.75rem;
		box-shadow: 0 12px 32px rgb(0 0 0 / 18%);

		animation: shout-panel-in 140ms ease;
	}

	/* Moved to the body from the compact header, as a sheet at the bottom of the screen */
	.shout-panel.is-floating {
		position: fixed;
		top: auto;
		right: max(0.5rem, env(safe-area-inset-right));
		bottom: max(0.5rem, env(safe-area-inset-bottom));
		left: max(0.5rem, env(safe-area-inset-left));

		width: auto;
		height: min(32rem, calc(100dvh - 1rem));
	}

	@keyframes shout-panel-in {
		from {
			opacity: 0;
			transform: translateY(-4px);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.shout-panel {
			animation: none;
		}
	}

	.shout-head {
		display: flex;
		align-items: center;
		gap: 0.5rem;

		padding: 0.5rem 0.5rem 0.5rem 1rem;
		border-bottom: 1px solid var(--color-border-1);
	}

	.shout-title {
		font-weight: 700;
		white-space: nowrap;
	}

	.shout-status {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		margin-inline-end: auto;

		color: var(--color-text-muted);
		font-size: 0.8rem;
	}

	.shout-status-dot {
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 999px;
		background: var(--color-border-2);
	}

	.shout-status[data-status='open'] .shout-status-dot {
		background: var(--color-success-3);
	}

	.shout-status[data-status='connecting'] .shout-status-dot {
		animation: shout-pulse 1s ease-in-out infinite alternate;
	}

	.shout-status[data-status='closed'] .shout-status-dot {
		background: var(--color-error-3);
	}

	@keyframes shout-pulse {
		to {
			opacity: 0.3;
		}
	}

	.shout-close {
		display: inline-flex;
		align-items: center;
		justify-content: center;

		min-width: 2rem;
		min-height: 2rem;
		padding: 0.25rem;

		color: var(--color-text-muted);
		background: transparent;
		border-radius: 0.4rem;

		&:hover {
			color: var(--color-text);
			background: var(--color-bg-4);
		}
	}

	.shout-list {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 0.15rem;

		margin: 0;
		padding: 0.5rem;
		overflow-y: auto;
		overscroll-behavior: contain;

		list-style: none;

		scrollbar-width: thin;
		scrollbar-color: var(--color-border-2) transparent;
	}

	.shout-message {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;

		padding: 0.35rem 0.5rem;
		border-radius: 0.5rem;

		&:hover {
			background: var(--color-bg-3);
		}
	}

	.shout-meta {
		display: flex;
		align-items: baseline;
		gap: 0.5rem;

		time {
			color: var(--color-text-muted);
			font-size: 0.75rem;
			font-variant-numeric: tabular-nums;
		}
	}

	.shout-author {
		overflow: hidden;

		color: var(--color-accent-primary);
		font-size: 0.85rem;
		font-weight: 700;
		text-decoration: none;
		text-overflow: ellipsis;
		white-space: nowrap;

		&:hover {
			text-decoration: underline;
		}
	}

	.is-self .shout-author {
		color: var(--color-accent-secondary);
	}

	.shout-body {
		overflow-wrap: anywhere;
		white-space: pre-wrap;
	}

	.shout-empty {
		margin: auto;
		padding: 1rem;

		color: var(--color-text-muted);
		text-align: center;
	}

	.shout-alert {
		margin: 0 0.5rem 0.5rem;
	}

	.shout-form {
		display: flex;
		align-items: center;
		gap: 0.4rem;

		margin: 0;
		padding: 0.5rem;
		border-top: 1px solid var(--color-border-1);
	}

	.shout-input {
		flex: 1;
		min-width: 0;

		&[aria-invalid='true'] {
			border-color: var(--color-error-3);
		}
	}

	.shout-count {
		color: var(--color-text-muted);
		font-size: 0.75rem;
		font-variant-numeric: tabular-nums;

		&.is-over {
			color: var(--color-error-3);
			font-weight: 700;
		}
	}

	.shout-send {
		flex: none;
	}
</style>
