<!--
    Popup to buy boosters when clicking one in the shop

    Boosters must be `.shop-booster` elements holding a `.shop-booster-display`
    (the collection tile and its price pill) and advertising `data-collection`
    and `data-price`. The popup picks one of the preset quantities and posts to
    /shop/buy, landing that many boosters in the inventory (see TRANS-30 for the
    purchase and TRANS-35 for the bulk quantity).
-->

<script>
	import { onMount } from "svelte";
	import { openTopup } from "./topup.js";

	let { target } = $props();

	const BOOSTER_SELECTOR = ".shop-booster";
	const DISPLAY_SELECTOR = ".shop-booster-display";
	// Pointer travel (px) after which a press counts as a tilt drag, not a click
	const DRAG_THRESHOLD = 6;
	// The only quantities the shop sells at once
	const QUANTITY_PRESETS = [1, 5, 10];

	let dialog;
	let preview;
	let title = $state("");
	let collection = $state("");
	let price = $state(0);
	let quantity = $state(1);
	let returnFocus = null;

	const labelBuy = $derived(target.dataset.labelBuy ?? "");
	const labelBalance = $derived(target.dataset.labelBalance ?? "");
	const labelFunds = $derived(target.dataset.labelFunds ?? "");
	const labelLogin = $derived(target.dataset.labelLogin ?? "");
	const labelLoginHint = $derived(target.dataset.labelLoginHint ?? "");
	const labelClose = $derived(target.dataset.labelClose ?? "");
	const labelQuantity = $derived(target.dataset.labelQuantity ?? "");
	const labelTotal = $derived(target.dataset.labelTotal ?? "");
	const labelCoins = $derived(target.dataset.labelCoins ?? "");
	// Cap of one checkout, advertised by the shop page (maxBuyQuantity in Go)
	const maxQuantity = $derived(Number(target.dataset.maxQuantity ?? 10));
	// Coin balance is only advertised when logged in
	const coins = $derived(target.dataset.coins);
	const total = $derived(price * quantity);
	const canAfford = $derived(
		coins !== undefined && Number(coins) >= total,
	);

	function show(booster) {
		title = booster.dataset.title ?? "";
		collection = booster.dataset.collection ?? "";
		price = Number(booster.dataset.price ?? 0);
		quantity = 1;

		preview.replaceChildren();
		const display = booster.querySelector(DISPLAY_SELECTOR);
		if (display) preview.append(display.cloneNode(true));

		returnFocus = booster;
		dialog.showModal();
	}

	function close() {
		dialog.close();
	}

	// Short on coins: offer the packs that bring the balance up to the total
	function topUp() {
		const missing = total - Number(coins);
		close();
		openTopup(missing);
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
		{#if coins === undefined}
			<p class="shop-buy-note">{labelLoginHint}</p>
			<div class="shop-buy-actions">
				<a href="/login">{labelLogin}</a>
				<button type="button" onclick={close}>{labelClose}</button>
			</div>
		{:else}
			<p class="shop-buy-balance">{labelBalance}: {coins}</p>
			<div class="shop-buy-quantity">
				<span class="shop-buy-quantity-label" id="shop-buy-quantity-label">{labelQuantity}</span>
				<div class="shop-buy-presets" role="group" aria-labelledby="shop-buy-quantity-label">
					{#each QUANTITY_PRESETS as preset}
						<button
							type="button"
							class="shop-buy-preset"
							aria-pressed={quantity === preset}
							disabled={preset > maxQuantity}
							onclick={() => (quantity = preset)}
						>{preset}</button>
					{/each}
				</div>
			</div>
			<p class="shop-buy-total">{labelTotal}: {total} {labelCoins}</p>
			{#if !canAfford}
				<p class="shop-buy-note">{labelFunds}</p>
			{/if}
			<div class="shop-buy-actions">
				{#if canAfford}
					<form action="/shop/buy" method="post">
						<input type="hidden" name="collection" value={collection} />
						<input type="hidden" name="quantity" value={quantity} />
						<button type="submit">{labelBuy}</button>
					</form>
				{:else}
					<button type="button" onclick={topUp}>{labelBuy}</button>
				{/if}
				<button type="button" onclick={close}>{labelClose}</button>
			</div>
		{/if}
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

	.shop-buy-quantity {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.shop-buy-quantity-label {
		color: var(--color-text-muted);
		font-size: 0.9em;
	}

	.shop-buy-presets {
		display: flex;
		gap: 0.5rem;
	}

	.shop-buy-preset {
		min-width: 2.25rem;
		height: 2.25rem;
		padding: 0 0.5rem;

		color: var(--color-text);
		font-variant-numeric: tabular-nums;

		background: var(--color-bg-2);
		border: 1px solid var(--color-border-1);
		border-radius: 0.5rem;

		cursor: pointer;
	}

	.shop-buy-preset:disabled {
		cursor: default;
		opacity: 0.45;
	}

	.shop-buy-preset[aria-pressed="true"] {
		border-color: var(--color-accent-primary);
		color: var(--color-accent-primary);
	}

	.shop-buy-total {
		margin: 0;

		font-weight: 650;
		font-variant-numeric: tabular-nums;
	}

	.shop-buy-note {
		margin: 0;

		color: var(--color-text-muted);
		font-size: 0.9em;
	}

	.shop-buy-actions {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 0.75rem;
	}

	.shop-buy-actions form {
		display: contents;
	}

	.shop-buy-actions a {
		margin-right: auto;

		color: var(--color-accent-primary);
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
