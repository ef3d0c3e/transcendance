<!--
    Builds a tradeup contract: clicking a card in the inventory adds a copy of
    it to the bar, clicking a filled slot takes that copy back out. The
    contract is submitted as one `card` field per copy.

    Cards must provide `data-tradeup-id`, `data-tradeup-foil`,
    `data-tradeup-count` (copies owned), `data-tradeup-rarity`,
    `data-tradeup-title`, `data-tradeup-collection-id` and
    `data-tradeup-order` (the place of the card in the collection-and-rarity
    order, which `#tradeup-sort` brings back). A contract is all one rarity,
    so cards of another rarity can't join once one is picked. Stacks with
    copies in the contract wear a `.tradeup-taken` pill.

    The optional `#tradeup-collection-filter` narrows the grid down to one
    collection, `#tradeup-rarity-filter` to one rarity, and `#tradeup-search`
    to the cards whose name or collection (`data-tradeup-collection`) holds
    its text. `#tradeup-sort` orders the grid, the biggest stacks first or in
    the collection-and-rarity order of `data-tradeup-order`. All of it without
    navigating, keeping the contract: the picks live here, not in the page.
    Picks the filters hide stay in it. `#tradeup-no-match` shows when they
    hide every card, and `#tradeup-filter-status` tells screen readers what
    they kept. The grid is paged in the page too, in the `.tradeup-pager`
    elements, so picks survive turning pages.
-->

