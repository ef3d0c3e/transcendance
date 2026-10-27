<script>
	import { onMount } from 'svelte';
	import Bell from '../icons/Bell.svelte';
	import { relativeTime, fullDate } from './relative-time.js';

	let { dataset } = $props();

	// Reload this often while the page is visible and the popup closed
	const REFRESH_MS = 60_000;

	let notifications = $state([]);
	let unread = $state(0);
	let locale = $state(undefined);
	// Hidden for logged out users
	let authenticated = $state(true);
	let loaded = $state(false);
	let loading = $state(false);
	let error = $state(false);
	let open = $state(false);

	let root = $state(undefined);
	let trigger = $state(undefined);

	async function load() {
		if (loading) {
			return;
		}
		loading = true;
		try {
			const response = await fetch('/api/notifications?limit=5', { credentials: 'include' });
			if (response.status === 401) {
				authenticated = false;
				return;
			}
			if (!response.ok) {
				throw new Error(`HTTP ${response.status}`);
			}
			const data = await response.json();
			notifications = data.notifications ?? [];
			unread = data.unread ?? 0;
			locale = data.locale;
			error = false;
			loaded = true;
		} catch (err) {
			console.error('Failed to load notifications:', err);
			error = true;
		} finally {
			loading = false;
		}
	}

	async function markAllRead() {
		const response = await fetch('/api/notifications/read_all', { method: 'POST', credentials: 'include' });
		if (response.ok) {
			for (const notification of notifications) {
				notification.Read = true;
			}
			unread = 0;
		}
	}

	function toggle() {
		// No room for the popup in the compact header's side menu, use the page
		if (root?.closest('header.is-compact')) {
			globalThis.location.href = '/notifications';
			return;
		}
		open = !open;
		if (open) {
			load();
		}
	}

	onMount(() => {
		load();

		const interval = setInterval(() => {
			if (!open && document.visibilityState === 'visible') {
				load();
			}
		}, REFRESH_MS);

		function handleVisibility() {
			if (document.visibilityState === 'visible') {
				load();
			}
		}

		function handleOutsidePointer(event) {
			if (open && root && !root.contains(event.target)) {
				open = false;
			}
		}

		function handleKeydown(event) {
			if (event.key === 'Escape' && open) {
				open = false;
				trigger?.focus();
			}
		}

		document.addEventListener('visibilitychange', handleVisibility);
		document.addEventListener('pointerdown', handleOutsidePointer);
		document.addEventListener('keydown', handleKeydown);
		return () => {
			clearInterval(interval);
			document.removeEventListener('visibilitychange', handleVisibility);
			document.removeEventListener('pointerdown', handleOutsidePointer);
			document.removeEventListener('keydown', handleKeydown);
		};
	});
</script>

