<script>
	import { apiFetch } from './api.js';

	// How long an error stays shown, in milliseconds
	const ERROR_DELAY = 4000;

	let { dataset } = $props();

	// Relation to the user, rendered by the server then given back by every action
	let relation = $derived(dataset.relation ?? '');
	let busy = $state(false);
	let error = $state('');

	// Actions available for each relation
	const ACTIONS = {
		'': ['add'],
		pending: ['cancel'],
		respond: ['accept', 'deny'],
		friends: ['unfriend'],
		blocked: ['unblock'],
		// Blocked by the other user, adding them explains why it can't be done
		block: ['add'],
	};
	// Dataset key of each action's label
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

	$effect(() => {
		if (!error) {
			return;
		}
		const timeout = setTimeout(() => (error = ''), ERROR_DELAY);
		return () => clearTimeout(timeout);
	});

	async function run(action) {
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
	}
</script>

<div class="friend-actions">
	{#each actions as action (action)}
		<button
			type="button"
			class="friend-action"
			class:is-main={action === 'add' || action === 'accept'}
			class:is-danger={action === 'block' || action === 'unfriend'}
			disabled={busy}
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

		&:hover:not(:disabled) {
			background: var(--color-bg-3);
		}

		&.is-main {
			color: var(--color-text-alt);
			background: var(--color-accent-primary);
			border-color: transparent;

			&:hover:not(:disabled) {
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
