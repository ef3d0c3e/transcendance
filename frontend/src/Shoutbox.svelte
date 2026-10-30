<script>
	import { onMount, tick } from 'svelte';
	import ChatBubble from '../icons/ChatBubble.svelte';
	import ChatConversations from './ChatConversations.svelte';
	import ChatUserMenu from './ChatUserMenu.svelte';
	import { relativeTime, fullDate } from './relative-time.js';
	import { ShoutboxConnection } from './shoutbox-socket.js';

	let { dataset } = $props();

	// Same as the server, counted in characters
	const MAX_LENGTH = 500;
	// Messages kept in memory, per list
	const MAX_MESSAGES = 200;
	// Give up waiting for the server's answer after this long
	const PENDING_TIMEOUT_MS = 10_000;
	// Scrolled this close to the bottom follows new messages
	const STICK_THRESHOLD_PX = 48;
	// Relative times are updated this often while the panel is open
	const CLOCK_MS = 30_000;
	// Opens the chat on a conversation, used by private message notifications
	const CHAT_PARAM = 'chat';

	const me = $derived(dataset.username);

	// Shoutbox messages
	let messages = $state([]);
	let shoutUnread = $state(0);
	// Latest first
	let conversations = $state([]);
	let ignored = $state([]);

	// `shout` or `messages`
	let tab = $state('shout');
	// Username of the conversation shown in the messages tab, none for the list
	let active = $state('');
	let privates = $state([]);
	// Whether `privates` were received for `active`
	let loaded = $state(false);

	// `connecting`, `open` or `closed`
	let status = $state('connecting');
	let closedReason = $state('');
	// Sent by the server, the browser's is used until then
	let locale = $state(undefined);
	// Read by relative times so they update
	let now = $state(Date.now());
	let open = $state(false);
	let compact = $state(false);
	let visible = $state(document.visibilityState === 'visible');

	// Draft of each composer: `shout`, or `@<username>` for a conversation
	let drafts = $state({});
	// Requests waiting for their answer, in order: `{ kind, key }`
	let waiting = $state([]);
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
	// Conversation the server was last told is shown
	let sentView;

	const composerKey = $derived(tab === 'shout' ? 'shout' : active ? `@${active}` : '');
	const draft = $derived(drafts[composerKey] ?? '');
	const length = $derived([...draft.trim()].length);
	const tooLong = $derived(length > MAX_LENGTH);
	const limited = $derived(limitedUntil > 0);
	const pending = $derived(waiting.some((request) => request.kind === 'send'));
	const canSend = $derived(status === 'open' && !pending && !limited && length > 0 && !tooLong);

	// Usernames ignored right now
	const ignoring = $derived(new Set(ignored.filter((user) => Date.parse(user.until) > now).map((user) => user.username)));
	const shown = $derived(messages.filter((message) => !ignoring.has(message.username)));
	const privateUnread = $derived(conversations.reduce((total, conversation) => total + conversation.unread, 0));
	const unread = $derived(privateUnread + shoutUnread);
	// Conversation on screen, its new messages are read right away
	const viewed = $derived(open && visible && tab === 'messages' ? active : '');

	function sameUser(a, b) {
		return a.toLowerCase() === b.toLowerCase();
	}

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

	// Messages of `incoming` not in `current` yet, the server can repeat some after reconnecting
	function merge(current, incoming) {
		const ids = new Set(current.map((message) => message.id));
		const fresh = incoming.filter((message) => !ids.has(message.id));
		return [fresh, [...current, ...fresh].toSorted((a, b) => a.id - b.id).slice(-MAX_MESSAGES)];
	}

	function addMessages(incoming) {
		const follow = nearBottom() || incoming.some((message) => message.username === me);
		const [fresh, merged] = merge(messages, incoming);
		if (fresh.length === 0) {
			return;
		}
		messages = merged;
		if (!(open && tab === 'shout')) {
			shoutUnread += fresh.filter((message) => message.username !== me).length;
		}
		if (follow && tab === 'shout') {
			scrollToBottom();
		}
	}

	function addPrivate(message) {
		const partner = message.from === me ? message.to : message.from;
		const isUnread = message.from !== me && viewed !== partner;
		const previous = conversations.find((conversation) => conversation.username === partner);
		if (previous?.last.id >= message.id) {
			return;
		}
		conversations = [
			{ username: partner, last: message, unread: (previous?.unread ?? 0) + (isUnread ? 1 : 0) },
			...conversations.filter((conversation) => conversation !== previous),
		];

		if (loaded && partner === active) {
			const follow = nearBottom() || message.from === me;
			privates = merge(privates, [message])[1];
			if (follow) {
				scrollToBottom();
			}
		}
	}

	function markRead(username) {
		conversations = conversations.map((conversation) =>
			conversation.username === username ? { ...conversation, unread: 0 } : conversation,
		);
	}

	function showConversation(event) {
		if (!active || !sameUser(event.with, active)) {
			return;
		}
		// Typed with another case
		active = event.with;
		privates = merge([], event.privates ?? [])[1];
		loaded = true;
		markRead(event.with);
		scrollToBottom();
	}

	function settle() {
		clearTimeout(pendingTimer);
		const [request, ...rest] = waiting;
		waiting = rest;
		return request;
	}

	function expect(kind, key) {
		waiting = [...waiting, { kind, key }];
		clearTimeout(pendingTimer);
		pendingTimer = setTimeout(() => {
			waiting = [];
		}, PENDING_TIMEOUT_MS);
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

	function handleError(event) {
		const request = settle();
		error = event.error;
		if (event.error === 'rate-limited') {
			rateLimited(event.retryAfter ?? 2000);
		}
		// Nobody to talk to, back to the list
		if (['user-not-found', 'self'].includes(event.error) && request?.kind !== 'ignore' && !loaded) {
			active = '';
		}
	}

	function handleEvent(event) {
		switch (event.type) {
			case 'history': {
				locale = event.locale;
				ignored = event.ignored ?? [];
				for (const conversation of (event.conversations ?? []).toReversed()) {
					addPrivate(conversation.last);
					const added = conversations.find((current) => current.username === conversation.username);
					added.unread = conversation.unread;
				}
				addMessages(event.messages ?? []);
				break;
			}
			case 'message': {
				addMessages([event.message]);
				break;
			}
			case 'private': {
				addPrivate(event.private);
				break;
			}
			case 'conversation': {
				if (waiting[0]?.kind === 'view') {
					settle();
				}
				showConversation(event);
				break;
			}
			case 'read': {
				markRead(event.with);
				break;
			}
			case 'ignored': {
				ignored = event.ignored ?? [];
				break;
			}
			case 'ack': {
				const request = settle();
				if (request?.kind === 'send') {
					drafts[request.key] = '';
				}
				error = '';
				break;
			}
			case 'error': {
				handleError(event);
				break;
			}
		}
	}

	function handleStatus(next, reason = '') {
		status = next;
		closedReason = reason;
		// Answers won't come on a new connection
		if (next !== 'open') {
			clearTimeout(pendingTimer);
			waiting = [];
			sentView = undefined;
		}
	}

	function send(event) {
		event.preventDefault();
		if (!canSend || !connection.send(draft, tab === 'shout' ? undefined : active)) {
			return;
		}
		expect('send', composerKey);
		error = '';
	}

	function handleInputKeydown(event) {
		// Enter sends, shift+enter is ignored as messages are a single line
		if (event.key === 'Enter') {
			send(event);
		}
	}

	function ignore(username, duration) {
		if (connection.ignore(username, duration)) {
			expect('ignore');
		}
	}

	function unignore(username) {
		if (connection.unignore(username)) {
			expect('ignore');
		}
	}

	async function showTab(next) {
		tab = next;
		error = '';
		if (next === 'shout') {
			shoutUnread = 0;
		}
		await scrollToBottom();
		input?.focus();
	}

	async function openConversation(username) {
		const known = conversations.find((conversation) => sameUser(conversation.username, username));
		const next = known?.username ?? username;
		// Shown already, or loaded as soon as the tab is
		if (next !== active) {
			active = next;
			privates = [];
			loaded = false;
		}
		await showTab('messages');
	}

	function back() {
		active = '';
		error = '';
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

	async function show() {
		open = true;
		compact = Boolean(root?.closest('header.is-compact'));
		if (compact) {
			const navToggle = document.querySelector('#nav-toggle');
			if (navToggle?.checked) {
				navToggle.checked = false;
				navToggle.dispatchEvent(new Event('change'));
			}
		}
		if (tab === 'shout') {
			shoutUnread = 0;
		}
		await scrollToBottom();
		input?.focus();
	}

	function toggle() {
		if (open) {
			open = false;
			return;
		}
		// Straight to the unread messages
		if (privateUnread > 0 && shoutUnread === 0 && !active) {
			tab = 'messages';
		}
		show();
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
		'user-not-found': dataset.labelErrorUserNotFound,
		self: dataset.labelErrorSelf,
		blocked: dataset.labelErrorBlocked,
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

	// Tell the server which conversation is on screen, again after reconnecting
	$effect(() => {
		const target = viewed;
		if (status !== 'open' || target === (sentView ?? '')) {
			return;
		}
		if (connection.view(target)) {
			sentView = target;
			if (target) {
				expect('view');
			}
		}
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

		// Links to a conversation on the page, such as on profiles, open it here instead of reloading
		function handleChatLink(event) {
			const link = event.target.closest?.('[data-chat-open]');
			if (!link || event.button !== 0 || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) {
				return;
			}
			event.preventDefault();
			openConversation(link.dataset.chatOpen);
			show();
		}

		function handleVisibility() {
			visible = document.visibilityState === 'visible';
		}

		document.addEventListener('pointerdown', handleOutsidePointer);
		document.addEventListener('keydown', handleKeydown);
		document.addEventListener('visibilitychange', handleVisibility);
		document.addEventListener('click', handleChatLink);
		// Connected right away, so private messages arrive live
		connection = new ShoutboxConnection({ onEvent: handleEvent, onStatus: handleStatus });

		const url = new URL(location.href);
		const chat = url.searchParams.get(CHAT_PARAM);
		if (chat) {
			url.searchParams.delete(CHAT_PARAM);
			history.replaceState(history.state, '', url);
			openConversation(chat);
			show();
		}

		return () => {
			document.removeEventListener('pointerdown', handleOutsidePointer);
			document.removeEventListener('keydown', handleKeydown);
			document.removeEventListener('visibilitychange', handleVisibility);
			document.removeEventListener('click', handleChatLink);
			clearTimeout(pendingTimer);
			clearTimeout(limitTimer);
			connection.close();
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
				<div class="shout-tabs" role="tablist">
					<button
						type="button"
						role="tab"
						class="shout-tab"
						aria-selected={tab === 'shout'}
						onclick={() => showTab('shout')}
					>
						{dataset.labelTabShout}
						{#if shoutUnread > 0}
							<span class="shout-tab-badge">{shoutUnread > 9 ? '9+' : shoutUnread}</span>
						{/if}
					</button>
					<button
						type="button"
						role="tab"
						class="shout-tab"
						aria-selected={tab === 'messages'}
						onclick={() => showTab('messages')}
					>
						{dataset.labelTabMessages}
						{#if privateUnread > 0}
							<span class="shout-tab-badge is-private">{privateUnread > 9 ? '9+' : privateUnread}</span>
						{/if}
					</button>
				</div>
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

			{#if tab === 'shout'}
				<ol class="shout-list" bind:this={list} data-chat-scroll aria-live="polite" aria-relevant="additions" aria-busy={status !== 'open'}>
					{#each shown as message (message.id)}
						<li class="shout-message" class:is-self={message.username === me}>
							<span class="shout-meta">
								{#if message.username === me}
									<a class="shout-author" href="/profile/{encodeURIComponent(message.username)}">{message.username}</a>
								{:else}
									<ChatUserMenu
										username={message.username}
										labels={dataset}
										ignored={ignoring.has(message.username)}
										onMessage={() => openConversation(message.username)}
										onIgnore={(duration) => ignore(message.username, duration)}
										onUnignore={() => unignore(message.username)}
									/>
								{/if}
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
			{:else if active}
				<div class="shout-conversation-head">
					<button type="button" class="shout-back" aria-label={dataset.labelBack} title={dataset.labelBack} onclick={back}>
						<svg width="18" height="18" viewBox="0 0 24 24" aria-hidden="true">
							<path d="M15 6l-6 6 6 6" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" fill="none" />
						</svg>
					</button>
					<ChatUserMenu
						username={active}
						labels={dataset}
						ignored={ignoring.has(active)}
						onIgnore={(duration) => ignore(active, duration)}
						onUnignore={() => unignore(active)}
					/>
				</div>
				{#if ignoring.has(active)}
					<p class="shout-notice">{dataset.labelIgnoringNotice}</p>
				{/if}
				<ol class="shout-list" bind:this={list} data-chat-scroll aria-live="polite" aria-relevant="additions" aria-busy={!loaded}>
					{#each privates as message (message.id)}
						<li class="shout-bubble" class:is-self={message.from === me}>
							<span class="shout-body">{message.body}</span>
							<time datetime={message.createdAt} title={fullDate(message.createdAt, locale)}>
								{ago(message.createdAt, now)}
							</time>
						</li>
					{:else}
						<li class="shout-empty">{loaded ? dataset.labelConversationEmpty : dataset.labelConnecting}</li>
					{/each}
				</ol>
			{:else}
				<ChatConversations
					{conversations}
					ignored={ignored.filter((user) => ignoring.has(user.username))}
					username={me}
					{locale}
					{now}
					labels={dataset}
					onOpen={openConversation}
					onUnignore={unignore}
				/>
			{/if}

			{#if status === 'closed'}
				<p class="section-error shout-alert" role="alert">{closedLabels[closedReason] ?? dataset.labelClosedError}</p>
			{:else if error}
				<p class="section-error shout-alert" role="alert">{errorLabels[error] ?? dataset.labelErrorInternal}</p>
			{/if}

			{#if composerKey}
				<form class="shout-form" onsubmit={send}>
					<input
						bind:this={input}
						bind:value={() => draft, (value) => (drafts[composerKey] = value)}
						type="text"
						class="shout-input"
						placeholder={tab === 'shout' ? dataset.labelPlaceholder : dataset.labelPrivatePlaceholder}
						aria-label={tab === 'shout' ? dataset.labelPlaceholder : dataset.labelPrivatePlaceholder}
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
			{/if}
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

	.shout-tabs {
		display: flex;
		gap: 0.25rem;
		min-width: 0;
	}

	.shout-tab {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;

		padding: 0.35rem 0.6rem;

		color: var(--color-text-muted);
		background: transparent;
		border-radius: 0.4rem;

		font-weight: 700;
		white-space: nowrap;

		&:hover {
			color: var(--color-text);
			background: var(--color-bg-3);
		}

		&[aria-selected='true'] {
			color: var(--color-text);
			background: var(--color-bg-4);
		}
	}

	.shout-tab-badge {
		min-width: 1.1rem;
		height: 1.1rem;
		padding: 0 0.25rem;

		color: var(--color-text);
		background: var(--color-border-2);
		border-radius: 999px;

		font-size: 0.65rem;
		line-height: 1.1rem;
		font-variant-numeric: tabular-nums;
		text-align: center;

		&.is-private {
			color: var(--color-text-alt);
			background: var(--color-accent-secondary);
		}
	}

	.shout-status {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		margin-inline: auto 0;

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

	/* The tabs need the room, the dot tells */
	.shout-status-text {
		position: absolute;
		overflow: hidden;
		width: 1px;
		height: 1px;
		clip-path: inset(50%);
		white-space: nowrap;
	}

	.shout-conversation-head {
		display: flex;
		align-items: center;
		gap: 0.25rem;

		padding: 0.35rem 0.5rem;
		border-bottom: 1px solid var(--color-border-1);
	}

	.shout-back {
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

	.shout-notice {
		margin: 0;
		padding: 0.4rem 0.75rem;

		color: var(--color-text-muted);
		background: var(--color-bg-3);
		border-bottom: 1px solid var(--color-border-1);

		font-size: 0.8rem;
	}

	.shout-list {
		flex: 1;
		min-height: 0;
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

	.shout-bubble {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		align-self: flex-start;

		max-width: 85%;
		margin-block: 0.1rem;
		padding: 0.35rem 0.65rem;

		background: var(--color-bg-3);
		border-radius: 0.75rem 0.75rem 0.75rem 0.25rem;

		time {
			color: var(--color-text-muted);
			font-size: 0.7rem;
			font-variant-numeric: tabular-nums;
		}

		&.is-self {
			align-self: flex-end;
			align-items: flex-end;

			background: color-mix(in srgb, var(--color-accent-primary), var(--color-bg-2) 80%);
			border-radius: 0.75rem 0.75rem 0.25rem;
		}
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

	/* Stretched, the input is as tall as the button */
	.shout-form {
		display: flex;
		align-items: stretch;
		gap: 0.4rem;

		margin: 0;
		padding: 0.5rem;
		border-top: 1px solid var(--color-border-1);
	}

	.shout-input {
		flex: 1;
		min-width: 0;
		padding-inline: 0.75rem;

		&[aria-invalid='true'] {
			border-color: var(--color-error-3);
		}
	}

	.shout-count {
		align-self: center;

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
