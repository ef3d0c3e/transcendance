<!--
    Builds a tradeup contract: clicking a card in the inventory adds a copy of
    it to the bar, clicking a filled slot takes that copy back out. The
    contract is submitted as one `card` field per copy.

    Cards must provide `data-tradeup-id`, `data-tradeup-foil`,
    `data-tradeup-count` (copies owned), `data-tradeup-rarity`,
    `data-tradeup-title` and `data-tradeup-collection-id`. A contract is all
    one rarity, so cards of another rarity can't join once one is picked.
    Stacks with copies in the contract wear a `.tradeup-taken` pill.

    The optional `#tradeup-collection-filter` narrows the grid down to one
    collection without navigating, keeping the contract: the picks live here,
    not in the page. Picks the filter hides stay in it.
-->

<script>
	import { onMount } from "svelte";

	let { target } = $props();

	const CARD_SELECTOR = ".tradeup-card[data-tradeup-id]";
	const TILE_SELECTOR = `${CARD_SELECTOR} .card-container`;
	const FILTER_SELECTOR = "#tradeup-collection-filter";
	// Pointer travel (px) after which a press counts as a tilt drag, not a click
	const DRAG_THRESHOLD = 6;

	const cost = $derived(Number(target.dataset.cost ?? 0));
	const labelSlots = $derived(target.dataset.labelSlots ?? "");
	const labelRemove = $derived(target.dataset.labelRemove ?? "");
	const labelClear = $derived(target.dataset.labelClear ?? "");
	const labelSubmit = $derived(target.dataset.labelSubmit ?? "");
	const labelEmpty = $derived(target.dataset.labelEmpty ?? "");

	// Copies put into the contract, in the order they were added
	let selection = $state([]);
	// Rarity of the contract so far, "" while it is still empty
	let rarity = $derived(selection[0]?.rarity ?? "");
	let filled = $derived(selection.length);
	let ready = $derived(filled === cost);

	function cardOf(tile) {
		return {
			id: tile.dataset.tradeupId,
			foil: tile.dataset.tradeupFoil ?? "",
			count: Number(tile.dataset.tradeupCount ?? 0),
			rarity: tile.dataset.tradeupRarity ?? "",
			title: tile.dataset.tradeupTitle ?? "",
		};
	}

	// Copies of a stack already in the contract
	function taken(id, foil) {
		let n = 0;
		for (const card of selection) {
			if (card.id === id && card.foil === foil) n++;
		}
		return n;
	}

	function canTake(card) {
		if (rarity !== "" && card.rarity !== rarity) return false;
		if (taken(card.id, card.foil) >= card.count) return false;
		return filled < cost;
	}

	function add(tile) {
		const card = cardOf(tile);
		if (!canTake(card)) return;
		selection.push(card);
	}

	function remove(index) {
		selection.splice(index, 1);
	}

	// Hides the tiles of the other collections without navigating, so a
	// contract in progress survives. `value` is a collection ID, "" shows all.
	function applyFilter(value) {
		for (const card of document.querySelectorAll(CARD_SELECTOR)) {
			card.hidden = value !== "" && card.dataset.tradeupCollectionId !== value;
		}

		// A refresh lands on the same filter
		const url = new URL(location.href);
		if (value === "") url.searchParams.delete("collection");
		else url.searchParams.set("collection", value);
		history.replaceState(history.state, "", url);
	}

	// Tiles of another rarity, exhausted stacks and a full contract are inert;
	// each stack also wears a pill with how many of its copies are taken
	$effect(() => {
		for (const card of document.querySelectorAll(CARD_SELECTOR)) {
			const info = cardOf(card);
			const blocked = !canTake(info);
			card.classList.toggle("tradeup-card-blocked", blocked);
			card.setAttribute("aria-disabled", blocked ? "true" : "false");

			const tile = card.querySelector(".card-container");
			if (!tile) continue;
			const count = taken(info.id, info.foil);
			let pill = tile.querySelector(":scope > .tradeup-taken");
			if (count === 0) {
				pill?.remove();
				continue;
			}
			if (!pill) {
				pill = document.createElement("span");
				pill.className = "tradeup-taken";
				pill.setAttribute("aria-hidden", "true");
				tile.append(pill);
			}
			pill.textContent = `×${count}`;
		}
	});

	onMount(() => {
		let press = null;

		function cardFrom(e) {
			if (!(e.target instanceof Element)) return null;
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
			add(card);
		}

		function onKeyDown(e) {
			if (e.key !== "Enter" && e.key !== " ") return;
			const card = cardFrom(e);
			if (!card || e.target !== card) return;

			e.preventDefault();
			add(card);
		}

		// Cards add to the contract, make them reachable and announced as buttons
		for (const tile of document.querySelectorAll(TILE_SELECTOR)) {
			tile.tabIndex = 0;
			tile.setAttribute("role", "button");
			tile.setAttribute("aria-label", tile.closest(CARD_SELECTOR).dataset.tradeupTitle);
		}

		document.addEventListener("pointerdown", onPointerDown, { passive: true });
		document.addEventListener("click", onClick);
		document.addEventListener("keydown", onKeyDown);

		const filter = document.querySelector(FILTER_SELECTOR);
		function onFilterChange() {
			applyFilter(filter?.value ?? "");
		}
		filter?.addEventListener("change", onFilterChange);

		// A shared link can arrive already filtered
		const initial = new URL(location.href).searchParams.get("collection") ?? "";
		if (filter && initial) {
			filter.value = initial;
			applyFilter(initial);
		}

		return () => {
			document.removeEventListener("pointerdown", onPointerDown);
			document.removeEventListener("click", onClick);
			document.removeEventListener("keydown", onKeyDown);
			filter?.removeEventListener("change", onFilterChange);
		};
	});
