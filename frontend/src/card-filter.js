/*
 * Filters and sorts a grid of card tiles in the page, without navigating, so
 * a selection built in a picker survives. Shared by the tradeup and foil
 * roulette pickers.
 *
 * Every control is optional:
 *  - `search`: text input, keeps the tiles whose name or collection holds it
 *  - `collection`: select of collection IDs, "" for all of them
 *  - `rarity`: select of rarities, "" for all of them
 *  - `sort`: select of `number`, `name`, `name-desc`, `rarity` or `count`
 *  - `noMatch`: element shown when the filters hide every tile
 *  - `pagers`: elements the page links are drawn in, above and below the
 *    grid. Each provides `data-page-size`, `data-label-nav`,
 *    `data-label-previous` and `data-label-next`. Without them every tile
 *    shows at once.
 *
 * `read(tile)` gives what a tile is filtered and sorted on:
 * `{ title, collection, collectionId, rarity, number, count }`.
 *
 * Paging happens here too, not by navigating: a picker's selection would not
 * survive a new page. Changing a filter goes back to the first page.
 *
 * The filters and page are mirrored in the URL (`q`, `collection`, `rarity`,
 * `sort`, `p`), so a refresh or a shared link lands on them. Returns a
 * cleanup function.
 */

// Highest rarity first, unknown rarities last
const RARITY_ORDER = ["legendary", "rare", "common"];

function rarityRank(rarity) {
	const i = RARITY_ORDER.indexOf(rarity);
	return i < 0 ? RARITY_ORDER.length : i;
}

// Case and accents are ignored when searching
function fold(text) {
	return text.normalize("NFD").replaceAll(/\p{Diacritic}/gu, "").toLowerCase();
}

function find(selector) {
	return selector ? document.querySelector(selector) : null;
}

// Page numbers around the current one, like the server-rendered paginator
const PAGE_SPREAD = 2;

