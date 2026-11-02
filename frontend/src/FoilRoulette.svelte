<!--
    Builds a foil roll: the first card clicked in the inventory is the one
    rolled, every card clicked after it is sacrificed to raise the odds.
    Clicking the rolled card or a sacrifice in the sidebar takes it back out.
    The roll is submitted as a `target` field and one `card` field per
    sacrifice, after a confirmation: a lost roll destroys the card.

    Cards must provide `data-foil-id`, `data-foil-foil` (the pattern, "" for a
    plain copy), `data-foil-count` (copies owned), `data-foil-rarity`,
    `data-foil-title`, `data-foil-collection-id` and `data-foil-collection`.
    Only plain copies can be rolled; any copy can be sacrificed. Each eligible
    tile wears a `.foil-bonus` badge with the chance it gives, and stacks with
    sacrificed copies a `.tradeup-taken` pill.

    The mount target provides the odds (`data-odds`, rarity to `{base, bonus}`
    in `data-scale` units), the chance cap (`data-cap`), the most sacrifices a
    roll takes (`data-max`), the rarities in the order the odds table lists
    them (`data-rarities`, `[{value, label}]`) and the labels, `data-label-foil`
    and `data-label-copies` naming the cards as the inventory does.
    `{title}` and `{chance}` in labels are replaced here.

    Filtering, sorting and paging are left to card-filter.js, on
    `#foil-search`, `#foil-collection-filter`, `#foil-rarity-filter`,
    `#foil-sort` and the `.foil-pager` elements.
-->

