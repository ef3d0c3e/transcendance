<!--
    Side by side panel to build a trade offer: the cards and coins the user
    gives on one side, the cards they want from the partner on the other

    The mount point must provide `data-target`, `data-stacks-url` and
    `data-submit-url`, optionally `data-counter-id`, `data-prefill-give` and
    `data-prefill-want` (JSON lists of card stacks), and the `data-label-*`
    strings
-->

<script>
	import { onMount } from "svelte";

	let { target } = $props();

	const targetName = $derived(target.dataset.target ?? "");
	const stacksUrl = $derived(target.dataset.stacksUrl ?? "");
	const submitUrl = $derived(target.dataset.submitUrl ?? "");
	const counterId = $derived(Number(target.dataset.counterId) || 0);

	const labelGive = $derived(target.dataset.labelGive ?? "");
	const labelGet = $derived(target.dataset.labelGet ?? "");
	const labelCoins = $derived(target.dataset.labelCoins ?? "");
	const labelSend = $derived(target.dataset.labelSend ?? "");
	const labelCounter = $derived(target.dataset.labelCounter ?? "");
	const labelBack = $derived(target.dataset.labelBack ?? "");
	const labelEmpty = $derived(target.dataset.labelEmpty ?? "");
	const labelLoading = $derived(target.dataset.labelLoading ?? "");
	const labelIncrease = $derived(target.dataset.labelIncrease ?? "");
	const labelDecrease = $derived(target.dataset.labelDecrease ?? "");
	const labelFoil = $derived(target.dataset.labelFoil ?? "");
	const labelError = $derived(target.dataset.labelError ?? "");

	let mine = $state([]);
	let theirs = $state([]);
	let balance = $state(0);
	// Copies put up per stack, by `card_id:foil`
	let give = $state({});
	let want = $state({});
	let coins = $state(0);
	let loading = $state(true);
	let sending = $state(false);
	let message = $state("");

	// Width of a picker column and of the fixed 30ch card box, in pixels. The
	// cards are scaled to the column so a tile never overflows it, whatever
	// the viewport.
	let pickerWidth = $state(0);
	let cardWidth = $state(0);
	const cardScale = $derived(
		pickerWidth && cardWidth ? Math.min(0.6, pickerWidth / cardWidth) : 0.6,
	);

	const key = (stack) => `${stack.card_id}:${stack.foil}`;

	// Stacks put up on each side, in inventory order
	const givePicked = $derived(mine.filter((stack) => give[key(stack)]));
	const wantPicked = $derived(theirs.filter((stack) => want[key(stack)]));

	function title(stack) {
		return stack.foil ? `${stack.title} · ${labelFoil}` : stack.title;
	}

	// Same rarity palette as the card badges
	function rarityColor(rarity) {
		switch (rarity) {
			case "rare":
				return "#ff1440";
			case "legendary":
				return "#ffdc50";
			default:
				return "#407fff";
		}
	}

	function toggle(side, stack) {
		const sel = side === "give" ? give : want;
		if (sel[key(stack)]) {
			delete sel[key(stack)];
		} else {
			sel[key(stack)] = 1;
		}
	}

	// Picking a tile, clicks on the card's own links are left to the browser
	function pick(e, side, stack) {
		if (e.target instanceof Element && e.target.closest("a")) return;
		toggle(side, stack);
	}

	function pickKey(e, side, stack) {
		if (e.key !== "Enter" && e.key !== " ") return;
		e.preventDefault();
		toggle(side, stack);
	}

	function step(side, stack, delta) {
		const sel = side === "give" ? give : want;
		const next = (sel[key(stack)] ?? 0) + delta;
		if (next <= 0) {
			delete sel[key(stack)];
			return;
		}
		sel[key(stack)] = Math.min(next, stack.count);
	}

	// The balance is the most that can be put up: cap the amount as it is typed.
	// An empty field is left alone so it can be retyped into.
	function capCoins(e) {
		const input = e.currentTarget;
		if (input.value === "") {
			return;
		}
		const amount = Math.min(Math.max(0, Number(input.value) || 0), balance);
		coins = amount;
		if (String(amount) !== input.value) {
			input.value = String(amount);
		}
	}

	// Start a side from the stacks of a counter, capped to the stacks owned
	function prefill(sel, stacks, encoded) {
		let items;
		try {
			items = JSON.parse(encoded || "[]");
		} catch {
			items = [];
		}
		for (const item of items) {
			const stack = stacks.find((s) => s.card_id === item.card_id && s.foil === item.foil);
			if (stack) {
				sel[key(stack)] = Math.max(1, Math.min(item.count, stack.count));
			}
		}
	}

	function items(sel, stacks) {
		return stacks
			.filter((stack) => sel[key(stack)])
			.map((stack) => ({
				card_id: stack.card_id,
				foil: stack.foil,
				count: sel[key(stack)],
			}));
	}

	async function send() {
		sending = true;
		message = "";
		try {
			const response = await fetch(submitUrl, {
				method: "POST",
				credentials: "same-origin",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({
					target: targetName,
					give: items(give, mine),
					coins: Math.max(0, coins || 0),
					want: items(want, theirs),
					counter_of: counterId,
				}),
			});
			if (response.ok) {
				const body = await response.json();
				location.assign(body.redirect ?? "/trades");
				return;
			}
			const body = await response.json().catch(() => null);
			message = body?.message ?? labelError;
		} catch {
			message = labelError;
		} finally {
			sending = false;
		}
	}

	onMount(async () => {
		try {
			const response = await fetch(stacksUrl, { credentials: "same-origin" });
			if (!response.ok) {
				throw new Error();
			}
			const body = await response.json();
			mine = body.mine ?? [];
			theirs = body.theirs ?? [];
			balance = Number(body.coins) || 0;
			prefill(give, mine, target.dataset.prefillGive);
			prefill(want, theirs, target.dataset.prefillWant);
		} catch {
			message = labelError;
		} finally {
			loading = false;
		}
	});