<script>
	import { onMount, tick } from "svelte";
	import { attachCardFilter } from "./card-filter.js";
	import { makeCardButton } from "./card-name.js";

	let { target } = $props();

	const CARD_SELECTOR = ".tradeup-card[data-tradeup-id]";
	const TILE_SELECTOR = `${CARD_SELECTOR} .card-container`;
	const FILTER_SELECTOR = "#tradeup-collection-filter";
	const SEARCH_SELECTOR = "#tradeup-search";
	const RARITY_SELECTOR = "#tradeup-rarity-filter";
	const SORT_SELECTOR = "#tradeup-sort";
	const NO_MATCH_SELECTOR = "#tradeup-no-match";
	const STATUS_SELECTOR = "#tradeup-filter-status";
	// Pointer travel (px) after which a press counts as a tilt drag, not a click
	const DRAG_THRESHOLD = 6;

	const cost = $derived(Number(target.dataset.cost ?? 0));
	const labelSlots = $derived(target.dataset.labelSlots ?? "");
	const labelRemove = $derived(target.dataset.labelRemove ?? "");
	const labelClear = $derived(target.dataset.labelClear ?? "");
	const labelSubmit = $derived(target.dataset.labelSubmit ?? "");
	const labelEmpty = $derived(target.dataset.labelEmpty ?? "");
	const labelFoil = $derived(target.dataset.labelFoil ?? "");
	const labelCopies = $derived(target.dataset.labelCopies ?? "");

	// Copies put into the contract, in the order they were added
	let selection = $state([]);
	// The slots' group, focused when the last filled slot is taken out
	let slotsElement = $state();
	// Rarity of the contract so far, "" while it is still empty
	const rarity = $derived(selection[0]?.rarity ?? "");
	const filled = $derived(selection.length);
	const ready = $derived(filled === cost);
	// One slot per card of the contract
	const slots = $derived(Array.from({ length: cost }, (_, index) => index));

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

	// The focused slot is gone: the focus goes to the slot taking its place,
	// else the one before, else the slots
	async function remove(index) {
		selection.splice(index, 1);
		await tick();
		const filledSlots = slotsElement.querySelectorAll(".tradeup-slot.is-filled");
		(filledSlots[index] ?? filledSlots[index - 1] ?? slotsElement).focus();
	}

	// The clear button is disabled once the contract is empty
	async function clear() {
		selection = [];
		await tick();
		slotsElement.focus();
	}

	// Tiles of another rarity, exhausted stacks and a full contract are inert;
	// each stack also wears a pill with how many of its copies are taken
	$effect(() => {
		for (const card of document.querySelectorAll(CARD_SELECTOR)) {
			const info = cardOf(card);
			const blocked = !canTake(info);
			card.classList.toggle("tradeup-card-blocked", blocked);

			const tile = card.querySelector(".card-container");
			if (!tile) continue;
			// On the button itself, so screen readers tell it does nothing
			tile.setAttribute("aria-disabled", blocked ? "true" : "false");
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

		// The focused element is the tile, inside the card's wrapper
		// A held key adds the card once, not until the contract is full
		function onKeyDown(e) {
			if (e.key !== "Enter" && e.key !== " ") return;
			if (e.repeat) return;
			if (!(e.target instanceof Element) || !e.target.matches(TILE_SELECTOR)) return;

			e.preventDefault();
			add(e.target.closest(CARD_SELECTOR));
		}

		// Cards add to the contract, make them reachable and announced as buttons
		for (const tile of document.querySelectorAll(TILE_SELECTOR)) {
			const card = tile.closest(CARD_SELECTOR);
			makeCardButton(tile, {
				foil: labelFoil,
				copies: labelCopies,
				count: card.dataset.tradeupCount,
			});
		}

		document.addEventListener("pointerdown", onPointerDown, { passive: true });
		document.addEventListener("click", onClick);
		document.addEventListener("keydown", onKeyDown);

		const detachFilter = attachCardFilter({
			tiles: CARD_SELECTOR,
			read: (tile) => ({
				title: tile.dataset.tradeupTitle ?? "",
				collection: tile.dataset.tradeupCollection ?? "",
				collectionId: tile.dataset.tradeupCollectionId ?? "",
				rarity: tile.dataset.tradeupRarity ?? "",
				number: Number(tile.dataset.tradeupId ?? 0),
				count: Number(tile.dataset.tradeupCount ?? 0),
				order: Number(tile.dataset.tradeupOrder ?? 0),
			}),
			search: SEARCH_SELECTOR,
			collection: FILTER_SELECTOR,
			rarity: RARITY_SELECTOR,
			sort: SORT_SELECTOR,
			noMatch: NO_MATCH_SELECTOR,
			status: STATUS_SELECTOR,
			pagers: ".tradeup-pager",
		});

		return () => {
			document.removeEventListener("pointerdown", onPointerDown);
			document.removeEventListener("click", onClick);
			document.removeEventListener("keydown", onKeyDown);
			detachFilter();
		};
	});
</script>

<div class="tradeup-picker">
	<div class="tradeup-slots" role="group" aria-label={labelSlots} tabindex="-1" bind:this={slotsElement}>
		{#each slots as i (i)}
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

	<!-- Pinned to the bottom of the sidebar, or of the screen on small ones -->
	<div class="tradeup-actions">
		<p class="tradeup-progress" aria-live="polite">
			<span class="tradeup-progress-label">{labelSlots}</span>
			<span class="tradeup-progress-count">{filled} / {cost}</span>
		</p>
		<button type="button" class="tradeup-clear" onclick={clear} disabled={filled === 0}>
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

	/* How full the contract is, on a row of its own above the buttons */
	.tradeup-progress {
		grid-column: 1 / -1;

		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 0.75rem;
		margin: 0;

		line-height: 1.2;
	}

	.tradeup-progress-label {
		min-width: 0;

		color: var(--color-text-muted);
		font-size: 0.875rem;
		font-weight: 600;
	}

	.tradeup-progress-count {
		font-size: 1.15rem;
		white-space: nowrap;
		font-variant-numeric: tabular-nums;
		font-weight: 650;
		letter-spacing: -0.01em;
	}

	.tradeup-slots {
		border-radius: 0.5rem;
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
			outline: 3px solid var(--color-accent-secondary);
			outline-offset: 2px;
		}
	}

	/* Pinned to the bottom of the scrolling sidebar, past the padding it
	   keeps for focus rings, so nothing shows under the bar */
	.tradeup-actions {
		position: sticky;
		bottom: -4px;
		z-index: 1;

		/* The progress on a row of its own, the two buttons sharing the next */
		display: grid;
		grid-template-columns: 1fr 1fr;
		align-items: center;
		gap: 0.6rem 0.75rem;

		margin: 0 -1.25rem -1.25rem;
		padding: 0.85rem 1.25rem 1.25rem;

		background: var(--color-bg-2);
		border-top: 1px solid var(--color-border-1);
		border-radius: 0 0 1rem 1rem;
	}

	/* Labels never break inside a word: the buttons share the row instead */
	.tradeup-clear,
	.tradeup-submit {
		display: inline-flex;
		align-items: center;
		justify-content: center;

		min-width: 0;
		min-height: 2.5rem;
		padding: 0.45rem 0.75rem;

		white-space: nowrap;

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
			outline: 3px solid var(--color-accent-secondary);
			outline-offset: 2px;
		}
	}

	.tradeup-clear:disabled,
	.tradeup-submit:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	/* The sidebar comes after the grid: the bar is fixed to the screen instead */
	@media (max-width: 900px) {
		.tradeup-actions {
			position: fixed;
			right: 0;
			bottom: 0;
			left: 0;
			z-index: 30;

			margin: 0;
			padding: 0.75rem max(1rem, env(safe-area-inset-right)) max(0.75rem, env(safe-area-inset-bottom))
				max(1rem, env(safe-area-inset-left));

			border: 1px solid var(--color-border-1);
			border-bottom: 0;
			border-radius: 1rem 1rem 0 0;
			box-shadow: 0 -6px 20px var(--color-shadow);
		}
	}

	/* Across a whole screen there is room for one row */
	@media (480px < width <= 900px) {
		.tradeup-actions {
			grid-template-columns: 1fr auto auto;
		}

		.tradeup-progress {
			grid-column: auto;
			justify-content: flex-start;
		}

		.tradeup-clear,
		.tradeup-submit {
			min-width: 7rem;
		}
	}
</style>
