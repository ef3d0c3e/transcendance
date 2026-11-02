/*
 * Filters, sorts and pages a grid of card tiles in the page, without
 * navigating, so a selection built in a picker survives. Shared by the
 * tradeup and foil roulette pickers.
 *
 * Every control is optional:
 *  - `search`: text input, keeps the tiles whose name or collection holds it
 *  - `collection`: select of collection IDs, '' for all of them
 *  - `rarity`: select of rarities, '' for all of them
 *  - `sort`: select of `number`, `name`, `name-desc`, `rarity`, `count` or
 *    `grouped`
 *  - `noMatch`: element shown when the filters hide every tile
 *  - `status`: live region telling screen readers what a filter change
 *    kept, from its `data-label-results` (`{count}` is replaced) or the
 *    `noMatch` text
 *  - `pagers`: elements the page links are drawn in, above and below the
 *    grid. Each provides `data-page-size`, `data-label-nav`,
 *    `data-label-previous` and `data-label-next`. Without them every tile
 *    shows at once.
 *
 * `read(tile)` gives what a tile is filtered and sorted on:
 * `{ title, collection, collectionId, rarity, number, count, order }`, where
 * `order` is the tile's place in an alternative order of the page's own, the
 * one the `grouped` sort brings back (the tradeup's collection-and-rarity).
 *
 * Paging happens here too, not by navigating: a picker's selection would not
 * survive a new page. Changing a filter goes back to the first page.
 *
 * The filters and page are mirrored in the URL (`q`, `collection`, `rarity`,
 * `sort`, `p`), so a refresh or a shared link lands on them. Returns a
 * cleanup function.
 */

// Highest rarity first, unknown rarities last
const RARITY_ORDER = ['legendary', 'rare', 'common'];

// Page numbers around the current one, like the server-rendered paginator
const PAGE_SPREAD = 2;

// Delay (ms) after the last filter change before its results are told, so
// typing in the search isn't told at every key
const ANNOUNCE_DELAY = 500;

function rarityRank(rarity) {
	const index = RARITY_ORDER.indexOf(rarity);
	return index === -1 ? RARITY_ORDER.length : index;
}

// Case and accents are ignored when searching
function fold(text) {
	return text.normalize('NFD').replaceAll(/\p{Diacritic}/gu, '').toLowerCase();
}

function find(selector) {
	return selector ? document.querySelector(selector) : undefined;
}

// How two tiles compare under each sort, 0 keeps the page's own order
const COMPARATORS = {
	'name': (collator, first, second) => collator.compare(first.title, second.title),
	'name-desc': (collator, first, second) => collator.compare(second.title, first.title),
	'rarity': (_, first, second) => rarityRank(first.rarity) - rarityRank(second.rarity),
	'count': (_, first, second) => second.count - first.count,
	'number': (_, first, second) => first.number - second.number,
	'grouped': (_, first, second) => (first.order ?? 0) - (second.order ?? 0),
};

function compareTiles(state, order, first, second) {
	const compare = COMPARATORS[order];
	const result = compare ? compare(state.collator, state.read(first), state.read(second)) : 0;
	return result || state.original.get(first) - state.original.get(second);
}

// Tiles all live in one grid; the no-match message stays last
function reorder(state, order) {
	const sorted = state.tiles.toSorted((first, second) => compareTiles(state, order, first, second));
	const { noMatch } = state.controls;
	for (const tile of sorted) {
		if (noMatch?.parentElement === tile.parentElement) {
			noMatch.before(tile);
		} else {
			tile.parentElement?.append(tile);
		}
	}
}

function readFilters({ controls }) {
	const query = controls.search?.value.trim() ?? '';
	return {
		query,
		folded: fold(query),
		collection: controls.collection?.value ?? '',
		rarity: controls.rarity?.value ?? '',
		sort: controls.sort?.value ?? '',
	};
}

function matchesFilters(info, filters) {
	return (filters.collection === '' || info.collectionId === filters.collection)
		&& (filters.rarity === '' || info.rarity === filters.rarity)
		&& (filters.folded === '' || fold(`${info.title} ${info.collection}`).includes(filters.folded));
}

