<!--
    Popup with the card a tradeup just awarded, opened as soon as the page
    loads. The card markup is server-rendered into a <template> on the mount
    target and revealed here with a scale-up, like the pack showcase.
-->

<script>
	import { onMount } from "svelte";

	let { target } = $props();

	const labelTitle = $derived(target.dataset.labelTitle ?? "");
	const labelClose = $derived(target.dataset.labelClose ?? "");
	const cardHtml = $derived(target.querySelector("template")?.innerHTML ?? "");

	let dialog;

	// Clicking the backdrop targets the dialog itself
	function onDialogClick(e) {
		if (e.target === dialog) dialog.close();
	}

	onMount(() => {
		dialog.showModal();
	});
</script>

<dialog
	class="popup tradeup-result"
	aria-labelledby="tradeup-result-title"
	bind:this={dialog}
	onclick={onDialogClick}
>
	<div class="tradeup-result-content">
		<h2 class="tradeup-result-title" id="tradeup-result-title">{labelTitle}</h2>
		<div class="tradeup-result-stage">
			<div class="tradeup-result-flash" aria-hidden="true"></div>
			<div class="tradeup-result-card">
				<!-- The server-rendered card of the page's <template> -->
				<!-- eslint-disable-next-line svelte/no-at-html-tags -->
				<div class="tradeup-result-inner">{@html cardHtml}</div>
			</div>
		</div>
		<button type="button" class="tradeup-result-close" onclick={() => dialog.close()}>
			{labelClose}
		</button>
	</div>
</dialog>

<style>
	/* Frame, backdrop, entrance and bottom sheet: the shared .popup of theme.css */
	.tradeup-result {
		--popup-width: 26rem;
	}

	.tradeup-result-content {
		display: flex;
		flex-direction: column;
		gap: 1.25rem;

		padding: 1.25rem;
	}

	.tradeup-result-title {
		margin: 0;

		font-size: 1.25rem;
		font-weight: 650;
		letter-spacing: -0.01em;
		text-align: center;
	}

	.tradeup-result-stage {
		--tradeup-scale: 0.8;

		position: relative;

		width: calc(30ch * var(--tradeup-scale));
		height: calc(30em * var(--tradeup-scale));
		margin: 0 auto;
	}

	/* Glow that bursts out from behind the card as it lands */
	.tradeup-result-flash {
		position: absolute;
		inset: -6%;

		background: radial-gradient(
			circle,
			var(--color-accent-primary) 0%,
			transparent 70%
		);
		border-radius: 50%;

		pointer-events: none;
		animation: tradeup-flash 1s ease-out both;
	}

	.tradeup-result-card {
		width: 100%;
		height: 100%;

		animation: tradeup-reveal 0.7s cubic-bezier(0.34, 1.4, 0.64, 1) both;
	}

	.tradeup-result-inner {
		width: 30ch;
		height: 30em;

		transform: scale(var(--tradeup-scale));
		transform-origin: top left;
	}

	.tradeup-result-close {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		align-self: flex-end;

		min-width: 5.5rem;
		min-height: 2.5rem;
		padding: 0.45rem 1rem;

		color: var(--color-text);
		background: var(--color-bg-4);
		border: 0;
		border-radius: 0.55rem;

		font-weight: 600;
		line-height: 1.3;

		transition: background-color 120ms ease;
	}

	.tradeup-result-close:hover {
		background: color-mix(in srgb, var(--color-bg-4), black 10%);
	}

	@keyframes tradeup-reveal {
		0% {
			opacity: 0;
			transform: scale(0.7);
		}

		100% {
			opacity: 1;
			transform: scale(1);
		}
	}

	@keyframes tradeup-flash {
		0% {
			opacity: 0.9;
			transform: scale(0.5);
		}

		100% {
			opacity: 0;
			transform: scale(1.6);
		}
	}

	@media (width <= 1100px) {
		.tradeup-result-stage {
			--tradeup-scale: 0.68;
		}
	}

	@media (width <= 820px) {
		.tradeup-result-stage {
			--tradeup-scale: 0.54;
		}
	}

	@media (width <= 560px) {
		.tradeup-result-stage {
			--tradeup-scale: 0.42;
		}
	}

	@media (max-width: 480px) {
		.tradeup-result-content {
			padding-bottom: max(1.25rem, env(safe-area-inset-bottom));
		}

		.tradeup-result-close {
			align-self: stretch;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.tradeup-result-card,
		.tradeup-result-flash {
			animation: none;
		}

		/* Without the burst, the glow would just sit there */
		.tradeup-result-flash {
			display: none;
		}
	}
</style>
