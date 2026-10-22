<!--
    Popup to buy a booster when clicking one in the shop

    Boosters must be `.shop-booster` elements holding a `.shop-booster-display`
    (the collection tile and its price pill). The buy button stays disabled
    until pack opening lands (see TRANS-30).
-->

<script>
	import { onMount } from "svelte";

	let { target } = $props();

	const BOOSTER_SELECTOR = ".shop-booster";
	const DISPLAY_SELECTOR = ".shop-booster-display";
	// Pointer travel (px) after which a press counts as a tilt drag, not a click
	const DRAG_THRESHOLD = 6;

	let dialog;
	let preview;
	let title = $state("");
	let returnFocus = null;

	const labelBuy = $derived(target.dataset.labelBuy ?? "");
	const labelComingSoon = $derived(target.dataset.labelComingSoon ?? "");
	const labelBalance = $derived(target.dataset.labelBalance ?? "");
	const labelClose = $derived(target.dataset.labelClose ?? "");
	// Coin balance is only advertised when logged in
	const coins = $derived(target.dataset.coins);

	function show(booster) {
		title = booster.dataset.title ?? "";

		preview.replaceChildren();
		const display = booster.querySelector(DISPLAY_SELECTOR);
		if (display) preview.append(display.cloneNode(true));

		returnFocus = booster;
		dialog.showModal();
	}

	function close() {
		dialog.close();
	}

	function onDialogClose() {
		preview.replaceChildren();
		returnFocus?.focus();
		returnFocus = null;
	}

	// Clicking the backdrop targets the dialog itself
	function onDialogClick(e) {
		if (e.target === dialog) close();
	}

	onMount(() => {
		let press = null;

		function boosterFrom(e) {
			if (!(e.target instanceof Element)) return null;
			return e.target.closest(BOOSTER_SELECTOR);
		}

		function onPointerDown(e) {
			press = { x: e.clientX, y: e.clientY };
		}

		function onClick(e) {
			const booster = boosterFrom(e);
			if (!booster) return;

			// Dragging to tilt the card isn't a click
			if (press && Math.hypot(e.clientX - press.x, e.clientY - press.y) > DRAG_THRESHOLD) return;
			show(booster);
		}

		function onKeyDown(e) {
			if (e.key !== "Enter" && e.key !== " ") return;
			const booster = boosterFrom(e);
			if (!booster || e.target !== booster) return;

			e.preventDefault();
			show(booster);
		}

		// Boosters open a dialog, make them reachable and announced as buttons
		for (const booster of document.querySelectorAll(BOOSTER_SELECTOR)) {
			booster.tabIndex = 0;
			booster.setAttribute("role", "button");
			booster.setAttribute("aria-haspopup", "dialog");
			booster.setAttribute("aria-label", booster.dataset.title ?? "");
		}

		document.addEventListener("pointerdown", onPointerDown, { passive: true });
		document.addEventListener("click", onClick);
		document.addEventListener("keydown", onKeyDown);

		return () => {
			document.removeEventListener("pointerdown", onPointerDown);
			document.removeEventListener("click", onClick);
			document.removeEventListener("keydown", onKeyDown);
		};
	});
</script>

<dialog
	class="shop-buy-popup"
	aria-labelledby="shop-buy-popup-title"
	bind:this={dialog}
	onclick={onDialogClick}
	onclose={onDialogClose}
>
	<div class="shop-buy-popup-content">
		<div class="shop-buy-preview" bind:this={preview}></div>
		<h3 id="shop-buy-popup-title">{title}</h3>
		{#if coins !== undefined}
			<p class="shop-buy-balance">{labelBalance}: {coins}</p>
		{/if}
		<p class="shop-buy-note">{labelComingSoon}</p>
		<div class="shop-buy-actions">
			<button type="button" disabled>{labelBuy}</button>
			<button type="button" onclick={close}>{labelClose}</button>
		</div>
	</div>
</dialog>

<style>
	.shop-buy-popup {
		width: fit-content;
		max-width: min(100% - 2rem, 32rem);
		max-height: calc(100% - 2rem);
		padding: 0;

		overflow: auto;

		color: var(--color-text);
		background: var(--color-bg-2);
		border: 1px solid var(--color-border-1);
		border-radius: 1rem;
		box-shadow: 0 10px 30px var(--color-shadow);
	}

	.shop-buy-popup::backdrop {
		background: rgb(0 0 0 / 55%);
		backdrop-filter: blur(3px);
	}

	.shop-buy-popup[open] {
		animation: shop-buy-popup-in 150ms ease;
	}

	.shop-buy-popup-content {
		display: flex;
		flex-direction: column;
		gap: 1rem;

		padding: 1.25rem 1.25rem 1rem;
	}

	.shop-buy-preview {
		display: flex;
		justify-content: center;
	}

	.shop-buy-preview :global(.shop-booster-display) {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.65rem;
	}

	.shop-buy-preview :global(.shop-price-pill) {
		display: inline-flex;
		align-items: center;
		gap: 0.35em;

		padding: 0.35em 0.9em;

		font-weight: 650;
		color: var(--color-text);

		background: var(--color-bg-2);
		border: 1px solid var(--color-border-1);
		border-radius: 999px;
		box-shadow: 0 4px 12px var(--color-shadow);
	}

	h3 {
		margin: 0;

		font-size: 1.25rem;
		font-weight: 650;
		letter-spacing: -0.01em;
	}

	.shop-buy-balance {
		margin: 0;

		font-variant-numeric: tabular-nums;
	}

	.shop-buy-note {
		margin: 0;

		color: var(--color-text-muted);
		font-size: 0.9em;
	}

	.shop-buy-actions {
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
	}

	@keyframes shop-buy-popup-in {
		from {
			opacity: 0;
			transform: translateY(0.5rem) scale(0.98);
		}
	}

	/* Small screens: bottom sheet, like the other popups */
	@media (max-width: 480px) {
		.shop-buy-popup {
			width: 100%;
			max-width: 100%;
			margin: auto 0 0;

			border-radius: 1rem 1rem 0 0;
			border-bottom: 0;
		}

		.shop-buy-popup-content {
			padding-bottom: max(1rem, env(safe-area-inset-bottom));
		}
	}
</style>