<script>
	import { onMount, tick } from "svelte";
	import { attachCardFilter } from "./card-filter.js";
	import { cardName, makeCardButton } from "./card-name.js";

	let { target: mount } = $props();

	const CARD_SELECTOR = ".foil-card[data-foil-id]";
	const TILE_SELECTOR = `${CARD_SELECTOR} .card-container`;
	// Pointer travel (px) after which a press counts as a tilt drag, not a click
	const DRAG_THRESHOLD = 6;

	function json(value, fallback) {
		try {
			return JSON.parse(value ?? "");
		} catch {
			return fallback;
		}
	}

	const odds = $derived(json(mount.dataset.odds, {}));
	const rarities = $derived(json(mount.dataset.rarities, []));
	const rarityName = (value) => rarities.find((r) => r.value === value)?.label ?? value;
	const cap = $derived(Number(mount.dataset.cap ?? 0));
	const scale = $derived(Number(mount.dataset.scale ?? 10000));
	const max = $derived(Number(mount.dataset.max ?? 0));
	const label = (name) => mount.dataset[name] ?? "";
	const cardLabels = $derived({ foil: label("labelFoil"), copies: label("labelCopies") });

	const lang = document.documentElement.lang || undefined;
	const percent = new Intl.NumberFormat(lang, { style: "percent", maximumFractionDigits: 2 });
	const signed = new Intl.NumberFormat(lang, { style: "percent", maximumFractionDigits: 2, signDisplay: "always" });
	const formatChance = (points) => percent.format(points / scale);
	const formatBonus = (points) => signed.format(points / scale);

	// Card rolled for a foil, null until one is picked
	let rolled = $state(null);
	// Copies sacrificed, in the order they were added
	let sacrifices = $state([]);

	let base = $derived(rolled ? (odds[rolled.rarity]?.base ?? 0) : 0);
	let bonus = $derived(sacrifices.reduce((sum, card) => sum + (odds[card.rarity]?.bonus ?? 0), 0));
	// What the sacrifices would bring past the cap is lost
	let chance = $derived(rolled ? Math.min(base + bonus, cap) : 0);
	let capped = $derived(rolled !== null && base + bonus >= cap);
	let full = $derived(sacrifices.length >= max);

	let dialog;
	let form;
	let confirmed = false;
	// Focused when the button that had the focus goes away: the sections the
	// rolled card and the sacrifices leave
	let rolledSection = $state();
	let sacrificesSection = $state();

	function cardOf(tile) {
		return {
			id: tile.dataset.foilId,
			foil: tile.dataset.foilFoil ?? "",
			count: Number(tile.dataset.foilCount ?? 0),
			rarity: tile.dataset.foilRarity ?? "",
			title: tile.dataset.foilTitle ?? "",
		};
	}

	const same = (a, b) => a.id === b.id && a.foil === b.foil;

	// Copies of a stack still free: not rolled, not sacrificed
	function free(card) {
		let n = card.count;
		if (rolled && same(rolled, card)) n--;
		for (const sacrifice of sacrifices) {
			if (same(sacrifice, card)) n--;
		}
		return n;
	}

	function sacrificed(card) {
		return sacrifices.filter((sacrifice) => same(sacrifice, card)).length;
	}

	// The first pick is rolled, plain copies only; the next are sacrifices,
	// until the chance is capped
	function canTake(card) {
		if (!(card.rarity in odds) || free(card) <= 0) return false;
		if (!rolled) return card.foil === "";
		return !capped && !full;
	}

	function add(tile) {
		const card = cardOf(tile);
		if (!canTake(card)) return;
		if (rolled) sacrifices.push(card);
		else rolled = card;
	}

	// The rolled card's button is gone: the focus goes to its section, which
	// now says to pick one
	async function unroll() {
		rolled = null;
		await tick();
		rolledSection.focus();
	}

	// The focused slot is gone: the focus goes to the slot taking its place,
	// else the one before, else the section
	async function remove(index) {
		sacrifices.splice(index, 1);
		await tick();
		const slots = sacrificesSection.querySelectorAll(".foil-slot");
		(slots[index] ?? slots[index - 1] ?? sacrificesSection).focus();
	}

	// The clear button is disabled once the roll is empty: the focus goes
	// back to its start, picking the card to roll
	async function clear() {
		rolled = null;
		sacrifices = [];
		await tick();
		rolledSection.focus();
	}

	// Name of a button taking a card back out, with what it shows of it
	function removeName(card) {
		return [`${label("labelRemove")} ${card.title}`, rarityName(card.rarity), card.foil ? label("labelFoil") : ""]
			.filter(Boolean)
			.join(", ");
	}

	// What a tile gives, as its badge shows it: its chance as the card
	// rolled, or the bonus it adds as a sacrifice once one is
	function chanceOf(card) {
		const odd = odds[card.rarity];
		if (!odd) return "";
		return rolled ? formatBonus(odd.bonus) : formatChance(odd.base);
	}

	// A tile's name: the card as the inventory names it, then what the
	// roulette makes of it, the badges being only seen
	function tileName(tile, card, { blocked, isRolled, count }) {
		const chanceLabel = rolled ? label("labelPerSacrifice") : label("labelChance");
		return [
			cardName(tile, { ...cardLabels, count: card.count }),
			isRolled ? label("labelRolled") : "",
			blocked ? "" : `${chanceLabel}: ${chanceOf(card)}`,
			count > 0 ? `${label("labelSacrifices")}: ${count}` : "",
		]
			.filter(Boolean)
			.join(", ");
	}

	function fill(text, values) {
		return Object.entries(values).reduce((s, [key, value]) => s.replaceAll(`{${key}}`, value), text);
	}

	function onSubmit(e) {
		if (confirmed) return;
		e.preventDefault();
		if (rolled) dialog.showModal();
	}

	function roll() {
		confirmed = true;
		dialog.close();
		form.requestSubmit();
	}

	onMount(() => {
		form = mount.closest("form");
		form?.addEventListener("submit", onSubmit);

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
		// A held key adds the card once, not until the roll is full
		function onKeyDown(e) {
			if (e.key !== "Enter" && e.key !== " ") return;
			if (!(e.target instanceof Element) || !e.target.matches(TILE_SELECTOR)) return;

			e.preventDefault();
			if (e.repeat) return;
			add(e.target.closest(CARD_SELECTOR));
		}

		// Cards are picked by clicking, make them reachable and announced as
		// buttons, out of the tab order of their links. The effect below
		// names them.
		for (const tile of document.querySelectorAll(TILE_SELECTOR)) {
			makeCardButton(tile, cardLabels);
		}

		document.addEventListener("pointerdown", onPointerDown, { passive: true });
		document.addEventListener("click", onClick);
		document.addEventListener("keydown", onKeyDown);

		const detachFilter = attachCardFilter({
			tiles: CARD_SELECTOR,
			read: (tile) => ({
				title: tile.dataset.foilTitle ?? "",
				collection: tile.dataset.foilCollection ?? "",
				collectionId: tile.dataset.foilCollectionId ?? "",
				rarity: tile.dataset.foilRarity ?? "",
				number: Number(tile.dataset.foilId ?? 0),
				count: Number(tile.dataset.foilCount ?? 0),
			}),
			search: "#foil-search",
			collection: "#foil-collection-filter",
			rarity: "#foil-rarity-filter",
			sort: "#foil-sort",
			noMatch: "#foil-no-match",
			pagers: ".foil-pager",
		});

		return () => {
			form?.removeEventListener("submit", onSubmit);
			document.removeEventListener("pointerdown", onPointerDown);
			document.removeEventListener("click", onClick);
			document.removeEventListener("keydown", onKeyDown);
			detachFilter();
		};
	});

	// Ineligible tiles are inert; eligible ones wear the chance they give, the
	// rolled one stands out, and sacrificed stacks wear how many copies went.
	// Their names say the same. After onMount, which made them buttons.
	$effect(() => {
		for (const card of document.querySelectorAll(CARD_SELECTOR)) {
			const info = cardOf(card);
			const blocked = !canTake(info);
			const isRolled = rolled !== null && same(rolled, info);
			const count = sacrificed(info);
			card.classList.toggle("tradeup-card-blocked", blocked && !isRolled);
			card.classList.toggle("foil-card-rolled", isRolled);

			const tile = card.querySelector(".card-container");
			if (!tile) continue;
			// On the button itself, so screen readers tell it does nothing
			tile.setAttribute("aria-disabled", blocked ? "true" : "false");
			tile.setAttribute("aria-label", tileName(tile, info, { blocked, isRolled, count }));

			let badge = tile.querySelector(":scope > .foil-bonus");
			if (blocked) {
				badge?.remove();
			} else {
				if (!badge) {
					badge = document.createElement("span");
					badge.className = "foil-bonus";
					badge.setAttribute("aria-hidden", "true");
					tile.append(badge);
				}
				badge.textContent = chanceOf(info);
			}

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
</script>

<div class="foil-roulette">
	<section class="foil-section" aria-labelledby="foil-rolled-title" tabindex="-1" bind:this={rolledSection}>
		<h2 class="foil-section-title" id="foil-rolled-title">{label("labelRolled")}</h2>
		{#if rolled}
			<button type="button" class="foil-rolled" aria-label={removeName(rolled)} onclick={unroll}>
				<span class="foil-thumb" style="background-image: url('/cards/{rolled.id}/thumbnail')"></span>
				<span class="foil-rolled-text">
					<span class="foil-rolled-name">{rolled.title}</span>
					<span class="foil-rolled-rarity">{rarityName(rolled.rarity)}</span>
				</span>
			</button>
		{:else}
			<p class="foil-hint">{label("labelPickRolled")}</p>
		{/if}
	</section>

	<section class="foil-section foil-odds" aria-labelledby="foil-chance-title">
		<h2 class="foil-section-title" id="foil-chance-title">{label("labelChance")}</h2>
		<p class="foil-chance" aria-live="polite">
			<span class="foil-chance-value">{rolled ? formatChance(chance) : "–"}</span>
			{#if capped}<span class="foil-chance-capped">{label("labelCapped")}</span>{/if}
		</p>
		<div
			class="foil-meter"
			role="meter"
			aria-labelledby="foil-chance-title"
			aria-valuemin="0"
			aria-valuemax={cap / scale * 100}
			aria-valuenow={chance / scale * 100}
			aria-valuetext={formatChance(chance)}
		>
			<span class="foil-meter-fill" style="width: {cap ? (chance / cap) * 100 : 0}%"></span>
		</div>
		{#if rolled}
			<dl class="foil-breakdown">
				<div>
					<dt>{label("labelBase")}</dt>
					<dd>{formatChance(base)}</dd>
				</div>
				<div>
					<dt>{label("labelBonus")}</dt>
					<dd>{formatBonus(Math.min(bonus, cap - base))}</dd>
				</div>
			</dl>
		{/if}
		<details class="foil-odds-details">
		<summary>{label("labelOdds")}</summary>
		<table class="foil-table">
			<caption class="sr-only">{label("labelOdds")}</caption>
			<thead>
				<tr>
					<th scope="col">{label("labelRarity")}</th>
					<th scope="col">{label("labelBase")}</th>
					<th scope="col">{label("labelPerSacrifice")}</th>
				</tr>
			</thead>
			<tbody>
				{#each rarities as rarity (rarity.value)}
					{#if odds[rarity.value]}
						<tr>
							<th scope="row">{rarity.label}</th>
							<td>{formatChance(odds[rarity.value].base)}</td>
							<td>{formatBonus(odds[rarity.value].bonus)}</td>
						</tr>
					{/if}
				{/each}
			</tbody>
		</table>
		</details>
	</section>

	<section class="foil-section" aria-labelledby="foil-sacrifices-title" tabindex="-1" bind:this={sacrificesSection}>
		<h2 class="foil-section-title" id="foil-sacrifices-title">
			{label("labelSacrifices")} <span class="foil-count">{sacrifices.length}</span>
		</h2>
		{#if sacrifices.length === 0}
			<p class="foil-hint">{rolled ? label("labelPickSacrifices") : label("labelSacrificesLater")}</p>
		{:else}
			<div class="foil-slots" role="group" aria-labelledby="foil-sacrifices-title">
				{#each sacrifices as card, i (i)}
					<button
						type="button"
						class="foil-slot"
						title={card.title}
						aria-label={removeName(card)}
						style="background-image: url('/cards/{card.id}/thumbnail')"
						onclick={() => remove(i)}
					>
						<span class="foil-slot-bonus" aria-hidden="true">{formatBonus(odds[card.rarity]?.bonus ?? 0)}</span>
					</button>
				{/each}
			</div>
		{/if}
	</section>

	{#if rolled}
		<input type="hidden" name="target" value={rolled.id} />
	{/if}
	{#each sacrifices as card, i (i)}
		<input type="hidden" name="card" value={card.foil ? `${card.id}:${card.foil}` : `${card.id}`} />
	{/each}

	<!-- Pinned to the bottom of the sidebar, or of the screen on small ones -->
	<div class="foil-actions">
		<p class="foil-actions-chance">
			<span class="foil-actions-label">{label("labelChance")}</span>
			<strong>{rolled ? formatChance(chance) : "–"}</strong>
		</p>
		<button type="button" class="foil-clear" onclick={clear} disabled={!rolled && sacrifices.length === 0}>
			{label("labelClear")}
		</button>
		<button type="submit" class="foil-submit" disabled={!rolled}>{label("labelSubmit")}</button>
	</div>
</div>

<!-- A lost roll destroys the card: said before rolling -->
<dialog
	class="popup inventory-confirm"
	aria-labelledby="foil-confirm-title"
	aria-describedby="foil-confirm-text"
	bind:this={dialog}
>
	<div class="inventory-confirm-content">
		<h2 id="foil-confirm-title">{label("labelConfirmTitle")}</h2>
		<p id="foil-confirm-text">
			{rolled ? fill(label("labelConfirmText"), { title: rolled.title, chance: formatChance(chance) }) : ""}
		</p>
		{#if sacrifices.length > 0}
			<p class="inventory-confirm-note">{fill(label("labelConfirmSacrifices"), { count: sacrifices.length })}</p>
		{/if}
		<div class="inventory-confirm-actions">
			<button type="button" class="inventory-confirm-cancel" onclick={() => dialog.close()}>
				{label("labelCancel")}
			</button>
			<button type="button" onclick={roll}>{label("labelSubmit")}</button>
		</div>
	</div>
</dialog>

<style>
	.foil-roulette {
		display: flex;
		flex-direction: column;
		gap: 1.25rem;

		padding: 1.25rem;

		background: var(--color-bg-2);
		border: 1px solid var(--color-border-1);
		border-radius: 1rem;
	}

	/* Parts of the panel, not cards of their own: the global section look is reset */
	.foil-section {
		display: flex;
		flex-direction: column;
		gap: 0.6rem;

		margin: 0;
		padding: 0;

		background: none;
		border: 0;
		border-radius: 0;
		box-shadow: none;
	}

	.foil-section + .foil-section {
		padding-top: 1rem;

		border-top: 1px solid var(--color-border-1);
	}

	.foil-section-title {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin: 0;

		color: var(--color-text-muted);
		font-size: 0.875rem;
		font-weight: 650;
		letter-spacing: 0.02em;
		text-transform: uppercase;
	}

	.foil-count {
		min-width: 1.5rem;
		padding: 0 0.4rem;

		color: var(--color-text);
		background: var(--color-bg-4);
		border-radius: 999px;

		font-variant-numeric: tabular-nums;
		text-align: center;
		text-transform: none;
	}

	.foil-hint {
		margin: 0;

		color: var(--color-text-muted);
		font-size: 0.9rem;
	}

	.foil-rolled {
		display: flex;
		align-items: center;
		gap: 0.75rem;

		padding: 0.5rem;

		color: var(--color-text);
		background: var(--color-bg-4);
		border: 1px solid var(--color-border-1);
		border-radius: 0.65rem;

		text-align: left;
		cursor: pointer;

		&:hover {
			border-color: var(--color-accent-primary);
		}

		&:focus-visible {
			outline: 3px solid var(--color-accent-secondary);
			outline-offset: 2px;
		}
	}

	.foil-thumb {
		flex: none;
		width: 3rem;
		aspect-ratio: 1;

		background-size: cover;
		background-position: center;
		border-radius: 0.45rem;
	}

	.foil-rolled-text {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}

	.foil-rolled-name {
		overflow: hidden;

		font-weight: 650;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.foil-rolled-rarity {
		color: var(--color-text-muted);
		font-size: 0.85rem;
	}

	.foil-chance {
		display: flex;
		align-items: baseline;
		flex-wrap: wrap;
		gap: 0.5rem;
		margin: 0;
	}

	.foil-chance-value {
		font-size: 2rem;
		font-variant-numeric: tabular-nums;
		font-weight: 700;
		letter-spacing: -0.02em;
		line-height: 1;
	}

	.foil-chance-capped {
		color: var(--color-text-muted);
		font-size: 0.85rem;
		font-weight: 600;
	}

	.foil-meter {
		overflow: hidden;

		height: 0.5rem;

		background: var(--color-bg-4);
		border-radius: 999px;
	}

	.foil-meter-fill {
		display: block;
		height: 100%;

		background: linear-gradient(90deg, var(--color-accent-primary), var(--color-accent-secondary));
		border-radius: inherit;

		transition: width 200ms ease;
	}

	.foil-breakdown {
		display: flex;
		gap: 1.25rem;
		margin: 0;

		font-size: 0.9rem;

		div {
			display: flex;
			gap: 0.35rem;
		}

		dt {
			color: var(--color-text-muted);
		}

		dd {
			margin: 0;

			font-variant-numeric: tabular-nums;
			font-weight: 650;
		}
	}

	.foil-table {
		width: 100%;

		border-collapse: collapse;
		font-size: 0.85rem;
		font-variant-numeric: tabular-nums;

		th,
		td {
			padding: 0.25rem 0.35rem;

			border-bottom: 1px solid var(--color-border-1);
			text-align: right;
		}

		th:first-child {
			text-align: left;
		}

		thead th {
			color: var(--color-text-muted);
			font-weight: 600;
		}

		tbody th {
			font-weight: 600;
		}
	}

	.foil-slots {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(2.75rem, 1fr));
		gap: 0.5rem;
	}

	.foil-slot {
		position: relative;

		aspect-ratio: 1;
		padding: 0;

		background-color: var(--color-bg-4);
		background-size: cover;
		background-position: center;
		border: 1px solid var(--color-border-1);
		border-radius: 0.45rem;

		cursor: pointer;

		&:hover {
			border-color: var(--color-accent-primary);
		}

		&:focus-visible {
			outline: 3px solid var(--color-accent-secondary);
			outline-offset: 2px;
		}
	}

	.foil-slot-bonus {
		position: absolute;
		right: 0;
		bottom: 0;
		left: 0;

		padding: 0.05rem 0;

		color: var(--color-text-alt);
		background: color-mix(in srgb, var(--color-accent-primary) 85%, transparent);
		border-radius: 0 0 0.4rem 0.4rem;

		font-size: 0.65rem;
		font-variant-numeric: tabular-nums;
		font-weight: 700;
	}

	.foil-odds-details summary {
		color: var(--color-text-muted);
		font-size: 0.875rem;
		font-weight: 600;

		cursor: pointer;
	}

	.foil-odds-details[open] summary {
		margin-bottom: 0.4rem;
	}

	/* Pinned to the bottom of the scrolling sidebar, past the padding it
	   keeps for focus rings, so nothing shows under the bar */
	.foil-actions {
		position: sticky;
		bottom: -4px;
		z-index: 1;

		/* The chance on a row of its own, the two buttons sharing the next */
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

	.foil-actions-chance {
		grid-column: 1 / -1;

		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 0.75rem;
		margin: 0;

		font-variant-numeric: tabular-nums;
		line-height: 1.2;

		strong {
			font-size: 1.15rem;
			white-space: nowrap;
		}
	}

	.foil-actions-label {
		min-width: 0;

		color: var(--color-text-muted);
		font-size: 0.875rem;
		font-weight: 600;
	}

	/* Labels never break inside a word: the buttons share the row instead */
	.foil-clear,
	.foil-submit {
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

	.foil-clear {
		color: var(--color-text);
		background: var(--color-bg-4);

		&:hover:not(:disabled) {
			background: color-mix(in srgb, var(--color-bg-4), black 10%);
		}
	}

	.foil-submit {
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

	.foil-clear:disabled,
	.foil-submit:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	/* The sidebar comes after the grid: the bar is fixed to the screen instead */
	@media (max-width: 900px) {
		.foil-actions {
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
		.foil-actions {
			grid-template-columns: 1fr auto auto;
		}

		.foil-actions-chance {
			grid-column: auto;
			justify-content: flex-start;
		}

		.foil-clear,
		.foil-submit {
			min-width: 7rem;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.foil-meter-fill {
			transition: none;
		}
	}
</style>