</script>

<div class="trade-menu">
	<!-- Invisible ruler: the card markup is a fixed 30ch box, this is the
	     width the tiles are scaled from -->
	<span class="trade-menu-probe" bind:clientWidth={cardWidth}></span>
	{#if loading}
		<p class="trade-menu-status">{labelLoading}</p>
	{:else}
		<section class="trade-menu-sides">
			<section class="trade-menu-side">
				<div class="trade-menu-side-head">
					<h2>{labelGive}</h2>
					<label class="trade-menu-coins">
						<span>{labelCoins}</span>
						<input type="number" min="0" max={balance} bind:value={coins} oninput={capCoins} />
					</label>
				</div>
				<div class="trade-menu-deal">
					{#each givePicked as stack (key(stack))}
						<div class="trade-menu-item" style="--tile-color: {rarityColor(stack.rarity)}">
							<img class="trade-menu-thumb" src={stack.image} alt="" />
							<span class="trade-menu-item-title">{title(stack)}</span>
							<span class="trade-menu-stepper">
								<button type="button" aria-label={labelDecrease} onclick={() => step("give", stack, -1)}>−</button>
								<span class="trade-menu-count">{give[key(stack)]}</span>
								<button type="button" aria-label={labelIncrease} onclick={() => step("give", stack, 1)}>+</button>
							</span>
						</div>
					{/each}
				</div>
				<div class="trade-menu-picker" bind:clientWidth={pickerWidth}
					style="--trade-card-scale: {cardScale}">
					{#each mine as stack (key(stack))}
						<div class="trade-menu-tile" class:is-picked={give[key(stack)]} role="button" tabindex="0"
							aria-label={title(stack)} onclick={(e) => pick(e, "give", stack)}
							onkeydown={(e) => pickKey(e, "give", stack)}>
							<div class="trade-menu-card">{@html stack.html}</div>
						</div>
					{:else}
						<p class="trade-menu-empty">{labelEmpty}</p>
					{/each}
				</div>
			</section>

			<section class="trade-menu-side">
				<div class="trade-menu-side-head">
					<h2>{labelGet}</h2>
				</div>
				<div class="trade-menu-deal">
					{#each wantPicked as stack (key(stack))}
						<div class="trade-menu-item" style="--tile-color: {rarityColor(stack.rarity)}">
							<img class="trade-menu-thumb" src={stack.image} alt="" />
							<span class="trade-menu-item-title">{title(stack)}</span>
							<span class="trade-menu-stepper">
								<button type="button" aria-label={labelDecrease} onclick={() => step("want", stack, -1)}>−</button>
								<span class="trade-menu-count">{want[key(stack)]}</span>
								<button type="button" aria-label={labelIncrease} onclick={() => step("want", stack, 1)}>+</button>
							</span>
						</div>
					{/each}
				</div>
				<div class="trade-menu-picker" bind:clientWidth={pickerWidth}
					style="--trade-card-scale: {cardScale}">
					{#each theirs as stack (key(stack))}
						<div class="trade-menu-tile" class:is-picked={want[key(stack)]} role="button" tabindex="0"
							aria-label={title(stack)} onclick={(e) => pick(e, "want", stack)}
							onkeydown={(e) => pickKey(e, "want", stack)}>
							<div class="trade-menu-card">{@html stack.html}</div>
						</div>
					{:else}
						<p class="trade-menu-empty">{labelEmpty}</p>
					{/each}
				</div>
			</section>
		</section>

		<footer class="trade-menu-footer">
			{#if message}
				<p class="trade-menu-error">{message}</p>
			{/if}
			<a class="trade-menu-back" href="/trades">{labelBack}</a>
			<button type="button" class="trade-menu-send" disabled={sending} onclick={send}>
				{counterId ? labelCounter : labelSend}
			</button>
		</footer>
	{/if}
</div>

<style>
	.trade-menu {
		width: 100%;
	}

	/* Measures 30ch, the fixed card width the tiles are scaled from. Fixed so
	   the ruler can never widen the page */
	.trade-menu-probe {
		position: fixed;
		top: 0;
		right: 0;
		visibility: hidden;
		width: 30ch;
		pointer-events: none;
	}

	.trade-menu-status {
		margin: 1.5rem 0;
		color: var(--color-text-muted);
	}

	/* The two sides of the offer, one section split down the middle.
	   The generic `section` card look is reset on purpose: its `margin: auto`
	   would shrink the halves instead of letting them fill their column */
	.trade-menu-sides {
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);

		margin: 0;
		padding: 0;

		background: var(--color-bg-2);
		border: 1px solid var(--color-border-1);
		border-radius: 1rem;
		box-shadow: 0 0 10px var(--color-shadow);
	}

	.trade-menu-side {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		min-width: 0;

		margin: 0;
		padding: 1rem;

		background: none;
		border: 0;
		border-radius: 0;
		box-shadow: none;
	}

	/* The line separating the two halves */
	.trade-menu-side + .trade-menu-side {
		border-left: 1px solid var(--color-border-1);
	}

	h2 {
		margin: 0;

		font-size: 1.15rem;
		font-weight: 650;
		letter-spacing: -0.01em;
	}

	/* Heading row, on both sides alike so the deal boxes start level */
	.trade-menu-side-head {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		min-height: 2.5rem;
	}

	/* What this side puts up, filled by picking below. Three rows at most: a
	   deal with more stacks scrolls inside the box instead of growing it */
	.trade-menu-deal {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		flex: 0 0 auto;
		min-height: 7.5rem;
		max-height: 9.5rem;
		overflow-y: auto;

		padding: 0.65rem;

		background: var(--color-bg-4);
		border: 1px dashed var(--color-border-1);
		border-radius: 0.75rem;
	}

	/* One line when it fits: the stepper drops under the title when it does
	   not, so a picked stack never pushes past the deal box */
	.trade-menu-item {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.6rem;
	}

	.trade-menu-thumb {
		width: 2.4rem;
		height: 2.4rem;
		object-fit: cover;

		border: 2px solid var(--tile-color, var(--color-border-1));
		border-radius: 0.45rem;
	}

	.trade-menu-item-title {
		flex: 1 1 50%;
		min-width: 0;

		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;

		font-size: 0.95rem;
	}

	.trade-menu-stepper {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		min-width: 0;
	}

	/* The buttons give way rather than push past the deal box where the row
	   is squeezed to the bone */
	.trade-menu-stepper button {
		width: 1.75rem;
		height: 1.75rem;
		min-width: 0;
		padding: 0;

		color: var(--color-text);
		background: var(--color-bg-2);
		border: 0;
		border-radius: 0.45rem;

		font-size: 1rem;
		line-height: 1;
		cursor: pointer;

		transition: background-color 120ms ease;
	}

	.trade-menu-stepper button:hover {
		background: color-mix(in srgb, var(--color-bg-2), black 10%);
	}

	.trade-menu-count {
		min-width: 0;
		overflow: hidden;

		text-align: center;
		font-variant-numeric: tabular-nums;
	}

	/* Coins put up along the cards, on the giving side only, next to the heading */
	.trade-menu-coins {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		min-width: 0;
		margin-left: auto;

		font-size: 0.95rem;
	}

	/* The label gives way to the amount field where the row gets tight */
	.trade-menu-coins span {
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.trade-menu-coins input {
		width: 6rem;
		padding: 0.35rem 0.5rem;

		color: var(--color-text);
		background: var(--color-bg-2);
		border: 1px solid var(--color-border-1);
		border-radius: 0.5rem;

		font-variant-numeric: tabular-nums;

		/* The stepper arrows are hidden, the amount is typed or capped */
		appearance: textfield;
		-moz-appearance: textfield;
	}

	.trade-menu-coins input::-webkit-outer-spin-button,
	.trade-menu-coins input::-webkit-inner-spin-button {
		-webkit-appearance: none;
		margin: 0;
	}

	/* Inventories to pick the stacks from. Two rows of cards at most: the
	   rest of the inventory scrolls, so the trade box keeps a set size */
	.trade-menu-picker {
		--trade-card-scale: 0.6;

		display: grid;
		grid-template-columns: repeat(auto-fill, calc(30ch * var(--trade-card-scale)));
		gap: 0.75rem;
		justify-content: center;
		align-content: start;

		flex: 1 1 auto;
		min-height: 0;
		max-height: calc(2 * 30em * var(--trade-card-scale) + 0.75rem);
		overflow-y: auto;
		overflow-x: hidden;
	}

	/* A pickable stack, shown as its card render */
	.trade-menu-tile {
		position: relative;
		width: calc(30ch * var(--trade-card-scale));
		height: calc(30em * var(--trade-card-scale));

		cursor: pointer;

		transition: transform 120ms ease;
	}

	.trade-menu-tile:hover {
		transform: translateY(-1px);
	}

	.trade-menu-tile.is-picked,
	.trade-menu-tile:focus-visible {
		border-radius: calc(var(--card-border-radius) * var(--trade-card-scale));
		outline: 2px solid var(--color-accent-primary);
	}

	/* The card markup is a fixed 30ch x 30em box; scale it down to the tile */
	.trade-menu-card {
		width: 30ch;
		height: 30em;

		transform: scale(var(--trade-card-scale));
		transform-origin: top left;
	}

	/* The copy count hangs off the card corner everywhere else; in these
	   packed tiles it would stick out of the box */
	.trade-menu-card :global(.card-stack) {
		top: 0.5rem;
		right: 0.5rem;
	}

	.trade-menu-empty {
		grid-column: 1 / -1;
		margin: 0;

		color: var(--color-text-muted);
	}

	.trade-menu-footer {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.75rem;

		margin-top: 1rem;
	}

	.trade-menu-error {
		flex: 1;
		margin: 0;
		padding: 0.55rem 0.75rem;

		color: var(--color-error-3);
		background: var(--color-error-1);
		border: 1px solid var(--color-error-2);
		border-radius: 0.6rem;

		font-size: 0.95rem;
	}

	.trade-menu-back,
	.trade-menu-send {
		display: inline-flex;
		align-items: center;
		justify-content: center;

		min-height: 2.5rem;
		padding: 0.45rem 1rem;

		border: 0;
		border-radius: 0.55rem;

		font-weight: 600;
		line-height: 1.3;
		text-decoration: none;

		transition: background-color 120ms ease;
	}

	.trade-menu-back {
		color: var(--color-text);
		background: var(--color-bg-4);
	}

	.trade-menu-back:hover {
		background: color-mix(in srgb, var(--color-bg-4), black 10%);
	}

	.trade-menu-send {
		color: var(--color-text-alt);
		background: var(--color-accent-primary);
		cursor: pointer;
	}

	.trade-menu-send:hover {
		background: var(--color-accent-primary-dark);
	}

	.trade-menu-send:disabled {
		opacity: 0.6;
		cursor: default;
	}

	@media (max-width: 48rem) {
		.trade-menu-side {
			padding: 0.65rem;
		}

		/* The coin line drops under the heading where the row gets too tight.
		   Both heads keep the same height so the deal boxes start level, and
		   the titles stay top-aligned so they share a line */
		.trade-menu-side-head {
			flex-wrap: wrap;
			align-items: flex-start;
			min-height: 4.5rem;
		}

		h2 {
			font-size: 1rem;
		}

		.trade-menu-coins {
			flex: 1 1 100%;
			margin-left: 0;
		}

		.trade-menu-coins input {
			flex: 1 1 auto;
			width: auto;
			min-width: 3rem;
		}

		/* The picked stacks pack tighter in the narrow deal box */
		.trade-menu-item {
			gap: 0.45rem;
		}

		.trade-menu-thumb {
			width: 1.8rem;
			height: 1.8rem;
		}

		.trade-menu-stepper button {
			width: 1.45rem;
			height: 1.45rem;
		}
	}

	@media (max-width: 24rem) {
		/* The deal rows pack tighter still so the stepper never gets squeezed */
		.trade-menu-item {
			gap: 0.35rem;
		}

		.trade-menu-thumb {
			width: 1.5rem;
			height: 1.5rem;
		}

		.trade-menu-stepper {
			gap: 0.25rem;
		}

		.trade-menu-stepper button {
			width: 1.25rem;
			height: 1.25rem;
		}
	}
</style>
