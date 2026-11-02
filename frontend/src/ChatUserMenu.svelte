<script>
	import { onMount } from 'svelte';

	// `onMessage` is left out where a conversation with the user is already shown
	let { username, labels, ignored, onMessage, onIgnore, onUnignore, class: className = '' } = $props();

	const DURATIONS = [
		['hour', 'labelIgnoreHour'],
		['day', 'labelIgnoreDay'],
		['week', 'labelIgnoreWeek'],
	];

	// Room the menu needs below its trigger, in pixels
	const MENU_HEIGHT = 180;

	let open = $state(false);
	// Opens upwards when there is no room below, in the scrolled list or the window
	let up = $state(false);
	let root = $state(undefined);
	let trigger = $state(undefined);

	function toggle() {
		open = !open;
		if (!open || !trigger) {
			return;
		}
		const bottom = trigger.closest('[data-chat-scroll]')?.getBoundingClientRect().bottom ?? window.innerHeight;
		up = trigger.getBoundingClientRect().bottom + MENU_HEIGHT > bottom;
	}

	function pick(action) {
		open = false;
		action();
	}

	function handleKeydown(event) {
		if (event.key === 'Escape' && open) {
			// Only the menu closes, not the panel
			event.stopPropagation();
			open = false;
			trigger?.focus();
		}
	}

	onMount(() => {
		function handleOutsidePointer(event) {
			if (open && !root?.contains(event.target)) {
				open = false;
			}
		}
		document.addEventListener('pointerdown', handleOutsidePointer);
		return () => document.removeEventListener('pointerdown', handleOutsidePointer);
	});
</script>

<span class="chat-user" bind:this={root} onkeydowncapture={handleKeydown}>
	<button
		bind:this={trigger}
		type="button"
		class="chat-user-trigger {className}"
		aria-haspopup="menu"
		aria-expanded={open}
		title={labels.labelUserMenu}
		onclick={toggle}
	>
		{username}
	</button>

	{#if open}
		<span class="chat-user-menu" class:is-up={up} role="menu" aria-label={username}>
			{#if onMessage}
				<button type="button" role="menuitem" onclick={() => pick(onMessage)}>{labels.labelMessageUser}</button>
			{/if}
			<a role="menuitem" href="/profile/{encodeURIComponent(username)}">{labels.labelViewProfile}</a>
			{#if ignored}
				<button type="button" role="menuitem" onclick={() => pick(onUnignore)}>{labels.labelUnignore}</button>
			{:else}
				{#each DURATIONS as [duration, label] (duration)}
					<button type="button" role="menuitem" class="is-danger" onclick={() => pick(() => onIgnore(duration))}>
						{labels[label]}
					</button>
				{/each}
			{/if}
		</span>
	{/if}
</span>

<style>
	.chat-user {
		position: relative;
		display: inline-flex;
		min-width: 0;
	}

	.chat-user-trigger {
		overflow: hidden;
		min-width: 0;
		padding: 0;

		color: var(--color-accent-primary);
		background: none;

		font-size: 0.85rem;
		font-weight: 700;
		text-overflow: ellipsis;
		white-space: nowrap;

		&:hover,
		&[aria-expanded='true'] {
			text-decoration: underline;
		}
	}

	.chat-user-menu {
		position: absolute;
		z-index: 2;
		top: calc(100% + 0.25rem);
		left: 0;

		display: flex;
		flex-direction: column;
		min-width: 11rem;
		padding: 0.25rem;

		background: var(--color-bg-1);
		border: 1px solid var(--color-border-1);
		border-radius: 0.5rem;
		box-shadow: 0 8px 20px rgb(0 0 0 / 18%);

		button,
		a {
			padding: 0.4rem 0.6rem;

			color: var(--color-text);
			background: none;
			border-radius: 0.35rem;

			font-size: 0.85rem;
			font-weight: 500;
			text-align: start;
			text-decoration: none;
			white-space: nowrap;

			&:hover,
			&:focus-visible {
				background: var(--color-bg-3);
			}
		}

		.is-danger {
			color: var(--color-error-3);
		}

		&.is-up {
			top: auto;
			bottom: calc(100% + 0.25rem);
		}
	}
</style>