// The element of one entry of the page links: the current page, a link, or a
// disabled arrow, drawn like templates/partials/paginator.html draws them
function pageElement(state, target, label) {
	if (!label && target === state.page) {
		const current = document.createElement('span');
		current.className = 'is-current';
		current.setAttribute('aria-current', 'page');
		// Focused after paging from the keyboard
		current.tabIndex = -1;
		return current;
	}
	// No page before or after: only seen, screen readers skip it
	if (target < 1 || target > state.pages) {
		const disabled = document.createElement('span');
		disabled.className = 'is-disabled';
		disabled.setAttribute('aria-hidden', 'true');
		return disabled;
	}
	// A real link, so it still works as one; a click is caught to stay on the page
	const link = document.createElement('a');
	const url = new URL(location.href);
	url.searchParams.set('p', String(target));
	link.href = url.href;
	link.dataset.page = String(target);
	return link;
}

function pageItem(state, target, text, label) {
	const element = pageElement(state, target, label);
	// The arrows' name, on links only: a span can't be named
	if (label && element instanceof HTMLAnchorElement) {
		element.setAttribute('aria-label', label);
	}
	element.textContent = text;

	const item = document.createElement('li');
	item.append(element);
	return item;
}

function drawPager(state, pager) {
	pager.replaceChildren();
	if (state.pages <= 1) {
		return;
	}

	const list = document.createElement('ul');
	list.append(pageItem(state, state.page - 1, '‹', pager.dataset.labelPrevious));
	const last = Math.min(state.pages, state.page + PAGE_SPREAD);
	for (let target = Math.max(1, state.page - PAGE_SPREAD); target <= last; target++) {
		list.append(pageItem(state, target, String(target)));
	}
	list.append(pageItem(state, state.page + 1, '›', pager.dataset.labelNext));

	const nav = document.createElement('nav');
	nav.className = 'paginator';
	nav.setAttribute('aria-label', pager.dataset.labelNav ?? '');
	nav.append(list);
	pager.append(nav);
}

// A refresh lands on the same filters. The sort select's first option is the
// page's own order, it stays out of the URL.
function syncUrl(state, filters) {
	const url = new URL(location.href);
	const { sort } = state.controls;
	const parameters = [
		['q', filters.query],
		['collection', filters.collection],
		['rarity', filters.rarity],
		['sort', sort && sort.selectedIndex > 0 ? filters.sort : ''],
		['p', state.page > 1 ? String(state.page) : ''],
	];
	for (const [name, value] of parameters) {
		if (value === '') {
			url.searchParams.delete(name);
		} else {
			url.searchParams.set(name, value);
		}
	}
	history.replaceState(history.state, '', url);
}

// Shows the tiles of the current page among those the filters keep, in the
// order the sort puts them
function apply(state, shouldKeepPage = false) {
	const filters = readFilters(state);
	if (state.controls.sort) {
		reorder(state, filters.sort);
	}
	// In the order they stand in once sorted, so pages follow the sort
	const matches = [...document.querySelectorAll(state.selector)]
		.filter((tile) => matchesFilters(state.read(tile), filters));
	state.matches = matches.length;
	if (state.controls.noMatch) {
		state.controls.noMatch.hidden = matches.length > 0;
	}

	const { pageSize } = state;
	state.pages = pageSize > 0 ? Math.max(1, Math.ceil(matches.length / pageSize)) : 1;
	state.page = Math.min(Math.max(shouldKeepPage ? state.page : 1, 1), state.pages);
	const shown = new Set(pageSize > 0 ? matches.slice((state.page - 1) * pageSize, state.page * pageSize) : matches);
	for (const tile of state.tiles) {
		tile.hidden = !shown.has(tile);
	}
	for (const pager of state.pagers) {
		drawPager(state, pager);
	}
	syncUrl(state, filters);
}

// Tell screen readers what the filters keep, the grid changing out of their
// sight: how many cards, or that none matches
function announce(state) {
	const { status, noMatch } = state.controls;
	if (!status) {
		return;
	}
	clearTimeout(state.announceTimer);
	state.announceTimer = setTimeout(() => {
		if (state.matches === 0) {
			status.textContent = noMatch?.textContent.trim() ?? '';
			return;
		}
		const count = state.numbers.format(state.matches);
		status.textContent = (status.dataset.labelResults ?? '').replaceAll('{count}', () => count);
	}, ANNOUNCE_DELAY);
}