export function attachCardFilter({ tiles, read, search, collection, rarity, sort, noMatch, pagers }) {
	const searchInput = find(search);
	const collectionSelect = find(collection);
	const raritySelect = find(rarity);
	const sortSelect = find(sort);
	const noMatchElement = find(noMatch);
	const pagerElements = pagers ? [...document.querySelectorAll(pagers)] : [];
	const pageSize = Number(pagerElements[0]?.dataset.pageSize ?? 0);
	let page = 1;

	const all = [...document.querySelectorAll(tiles)];
	// The order the page came in, ties fall back to it
	const original = new Map(all.map((tile, i) => [tile, i]));
	const collator = new Intl.Collator(document.documentElement.lang || undefined, { sensitivity: "base" });

	function compare(order, a, b) {
		const x = read(a);
		const y = read(b);
		switch (order) {
			case "name":
				return collator.compare(x.title, y.title);
			case "name-desc":
				return collator.compare(y.title, x.title);
			case "rarity":
				return rarityRank(x.rarity) - rarityRank(y.rarity);
			case "count":
				return y.count - x.count;
			case "number":
				return x.number - y.number;
		}
		return 0;
	}

	function reorder(order) {
		const sorted = all.toSorted((a, b) => compare(order, a, b) || original.get(a) - original.get(b));
		// Tiles all live in one grid; the no-match message stays last
		for (const tile of sorted) {
			tile.parentElement?.insertBefore(tile, noMatchElement?.parentElement === tile.parentElement ? noMatchElement : null);
		}
	}

	// Draws the page links in every pager. Links point at the page they open,
	// so they still work as links; a click is caught to stay on the page.
	function drawPagers(pages) {
		for (const pager of pagerElements) {
			pager.replaceChildren();
			if (pages <= 1) continue;

			const nav = document.createElement("nav");
			nav.className = "paginator";
			nav.setAttribute("aria-label", pager.dataset.labelNav ?? "");
			const list = document.createElement("ul");
			nav.append(list);

			function item(target, text, label) {
				const li = document.createElement("li");
				let element;
				if (target === page && !label) {
					element = document.createElement("span");
					element.className = "is-current";
					element.setAttribute("aria-current", "page");
					// Focused after paging from the keyboard
					element.tabIndex = -1;
				} else if (target < 1 || target > pages) {
					element = document.createElement("span");
					element.setAttribute("aria-disabled", "true");
				} else {
					element = document.createElement("a");
					const url = new URL(location.href);
					url.searchParams.set("p", String(target));
					element.href = url.toString();
					element.dataset.page = String(target);
				}
				if (label) element.setAttribute("aria-label", label);
				element.textContent = text;
				li.append(element);
				list.append(li);
			}

			item(page - 1, "‹", pager.dataset.labelPrevious);
			for (let n = Math.max(1, page - PAGE_SPREAD); n <= Math.min(pages, page + PAGE_SPREAD); n++) {
				item(n, String(n));
			}
			item(page + 1, "›", pager.dataset.labelNext);
			pager.append(nav);
		}
	}

	function onPagerClick(e) {
		if (!(e.target instanceof Element)) return;
		const link = e.target.closest("a[data-page]");
		if (!link) return;
		e.preventDefault();
		const pager = link.closest("[data-page-size]");
		const arrow = link.getAttribute("aria-label");
		// A click from the keyboard has no pointer detail
		const fromKeyboard = e.detail === 0;

		page = Number(link.dataset.page);
		apply({ keepPage: true });

		// Back to the top of the grid when it scrolled out of view
		const anchor = pagerElements[0];
		if (anchor && anchor.getBoundingClientRect().top < 0) {
			anchor.scrollIntoView({
				block: "start",
				behavior: matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth",
			});
		}
		// The links were redrawn: keyboard users stay on the arrow they used,
		// or land on the new current page
		if (fromKeyboard && pager) {
			const next = (arrow && pager.querySelector(`a[aria-label="${CSS.escape(arrow)}"]`))
				|| pager.querySelector(".is-current");
			next?.focus();
		}
	}

	function apply({ keepPage = false } = {}) {
		const query = searchInput?.value.trim() ?? "";
		const folded = fold(query);
		const collectionValue = collectionSelect?.value ?? "";
		const rarityValue = raritySelect?.value ?? "";
		const sortValue = sortSelect?.value ?? "";

		if (sortSelect) reorder(sortValue);
		// In the order they stand in once sorted, so pages follow the sort
		const ordered = [...document.querySelectorAll(tiles)];
		const matches = ordered.filter((tile) => {
			const info = read(tile);
			return (collectionValue === "" || info.collectionId === collectionValue)
				&& (rarityValue === "" || info.rarity === rarityValue)
				&& (folded === "" || fold(`${info.title} ${info.collection}`).includes(folded));
		});
		if (noMatchElement) noMatchElement.hidden = matches.length > 0;

		const pages = pageSize > 0 ? Math.max(1, Math.ceil(matches.length / pageSize)) : 1;
		if (!keepPage) page = 1;
		page = Math.min(Math.max(page, 1), pages);
		const shown = new Set(pageSize > 0 ? matches.slice((page - 1) * pageSize, page * pageSize) : matches);
		for (const tile of all) {
			tile.hidden = !shown.has(tile);
		}
		drawPagers(pages);

		// A refresh lands on the same filters. The sort select's first option
		// is the page's own order, it stays out of the URL.
		const url = new URL(location.href);
		const params = [
			["q", query],
			["collection", collectionValue],
			["rarity", rarityValue],
			["sort", sortSelect && sortSelect.selectedIndex > 0 ? sortValue : ""],
			["p", page > 1 ? String(page) : ""],
		];
		for (const [name, value] of params) {
			if (value === "") url.searchParams.delete(name);
			else url.searchParams.set(name, value);
		}
		history.replaceState(history.state, "", url);
	}

	// Filtering happens as the text is typed, Enter would leave the page
	function onSearchKeyDown(e) {
		if (e.key === "Enter") e.preventDefault();
	}

	// Listeners get an event as first argument, not options
	const onChange = () => apply();
	searchInput?.addEventListener("input", onChange);
	searchInput?.addEventListener("keydown", onSearchKeyDown);
	for (const select of [collectionSelect, raritySelect, sortSelect]) {
		select?.addEventListener("change", onChange);
	}
	for (const pager of pagerElements) {
		pager.addEventListener("click", onPagerClick);
	}

	// A shared link can arrive already filtered
	const params = new URL(location.href).searchParams;
	page = Math.max(1, Number.parseInt(params.get("p") ?? "1", 10) || 1);
	function restore(element, name) {
		const value = params.get(name);
		if (!element || !value) return;
		if (element instanceof HTMLSelectElement && ![...element.options].some((o) => o.value === value)) return;
		element.value = value;
	}
	restore(searchInput, "q");
	restore(collectionSelect, "collection");
	restore(raritySelect, "rarity");
	restore(sortSelect, "sort");
	apply({ keepPage: true });

	return () => {
		searchInput?.removeEventListener("input", onChange);
		searchInput?.removeEventListener("keydown", onSearchKeyDown);
		for (const select of [collectionSelect, raritySelect, sortSelect]) {
			select?.removeEventListener("change", onChange);
		}
		for (const pager of pagerElements) {
			pager.removeEventListener("click", onPagerClick);
		}
	};
}
