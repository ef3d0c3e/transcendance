<!--
    Popup with stack details (copies, acquisition dates) when clicking an inventory card

    Cards must provide `data-copies`, `data-acquired-first` and `data-acquired-last` (RFC 3339 dates).
    On their owner's inventory they also provide `data-sell-id`, `data-sell-foil`, `data-sell-price`
    and `data-sell-owned`, the copies free to sell, and the popup offers to sell some to the shop.
-->

<script>
	import { onMount, tick } from "svelte";
	import { postSale, applySale, focusNotice } from "./sell.js";
	import { makeCardButton } from "./card-name.js";

	let { target } = $props();

	const CARD_SELECTOR = ".inventory-cards .card-container[data-acquired-first]";
	// Pointer travel (px) after which a press counts as a tilt drag, not a click
	const DRAG_THRESHOLD = 6;

	let dialog;
	let title = $state("");
	let copies = $state(1);
	// Single date for one copy, first and last otherwise
	let dates = $state([]);
	let returnFocus = null;

	let sellId = $state("");
	let sellFoil = $state("");
	let sellPrice = $state(0);
	let sellOwned = $state(0);
	let selling = $state(false);
	let quantity = $state(1);
	let error = $state("");
	let pending = $state(false);

	const labelAcquired = $derived(target.dataset.labelAcquired ?? "");
	const labelAcquiredFirst = $derived(target.dataset.labelAcquiredFirst ?? "");
	const labelAcquiredLast = $derived(target.dataset.labelAcquiredLast ?? "");
	const labelCopies = $derived(target.dataset.labelCopies ?? "");
	const labelClose = $derived(target.dataset.labelClose ?? "");
	const labelValue = $derived(target.dataset.labelValue ?? "");
	const labelSell = $derived(target.dataset.labelSell ?? "");
	const labelQuantity = $derived(target.dataset.labelQuantity ?? "");
	const labelTotal = $derived(target.dataset.labelTotal ?? "");
	const labelConfirm = $derived(target.dataset.labelConfirm ?? "");
	const labelCoins = $derived(target.dataset.labelCoins ?? "");
	const labelError = $derived(target.dataset.labelError ?? "");
	const labelFoil = $derived(target.dataset.labelFoil ?? "");
	const sellAction = $derived(target.dataset.sellAction ?? "");
	// The popup offers to sell only where the card carries a price and the
	// owner has copies of it that are not up in trades
	const sellable = $derived(sellAction !== "" && sellId !== "" && sellOwned > 0);
	const total = $derived(sellPrice * quantity);
	const numberFormat = $derived(new Intl.NumberFormat(target.dataset.locale || undefined));
	const dateFormat = $derived(
		new Intl.DateTimeFormat(target.dataset.locale || undefined, {
			dateStyle: "long",
			timeStyle: "short",
		}),
	);

	function show(card) {
		title = card.querySelector(".card-title")?.textContent ?? "";
		copies = Number(card.dataset.copies) || 1;
		const date = (label, iso) => ({ label, iso, text: dateFormat.format(new Date(iso)) });
		dates =
			copies > 1
				? [
						date(labelAcquiredFirst, card.dataset.acquiredFirst),
						date(labelAcquiredLast, card.dataset.acquiredLast),
					]
				: [date(labelAcquired, card.dataset.acquiredFirst)];
		sellId = card.dataset.sellId ?? "";
		sellFoil = card.dataset.sellFoil ?? "";
		sellPrice = Number(card.dataset.sellPrice ?? 0);
		sellOwned = Number(card.dataset.sellOwned ?? 0);
		selling = false;
		quantity = 1;
		error = "";
		returnFocus = card;
		dialog.showModal();
	}

	function close() {
		dialog.close();
	}

	// The sell button is replaced by the form: the focus goes to its first
	// control instead of falling to the dialog
	async function startSelling() {
		selling = true;
		await tick();
		dialog.querySelector(".inventory-sell-form input[type=range], .inventory-sell-form button[type=submit]")?.focus();
	}

	function onDialogClose() {
		returnFocus?.focus();
		returnFocus = null;
	}

	// Clicking the backdrop targets the dialog itself. Pressed inside (the
	// quantity slider) and released over the backdrop isn't one.
	let pressedBackdrop = false;

	function onDialogPointerdown(e) {
		pressedBackdrop = e.target === dialog;
	}

	function onDialogClick(e) {
		if (e.target === dialog && pressedBackdrop) close();
		pressedBackdrop = false;
	}

	// Sell the copies picked without reloading, then close on the sale: the
	// page behind the popup gets the coins, the copies and the message. The
	// confirm button is only marked busy meanwhile: disabled, it would drop
	// the focus, which stays on it when the sale fails.
	async function onSellSubmit(e) {
		e.preventDefault();
		if (pending) return;
		pending = true;
		error = "";

		const result = await postSale(e.currentTarget);
		pending = false;
		if (result.error) {
			error = result.error;
			return;
		}

		const tile = returnFocus;
		// The copies sold are those of a card the page lists
		applySale(result, { sold: Number(quantity), inView: true });
		close();
		// Selling the last copy takes the card out of the page, and the focus
		// with it: the sale's message gets the focus instead
		if (tile && !document.contains(tile)) {
			focusNotice();
		}
	}

	onMount(() => {
		let press = null;

		function cardFrom(e) {
			if (!(e.target instanceof Element)) return null;
			// Let the collection link work
			if (e.target.closest(".card-collection")) return null;
			return e.target.closest(CARD_SELECTOR);
		}

		function onPointerDown(e) {
			press = { x: e.clientX, y: e.clientY };
		}

		function onClick(e) {
			const card = cardFrom(e);
			if (!card) return;

			// Dragging to tilt the card isn't a click
			if (press && Math.hypot(e.clientX - press.x, e.clientY - press.y) > DRAG_THRESHOLD) return;
			show(card);
		}

		function onKeyDown(e) {
			if (e.key !== "Enter" && e.key !== " ") return;
			const card = cardFrom(e);
			if (!card || e.target !== card) return;

			e.preventDefault();
			show(card);
		}

		// Cards open a dialog, make them reachable and announced as buttons
		for (const card of document.querySelectorAll(CARD_SELECTOR)) {
			makeCardButton(card, { foil: labelFoil, copies: labelCopies });
			card.setAttribute("aria-haspopup", "dialog");
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
	class="popup inventory-popup"
	aria-labelledby="inventory-popup-title"
	bind:this={dialog}
	onpointerdown={onDialogPointerdown}
	onclick={onDialogClick}
	onclose={onDialogClose}
>
	<div class="inventory-popup-content">
		<h2 id="inventory-popup-title">{title}</h2>
		<dl>
			{#if copies > 1}
				<div>
					<dt>{labelCopies}</dt>
					<dd>{copies}</dd>
				</div>
			{/if}
			{#each dates as { label, iso, text } (label)}
				<div>
					<dt>{label}</dt>
					<dd><time datetime={iso}>{text}</time></dd>
				</div>
			{/each}
			{#if sellable}
				<div>
					<dt>{labelValue}</dt>
					<dd>{numberFormat.format(sellPrice)} {labelCoins}</dd>
				</div>
			{/if}
		</dl>
		{#if sellable && selling}
			<form class="inventory-sell-form" method="post" action={sellAction}
				data-label-error={labelError} onsubmit={onSellSubmit}>
				<input type="hidden" name="card" value={sellId} />
				<input type="hidden" name="foil" value={sellFoil} />
				<input type="hidden" name="quantity" value={quantity} />
				{#if sellOwned > 1}
					<label class="inventory-sell-quantity">
						<span>{labelQuantity}</span>
						<input type="range" min="1" max={sellOwned} bind:value={quantity} />
						<output>{quantity} / {sellOwned}</output>
					</label>
				{/if}
				<p class="inventory-sell-total">{labelTotal}: {numberFormat.format(total)} {labelCoins}</p>
				{#if error}
					<p class="section-error" role="alert">{error}</p>
				{/if}
				<div class="inventory-popup-actions">
					<button type="button" class="inventory-popup-neutral" onclick={close}>{labelClose}</button>
					<button type="submit" aria-disabled={pending ? "true" : undefined}>{labelConfirm}</button>
				</div>
			</form>
		{:else}
			<div class="inventory-popup-actions">
				{#if sellable}
					<button type="button" onclick={startSelling}>{labelSell}</button>
				{/if}
				<button type="button" class:inventory-popup-neutral={sellable} onclick={close}>{labelClose}</button>
			</div>
		{/if}
	</div>
</dialog>

<style>
	/* Frame, backdrop and bottom sheet: the shared .popup of theme.css */
	.inventory-popup {
		--popup-width: 22rem;
	}

	.inventory-popup-content {
		display: flex;
		flex-direction: column;
		gap: 1rem;

		padding: 1.25rem 1.25rem 1rem;
	}

	h2 {
		margin: 0;

		font-size: 1.25rem;
		font-weight: 650;
		letter-spacing: -0.01em;
	}

	dl {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;

		margin: 0;
	}

	dt {
		color: var(--color-text-muted);
		font-size: 0.875rem;
		font-weight: 600;
	}

	dd {
		margin: 0.15rem 0 0;

		font-size: 1.05rem;
		font-variant-numeric: tabular-nums;
	}

	.inventory-popup-actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: flex-end;
		gap: 0.5rem;
	}

	/* Closing is the neutral action next to selling */
	.inventory-popup-neutral {
		color: var(--color-text);
		background: var(--color-bg-4);
	}

	.inventory-popup-neutral:hover {
		background: color-mix(in srgb, var(--color-bg-4), black 10%);
	}

	.inventory-sell-form {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.inventory-sell-quantity {
		display: flex;
		align-items: center;
		gap: 0.6rem;

		font-size: 0.875rem;
		font-weight: 600;
	}

	.inventory-sell-quantity input {
		flex: 1;
	}

	.inventory-sell-quantity output {
		color: var(--color-text-muted);
		font-variant-numeric: tabular-nums;
	}

	.inventory-sell-total {
		margin: 0;

		font-weight: 650;
		font-variant-numeric: tabular-nums;
	}

	@media (max-width: 480px) {
		.inventory-popup-content {
			padding-bottom: max(1rem, env(safe-area-inset-bottom));
		}

		.inventory-popup-actions button {
			flex: 1;
		}
	}

</style>