// Back to the top of the grid when it scrolled out of view
function scrollToGrid(state) {
	const [anchor] = state.pagers;
	if (!anchor || anchor.getBoundingClientRect().top >= 0) {
		return;
	}
	const reduced = matchMedia('(prefers-reduced-motion: reduce)').matches;
	anchor.scrollIntoView({ block: 'start', behavior: reduced ? 'auto' : 'smooth' });
}

function onPagerClick(state, event) {
	const link = event.target instanceof Element ? event.target.closest('a[data-page]') : undefined;
	if (!link) {
		return;
	}
	event.preventDefault();
	const pager = link.closest('[data-page-size]');
	const arrow = link.getAttribute('aria-label');

	state.page = Number(link.dataset.page);
	apply(state, true);
	scrollToGrid(state);

	// The links were redrawn: keyboard users, whose clicks have no pointer
	// detail, stay on the arrow they used or land on the new current page
	if (!pager || event.detail !== 0) {
		return;
	}
	const next = (arrow && pager.querySelector(`a[aria-label="${CSS.escape(arrow)}"]`))
		|| pager.querySelector('.is-current');
	next?.focus();
}

// A select only takes values it offers
function canRestore(element, value) {
	const isOffered = !(element instanceof HTMLSelectElement)
		|| [...element.options].some((option) => option.value === value);
	return Boolean(element) && Boolean(value) && isOffered;
}

// A shared link can arrive already filtered
function restore(state) {
	const parameters = new URL(location.href).searchParams;
	const page = Math.trunc(Number(parameters.get('p') ?? '1'));
	state.page = Math.max(1, page || 1);

	const fields = [['q', 'search'], ['collection', 'collection'], ['rarity', 'rarity'], ['sort', 'sort']];
	for (const [name, control] of fields) {
		const element = state.controls[control];
		const value = parameters.get(name);
		if (canRestore(element, value)) {
			element.value = value;
		}
	}
}

function createState({ tiles, read, search, collection, rarity, sort, noMatch, status, pagers }) {
	const all = [...document.querySelectorAll(tiles)];
	const pagerElements = pagers ? [...document.querySelectorAll(pagers)] : [];
	const lang = document.documentElement.lang || undefined;
	return {
		selector: tiles,
		read,
		tiles: all,
		// The order the page came in, ties fall back to it
		original: new Map(all.map((tile, index) => [tile, index])),
		collator: new Intl.Collator(lang, { sensitivity: 'base' }),
		numbers: new Intl.NumberFormat(lang),
		controls: {
			search: find(search),
			collection: find(collection),
			rarity: find(rarity),
			sort: find(sort),
			noMatch: find(noMatch),
			status: find(status),
		},
		pagers: pagerElements,
		pageSize: Number(pagerElements[0]?.dataset.pageSize ?? 0),
		page: 1,
		pages: 1,
		// Tiles the filters keep, and the pending announcement of it
		matches: all.length,
		announceTimer: undefined,
	};
}

// Filtering happens as the text is typed, Enter would leave the page
function onSearchKeyDown(event) {
	if (event.key === 'Enter') {
		event.preventDefault();
	}
}

export function attachCardFilter(options) {
	const state = createState(options);
	const { search, collection, rarity, sort } = state.controls;
	const selects = [collection, rarity, sort].filter(Boolean);
	const onChange = () => {
		apply(state);
		announce(state);
	};
	const onClick = (event) => onPagerClick(state, event);

	search?.addEventListener('input', onChange);
	search?.addEventListener('keydown', onSearchKeyDown);
	for (const select of selects) {
		select.addEventListener('change', onChange);
	}
	for (const pager of state.pagers) {
		pager.addEventListener('click', onClick);
	}

	restore(state);
	apply(state, true);

	return () => {
		clearTimeout(state.announceTimer);
		search?.removeEventListener('input', onChange);
		search?.removeEventListener('keydown', onSearchKeyDown);
		for (const select of selects) {
			select.removeEventListener('change', onChange);
		}
		for (const pager of state.pagers) {
			pager.removeEventListener('click', onClick);
		}
	};
}
