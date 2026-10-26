<!--
    Popup with stack details (copies, acquisition dates) when clicking an inventory card

    Cards must provide `data-copies`, `data-acquired-first` and `data-acquired-last` (RFC 3339 dates).
    On their owner's inventory they also provide `data-sell-id`, `data-sell-price`,
    `data-sell-owned` and optionally `data-sell-foil`, and the popup offers
    to sell copies of the card to the shop.
-->

<script>
	import { onMount } from "svelte";

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
	const sellAction = $derived(target.dataset.sellAction ?? "");
	// The popup offers to sell only where the card carries a price and the
	// owner has copies of it
	const sellable = $derived(sellAction !== "" && sellId !== "" && sellOwned > 0);
	const total = $derived(sellPrice * quantity);
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
		returnFocus = card;
		dialog.showModal();
	}

	function close() {
		dialog.close();
	}

	function onDialogClose() {
		returnFocus?.focus();
		returnFocus = null;
	}

	// Clicking the backdrop targets the dialog itself
	function onDialogClick(e) {
		if (e.target === dialog) close();
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
			card.tabIndex = 0;
			card.setAttribute("role", "button");
			card.setAttribute("aria-haspopup", "dialog");
			card.setAttribute("aria-label", card.querySelector(".card-title")?.textContent ?? "");
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
	class="inventory-popup"
	aria-labelledby="inventory-popup-title"
	bind:this={dialog}
	onclick={onDialogClick}
	onclose={onDialogClose}
>
	<div class="inventory-popup-content">
		<h3 id="inventory-popup-title">{title}</h3>
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
					<dd>{sellPrice} {labelCoins}</dd>
				</div>
			{/if}
		</dl>
		{#if sellable && selling}
			<form class="inventory-sell-form" method="post" action={sellAction}>
				<input type="hidden" name="card" value={sellId} />
				<input type="hidden" name="foil" value={sellFoil} />
				<input type="hidden" name="quantity" value={quantity} />
				<label class="inventory-sell-quantity">
					<span>{labelQuantity}</span>
					<input type="range" min="1" max={sellOwned} bind:value={quantity} />
					<output>{quantity} / {sellOwned}</output>
				</label>
				<p class="inventory-sell-total">{labelTotal}: {total} {labelCoins}</p>
				<button type="submit">{labelConfirm}</button>
			</form>
		{:else if sellable}
			<button type="button" onclick={() => (selling = true)}>{labelSell}</button>
		{/if}
		<button type="button" onclick={close}>{labelClose}</button>
	</div>
</dialog>

<style>
	.inventory-popup {
		width: min(100% - 2rem, 22rem);
		padding: 0;

		color: var(--color-text);
		background: var(--color-bg-2);
		border: 1px solid var(--color-border-1);
		border-radius: 1rem;
		box-shadow: 0 10px 30px var(--color-shadow);
	}

	.inventory-popup::backdrop {
		background: rgb(0 0 0 / 55%);
		backdrop-filter: blur(3px);
	}

	.inventory-popup[open] {
		animation: popup-in 150ms ease;
	}

	.inventory-popup-content {
		display: flex;
		flex-direction: column;
		gap: 1rem;

		padding: 1.25rem 1.25rem 1rem;
	}

	h3 {
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

	button {
		align-self: flex-end;
	}

	@keyframes popup-in {
		from {
			opacity: 0;
			transform: translateY(0.5rem) scale(0.98);
		}
	}

	/* Small screens: bottom sheet, like the other popups */
	@media (max-width: 480px) {
		.inventory-popup {
			width: 100%;
			max-width: 100%;
			margin: auto 0 0;

			border-radius: 1rem 1rem 0 0;
			border-bottom: 0;
		}

		.inventory-popup-content {
			padding-bottom: max(1rem, env(safe-area-inset-bottom));
		}

		button {
			align-self: stretch;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.inventory-popup[open] {
			animation: none;
		}
	}
</style>