</script>

<div class="tradeup-picker">
	<p class="tradeup-progress" aria-live="polite">
		<span class="tradeup-progress-count">{filled} / {cost}</span>
	</p>

	<div class="tradeup-slots" role="group" aria-label={labelSlots}>
		{#each Array(cost) as _, i}
			{#if i < filled}
				<button
					type="button"
					class="tradeup-slot is-filled"
					title={selection[i].title}
					aria-label="{labelRemove} {selection[i].title}"
					style="background-image: url('/cards/{selection[i].id}/thumbnail')"
					onclick={() => remove(i)}
				></button>
			{:else}
				<span class="tradeup-slot is-empty" title={labelEmpty}></span>
			{/if}
		{/each}
	</div>

	{#each selection as card, i (i)}
		<input type="hidden" name="card" value={card.foil ? `${card.id}:${card.foil}` : `${card.id}`} />
	{/each}

	<div class="tradeup-actions">
		<button type="button" class="tradeup-clear" onclick={() => (selection = [])} disabled={filled === 0}>
			{labelClear}
		</button>
		<button type="submit" class="tradeup-submit" disabled={!ready}>{labelSubmit}</button>
	</div>
</div>

<style>
	.tradeup-picker {
		display: flex;
		flex-direction: column;
		gap: 1rem;

		padding: 1.25rem;

		background: var(--color-bg-2);
		border: 1px solid var(--color-border-1);
		border-radius: 1rem;
	}

	.tradeup-progress {
		margin: 0;

		font-variant-numeric: tabular-nums;
		font-weight: 650;
		letter-spacing: -0.01em;
	}

	.tradeup-slots {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(2.5rem, 1fr));
		gap: 0.5rem;
	}

	.tradeup-slot {
		aspect-ratio: 1;
		padding: 0;

		background-color: var(--color-bg-4);
		background-size: cover;
		background-position: center;
		border: 1px solid var(--color-border-1);
		border-radius: 0.45rem;

		cursor: default;
	}

	.tradeup-slot.is-empty {
		opacity: 0.55;
	}

	.tradeup-slot.is-filled {
		cursor: pointer;

		&:hover {
			border-color: var(--color-accent-primary);
		}

		&:focus-visible {
			outline: 3px solid color-mix(in srgb, var(--color-accent-secondary) 55%, transparent);
			outline-offset: 2px;
		}
	}

	.tradeup-actions {
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
	}

	.tradeup-clear,
	.tradeup-submit {
		display: inline-flex;
		align-items: center;
		justify-content: center;

		min-width: 5.5rem;
		min-height: 2.5rem;
		padding: 0.45rem 1rem;

		border: 0;
		border-radius: 0.55rem;

		font-weight: 600;
		line-height: 1.3;

		transition: background-color 120ms ease;
	}

	.tradeup-clear {
		color: var(--color-text);
		background: var(--color-bg-4);

		&:hover:not(:disabled) {
			background: color-mix(in srgb, var(--color-bg-4), black 10%);
		}
	}

	.tradeup-submit {
		color: var(--color-text-alt);
		background: var(--color-accent-primary);

		&:hover:not(:disabled) {
			color: var(--color-text-alt);
			background: var(--color-accent-primary-dark);
		}

		&:focus-visible {
			outline: 3px solid color-mix(in srgb, var(--color-accent-secondary) 55%, transparent);
			outline-offset: 2px;
		}
	}

	.tradeup-clear:disabled,
	.tradeup-submit:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	@media (max-width: 480px) {
		.tradeup-actions {
			flex-direction: column-reverse;
		}

		.tradeup-clear,
		.tradeup-submit {
			width: 100%;
		}
	}
</style>
