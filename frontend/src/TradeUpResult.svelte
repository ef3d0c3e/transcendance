<!--
    Popup with the card a tradeup just awarded, opened as soon as the page
    loads. The card markup is server-rendered into a <template> on the mount
    target and revealed here with a scale-up, like the pack showcase.

    Also tells the outcome of a foil roll. With `data-outcome="lost"`, the
    card was destroyed: it shows faded out, without the burst.
    `data-label-text` adds a line under the card.
-->

<script>
	import { onMount } from "svelte";

	let { target } = $props();

	const labelTitle = $derived(target.dataset.labelTitle ?? "");
	const labelClose = $derived(target.dataset.labelClose ?? "");
	const labelText = $derived(target.dataset.labelText ?? "");
	const lost = $derived(target.dataset.outcome === "lost");
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
	class:is-lost={lost}
	aria-labelledby="tradeup-result-title"
	aria-describedby={labelText ? "tradeup-result-text" : undefined}
	bind:this={dialog}
	onclick={onDialogClick}
>
	<div class="tradeup-result-content">
		<h2 class="tradeup-result-title" id="tradeup-result-title">{labelTitle}</h2>
		<div class="tradeup-result-stage">
			{#if !lost}<div class="tradeup-result-flash" aria-hidden="true"></div>{/if}
			<div class="tradeup-result-card">
				<!-- The server-rendered card of the page's <template> -->
				<!-- eslint-disable-next-line svelte/no-at-html-tags -->
				<div class="tradeup-result-inner">{@html cardHtml}</div>
			</div>
		</div>
		{#if labelText}<p class="tradeup-result-text" id="tradeup-result-text">{labelText}</p>{/if}
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

	/* The glow is clipped here: past the popup it would only make it scroll */
	.tradeup-result-content {
		display: flex;
		flex-direction: column;
		gap: 1.25rem;

		padding: 1.25rem;
		overflow: clip;
	}

	.tradeup-result-title {
		margin: 0;

		font-size: 1.25rem;
		font-weight: 650;
		letter-spacing: -0.01em;
		text-align: center;
	}

	/* Takes the size of the scaled card, the glow sits behind it */
	.tradeup-result-stage {
		--tradeup-scale: 0.8;

		position: relative;

		align-self: center;
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

		/* Gone once it burst: no transform left to count in the popup's size */
		opacity: 0;

		pointer-events: none;
		animation: tradeup-flash 1s ease-out backwards;
	}

	.tradeup-result-card {
		position: relative;

		animation: tradeup-reveal 0.7s cubic-bezier(0.34, 1.4, 0.64, 1) both;
	}

	.tradeup-result-text {
		margin: 0;

		color: var(--color-text-muted);
		text-align: center;
	}

	/* A destroyed card fades away instead of landing */
	.is-lost .tradeup-result-card {
		filter: grayscale(1);
		opacity: 0.45;

		animation: tradeup-destroy 0.9s ease-out both;
	}

	@keyframes tradeup-destroy {
		0% {
			filter: grayscale(0);
			opacity: 1;
			transform: scale(1);
		}

		20% {
			transform: translateX(-4%) rotate(-2deg);
		}

		40% {
			transform: translateX(4%) rotate(2deg);
		}

		60% {
			transform: translateX(-2%) rotate(-1deg);
		}

		100% {
			filter: grayscale(1);
			opacity: 0.45;
			transform: scale(0.94);
		}
	}

	/* Zoomed, not transformed: the layout shrinks with the card, so the
	   popup fits it exactly whatever size the card renders at */
	.tradeup-result-inner {
		zoom: var(--tradeup-scale);
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

	/* The card is the tallest part: it shrinks with the screen height */
	@media (height <= 760px) {
		.tradeup-result-stage {
			--tradeup-scale: 0.65;
		}
	}

	@media (height <= 620px) {
		.tradeup-result-stage {
			--tradeup-scale: 0.5;
		}
	}

	@media (height <= 480px) {
		.tradeup-result-stage {
			--tradeup-scale: 0.38;
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