{#if authenticated}
	<div class="notif-root" bind:this={root}>
		<button
			bind:this={trigger}
			type="button"
			class="notif-bell"
			aria-label={unread > 0 ? `${dataset.labelTitle}, ${unread} ${dataset.labelUnread}` : dataset.labelTitle}
			aria-haspopup="true"
			aria-expanded={open}
			aria-controls="notification-panel"
			onclick={toggle}
		>
			<Bell />
			<span class="notif-bell-label">{dataset.labelTitle}</span>
			{#if unread > 0}
				<span class="notif-badge" aria-hidden="true">{unread > 9 ? '9+' : unread}</span>
			{/if}
		</button>

		{#if open}
			<div id="notification-panel" class="notif-panel" role="dialog" aria-label={dataset.labelTitle}>
				<div class="notif-panel-head">
					<span class="notif-panel-title">{dataset.labelTitle}</span>
					{#if unread > 0}
						<button type="button" class="notification-text-button" onclick={markAllRead}>
							{dataset.labelMarkAllRead}
						</button>
					{/if}
				</div>

				<div class="notif-panel-body" aria-busy={loading}>
					{#if !loaded && !error}
						<ul class="notification-list" aria-label={dataset.labelLoading}>
							{#each [0, 1, 2] as index (index)}
								<li class="notification-item notif-skeleton" aria-hidden="true">
									<span class="notification-icon"></span>
									<span class="notif-skeleton-lines"><span></span><span></span></span>
								</li>
							{/each}
						</ul>
					{:else if error && !loaded}
						<p class="section-error notif-error" role="alert">{dataset.labelError}</p>
					{:else if notifications.length === 0}
						<p class="notification-empty">{dataset.labelNone}</p>
					{:else}
						<ul class="notification-list">
							{#each notifications as notification (notification.ID)}
								<li class="notification-item" class:is-unread={!notification.Read}>
									<span class="notification-icon" data-type={notification.Type} aria-hidden="true"></span>
									<a class="notification-body" href="/notification/{notification.ID}">
										<span class="notification-title">{notification.Title}</span>
										<span class="notification-desc">{notification.Desc}</span>
										<time
											class="notification-time"
											datetime={notification.CreatedAt}
											title={fullDate(notification.CreatedAt, locale)}
										>
											{relativeTime(notification.CreatedAt, locale)}
										</time>
									</a>
								</li>
							{/each}
						</ul>
					{/if}
				</div>

				<a class="notif-panel-footer" href="/notifications">{dataset.labelViewAll}</a>
			</div>
		{/if}
	</div>
{/if}

<style>
	.notif-root {
		position: relative;
		display: inline-flex;
	}

	/* Neutral button, like the language switcher */
	.notif-bell {
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

	.notif-badge {
		position: absolute;
		top: -0.35rem;
		right: -0.35rem;

		min-width: 1.25rem;
		height: 1.25rem;
		padding: 0 0.3rem;

		color: var(--color-text-alt);
		background: var(--color-accent-primary);
		border: 2px solid var(--color-bg-2);
		border-radius: 999px;

		font-size: 0.7rem;
		font-weight: 700;
		line-height: calc(1.25rem - 4px);
		font-variant-numeric: tabular-nums;
		text-align: center;
	}

	.notif-panel {
		position: absolute;
		z-index: 1000;
		top: calc(100% + 0.5rem);
		right: 0;

		display: flex;
		flex-direction: column;

		width: min(24rem, calc(100vw - 2rem));
		max-height: min(32rem, calc(100dvh - 6rem));

		color: var(--color-text);
		background: var(--color-bg-2);
		border: 1px solid var(--color-border-1);
		border-radius: 0.75rem;
		box-shadow: 0 12px 32px rgb(0 0 0 / 18%);

		animation: notif-panel-in 140ms ease;
	}

	/* Only labelled in the compact header's side menu */
	.notif-bell-label {
		display: none;
	}

	:global(header.is-compact) .notif-root {
		flex-direction: column;
		align-items: stretch;
	}

	:global(header.is-compact) .notif-bell {
		gap: 0.5rem;
		padding: 0.45rem 0.9rem;
		font-weight: 600;

		.notif-badge {
			position: static;
		}

		.notif-bell-label {
			display: inline;
		}
	}

	@keyframes notif-panel-in {
		from {
			opacity: 0;
			transform: translateY(-4px);
		}
	}

	.notif-panel-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;

		padding: 0.6rem 0.6rem 0.6rem 1rem;
		border-bottom: 1px solid var(--color-border-1);
	}

	.notif-panel-title {
		font-weight: 700;
		white-space: nowrap;
	}

	.notif-panel-body {
		flex: 1;
		overflow-y: auto;
		padding: 0.35rem;

		scrollbar-width: thin;
		scrollbar-color: var(--color-border-2) transparent;
	}

	.notif-panel-footer {
		padding: 0.7rem 1rem;

		border-top: 1px solid var(--color-border-1);
		border-radius: 0 0 0.75rem 0.75rem;

		font-weight: 600;
		text-align: center;
		text-decoration: none;

		&:hover {
			background: var(--color-bg-3);
		}
	}

	.notif-error {
		margin: 0.5rem;
	}

	.notif-skeleton-lines {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		padding-top: 0.3rem;

		span {
			height: 0.7rem;
			border-radius: 999px;
			background: var(--color-bg-4);
			animation: notif-pulse 1.2s ease-in-out infinite alternate;
		}

		span:last-child {
			width: 70%;
		}
	}

	.notif-skeleton .notification-icon {
		animation: notif-pulse 1.2s ease-in-out infinite alternate;
	}

	@keyframes notif-pulse {
		to {
			opacity: 0.45;
		}
	}
</style>
