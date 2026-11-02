<script>
	import { tick } from 'svelte';
	import { apiFetch } from './api.js';

	let { dataset } = $props();

	// Relation to the user, rendered by the server then given back by every action
	let relation = $derived(dataset.relation ?? '');
	let busy = $state(false);
	// Stays until the next action, so it can be read at any pace
	let error = $state('');
	let actionsElement = $state();

	const ACTIONS = {
		'': ['add'],
		pending: ['cancel'],
		respond: ['accept', 'deny'],
		friends: ['unfriend'],
		blocked: ['unblock'],
		// Blocked by the other user, adding them explains why it can't be done
		block: ['add'],
	};
	const LABELS = {
		add: 'labelAdd',
		cancel: 'labelCancel',
		accept: 'labelAccept',
		deny: 'labelDeny',
		unfriend: 'labelUnfriend',
		block: 'labelBlock',
		unblock: 'labelUnblock',
	};

	let actions = $derived.by(() => {
		const list = [...(ACTIONS[relation] ?? ACTIONS[''])];
		if (dataset.block !== undefined && relation !== 'blocked') {
			list.push('block');
		}
		return list;
	});

	// The clicked button is replaced by the new relation's: the focus goes to
	// the first of them, whose name tells what changed
	async function run(action) {
		if (busy) {
			return;
		}
		busy = true;
		error = '';
		try {
			const params = new URLSearchParams({ target: dataset.id, action });
			const response = await apiFetch(`/friendaction?${params}`, { method: 'POST' });
			const data = await response.json();
			if (typeof data.relation === 'string') {
				relation = data.relation;
			}
			if (!response.ok) {
				error = data.message ?? dataset.labelError;
			}
		} catch {
			error = dataset.labelError;
		} finally {
			busy = false;
		}
		await tick();
		if (actionsElement && !actionsElement.contains(document.activeElement)) {
			actionsElement.querySelector('button')?.focus();
		}
	}
</script>

<!-- aria-disabled while waiting: disabled would drop the focus -->
<div class="friend-actions" bind:this={actionsElement}>
	{#each actions as action (action)}
		<button
			type="button"
			class="friend-action"
			class:is-main={action === 'add' || action === 'accept'}
			class:is-danger={action === 'block' || action === 'unfriend'}
			aria-disabled={busy}
			onclick={() => run(action)}
		>
			{dataset[LABELS[action]]}
		</button>
	{/each}
</div>
{#if error}
	<p class="friend-error" role="alert">{error}</p>
{/if}

<style>
	.friend-actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: inherit;
		gap: 0.4rem;
	}

	.friend-action {
		padding: 0.35rem 0.75rem;

		color: var(--color-text);
		background: var(--color-bg-4);
		border: 1px solid var(--color-border-1);
		border-radius: 0.55rem;

		font-size: 0.85rem;
		font-weight: 600;
		line-height: 1.3;

		transition: background-color 120ms ease;

		&:hover:not([aria-disabled='true']) {
			background: var(--color-bg-3);
		}

		&.is-main {
			color: var(--color-text-alt);
			background: var(--color-accent-primary);
			border-color: transparent;

			&:hover:not([aria-disabled='true']) {
				background: var(--color-accent-primary-dark);
			}
		}

		&.is-danger {
			color: var(--color-error-3);
		}
	}

	.friend-error {
		margin: 0.4rem 0 0;

		color: var(--color-error-3);
		font-size: 0.8rem;
	}
</style>
