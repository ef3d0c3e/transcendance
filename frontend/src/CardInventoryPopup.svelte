<script>
	import { onMount, tick } from "svelte";

	let { target } = $props();

	let open = $state(false);

	let title = $state("");
	let acquired = $state("");
	let collectionTitle = $state("");
	let collectionUrl = $state("");
	let labelCollection = $state("");
	let labelClose = $state("");

	let popup;

	onMount(() => {
		if (target) {
			labelCollection = target.dataset.labelCollection ?? "";
			labelClose = target.dataset.labelClose ?? "";
		}

		// Inventory cards carry their acquisition info in data attributes;
		// clicking one opens this dialog instead of printing it on the face.
		function handleClick(event) {
			const card = event.target.closest?.(".card-container[data-acquired]");
			if (!card) {
				return;
			}

			// In-card links (e.g. the footer collection link) keep working
			if (event.target.closest("a")) {
				return;
			}

			title = card.dataset.cardTitle ?? "";
			acquired = card.dataset.acquired ?? "";
			collectionTitle = card.dataset.collectionTitle ?? "";
			collectionUrl = card.dataset.collectionUrl ?? "";

			open = true;

			tick().then(() => popup?.focus());
		}

		document.addEventListener("click", handleClick);

		return () => {
			document.removeEventListener("click", handleClick);
		};
	});

	function closepopup() {
		open = false;
	}

	function handleBackdropClick(event) {
		if (event.target === event.currentTarget) {
			closepopup();
		}
	}

	function handleKeydown(event) {
		if (event.key === "Escape" && open) {
			event.preventDefault();
			closepopup();
		}
	}
</script>

<div
	class="backdrop"
	class:open
	role="presentation"
	aria-hidden={!open}
	onclick={handleBackdropClick}
>
	<div
		class="popup"
		role="dialog"
		aria-modal="true"
		aria-label={title}
		tabindex="-1"
		bind:this={popup}
	>
		<h2 class="popup-title">{title}</h2>
		<p class="popup-acquired">{acquired}</p>
		<a class="popup-collection" href={collectionUrl}>
			{labelCollection}
			<span class="popup-collection-name">{collectionTitle}</span>
		</a>
		<button type="button" class="popup-close" onclick={closepopup}>
			{labelClose}
		</button>
	</div>
</div>

<svelte:window onkeydown={handleKeydown} />

<style>
	.backdrop {
		position: fixed;
		inset: 0;
		z-index: 1000;

		display: flex;
		align-items: center;
		justify-content: center;

		/* Respect notches */
		padding: max(1rem, env(safe-area-inset-top))
			max(1rem, env(safe-area-inset-right))
			max(1rem, env(safe-area-inset-bottom))
			max(1rem, env(safe-area-inset-left));

		background: rgb(0 0 0 / 55%);
		backdrop-filter: blur(3px);

		visibility: hidden;
		opacity: 0;
		pointer-events: none;

		transition:
			opacity 150ms ease,
			visibility 150ms ease;
	}

	.backdrop.open {
		visibility: visible;
		opacity: 1;
		pointer-events: auto;
	}

	.popup {
		position: relative;

		display: flex;
		flex-direction: column;
		gap: 0.75rem;

		margin: 1rem;
		width: min(100%, 24rem);

		max-height: 100dvh;

		padding: 1.25rem;

		overflow: auto;

		overscroll-behavior: contain;

		background: var(--color-bg-2);
		color: var(--color-text);
		border: 1px solid var(--color-border-1);
		border-radius: 0.75rem;

		/* Makes focus outlines easier to see */
		outline: none;

		transform: translateY(0) scale(1);

		transition:
			transform 150ms ease,
			opacity 150ms ease;
	}

	.backdrop:not(.open) .popup {
		transform: translateY(0.5rem) scale(0.98);
		opacity: 0;
	}

	.backdrop.open .popup {
		transform: translateY(0) scale(1);
		opacity: 1;
	}

	.popup-title {
		margin: 0;

		font-size: 1.15rem;
		line-height: 1.3;
	}

	.popup-acquired {
		margin: 0;

		color: var(--color-text-muted);
	}

	.popup-collection {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;

		color: var(--color-accent-primary);
		text-decoration: none;
	}

	.popup-collection:hover {
		text-decoration: underline;
	}

	.popup-collection-name {
		color: var(--color-text-muted);
	}

	.popup-close {
		align-self: flex-start;

		padding: 0.5rem 1rem;

		color: var(--color-text);
		background: var(--color-bg-4);
		border: 1px solid var(--color-border-1);
		border-radius: 0.5rem;

		cursor: pointer;
	}

	.popup-close:hover {
		background: var(--color-bg-3);
	}

	/* Small screens */
	@media (max-width: 480px) {
		.backdrop {
			align-items: flex-end;
			padding: 0;
		}

		.popup {
			width: 100%;
			max-height: min(90dvh, 42rem);

			border-radius: 0.75rem 0.75rem 0 0;
		}

		.backdrop:not(.open) .popup {
			transform: translateY(1rem);
		}
	}

	/* Reduced motion */
	@media (prefers-reduced-motion: reduce) {
		.backdrop,
		.popup {
			transition: none;
		}
	}
</style>
