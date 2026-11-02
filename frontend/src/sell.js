// Selling inventory cards without reloading the page: the card popup
// (InventoryCardPopup.svelte) and the duplicates dialog of
// templates/pages/inventory.html both post their sale here, then apply what
// it changed to the page behind them.

import { apiFetch } from './api.js';
import { refreshCardName } from './card-name.js';
import { updateNavBalance } from './topup.js';

// Post a sale form in the background: what the sale did if it went through,
// why it failed otherwise. A sale that can't be made (the copies went in
// another tab) is an expected failure, kept out of the console.
export async function postSale(form) {
	try {
		const response = await apiFetch(form.action, {
			method: 'POST',
			// Exactly this value: the server answers JSON to it, a redirect otherwise
			headers: { Accept: 'application/json' },
			body: new URLSearchParams(new FormData(form)),
		});
		const result = await response.json();
		return response.ok
			? result
			: { error: result.error ?? form.dataset.labelError ?? '' };
	} catch {
		// Not JSON (logged out, the login page) or offline
		return { error: form.dataset.labelError ?? '' };
	}
}

// Translations for what a sale changes in the page, on the card popup's
// mount point. The count next to the title comes in the page's two plural
// forms, rendered for 1 and 2 copies, and only where it counts copies.
function pageLabels() {
	const dataset = document.querySelector('#inventory-card-popup')?.dataset ?? {};
	return {
		foil: dataset.labelFoil ?? '',
		copies: dataset.labelCopies ?? '',
		countOne: dataset.labelCountOne,
		countOther: dataset.labelCountOther,
		isFiltered: dataset.filtered !== undefined,
		stale: dataset.labelStale ?? '',
		pageSold: dataset.labelPageSold ?? '',
		reload: dataset.labelReload ?? '',
	};
}

// "12 cards" for `count`, in the plural form the language takes for it
function countText(count, labels) {
	const lang = document.documentElement.lang || undefined;
	const pattern = new Intl.PluralRules(lang).select(count) === 'one' ? labels.countOne : labels.countOther;
	const number = new Intl.NumberFormat(lang).format(count);
	return pattern.replace(/\d+/, () => number);
}

// The counts of the page lose the copies sold: the cards tab's, and the one
// next to the title. That one only counts the cards the filters keep: copies
// sold from the whole inventory can't be told apart there. Returns whether
// it went stale for that reason.
function updateCounts({ sold, inView }, labels) {
	const tab = document.querySelector('.inventory-tab-cards .inventory-tab-count');
	const total = Number(tab?.textContent);
	if (tab && Number.isSafeInteger(total)) {
		tab.textContent = String(Math.max(0, total - sold));
	}

	const heading = document.querySelector('.inventory-header .inventory-count');
	if (!heading || !labels.countOne || !labels.countOther) {
		return false;
	}
	if (!inView && labels.isFiltered) {
		return true;
	}
	const left = Number(heading.textContent.replaceAll(/\D/g, '')) - sold;
	if (left > 0) {
		heading.textContent = countText(left, labels);
	} else {
		heading.hidden = true;
	}
	return false;
}

// Say what the sale did in the page's notice, in a live region that was
// there from the start, so screen readers announce it. `hint` follows on a
// line of its own.
function showNotice(message, hint) {
	const notice = document.querySelector('[data-inventory-notice]');
	if (!notice || !message) {
		return;
	}
	notice.className = 'shop-message shop-message-success';
	notice.replaceChildren(message);
	if (hint) {
		const line = document.createElement('span');
		line.className = 'shop-message-hint';
		line.textContent = hint;
		notice.append(' ', line);
	}
	notice.hidden = false;
}

// Selling the last cards of the page leaves it empty between its page
// links: it says so, with a way to load the cards left
function showEmptyPage(labels) {
	const grid = document.querySelector('.inventory-cards');
	if (!grid || grid.querySelector('.card-container') || !labels.pageSold) {
		return;
	}
	const text = document.createElement('p');
	text.textContent = labels.pageSold;
	const link = document.createElement('a');
	link.href = location.href;
	link.textContent = labels.reload;
	const message = document.createElement('div');
	message.className = 'inventory-message';
	message.append(text, link);
	grid.replaceWith(message);
}

// Show a sale's outcome in the page that made it: the balance, the stacks it
// took copies from, the counts and what selling the duplicates left would
// pay. `sale` says how many copies were `sold`, and whether they were all
// `inView`, among the cards the page lists.
export function applySale(result, sale) {
	updateNavBalance(result.navBalance);

	const wallet = document.querySelector('#shop-coins');
	if (wallet && result.coinsLabel) {
		wallet.textContent = result.coinsLabel;
	}

	const labels = pageLabels();
	const stacks = result.stacks ?? [];
	for (const stack of stacks) {
		updateStack(stack, labels);
	}
	updateDuplicates(result.duplicates);
	const isStale = updateCounts(sale, labels);
	showNotice(result.message, isStale ? labels.stale : '');
	showEmptyPage(labels);
}

// Where the focus goes when the popup or button that had it is gone with the
// sale: the message saying what the sale did
export function focusNotice() {
	const notice = document.querySelector('[data-inventory-notice]');
	if (notice) {
		notice.focus();
	}
}

// Update one stack the sale took copies from: what the card popup reads off
// its tile, the copy badge and the name telling them, or the whole tile when
// its last copy went
function updateStack({ card, foil, copies, owned, acquiredLast }, labels) {
	const selector = `.inventory-cards .card-container[data-sell-id="${CSS.escape(String(card))}"]`;
	for (const tile of document.querySelectorAll(selector)) {
		if ((tile.dataset.sellFoil ?? '') !== foil) {
			continue;
		}
		if (copies === 0) {
			tile.remove();
			return;
		}
		tile.dataset.copies = String(copies);
		tile.dataset.sellOwned = String(owned);
		tile.dataset.acquiredLast = acquiredLast;
		const badge = tile.querySelector('.card-stack');
		if (copies > 1) {
			if (badge) {
				badge.textContent = `×${copies}`;
			}
		} else {
			badge?.remove();
		}
		if (tile.hasAttribute('aria-label')) {
			refreshCardName(tile, labels);
		}
		return;
	}
}

// What selling the duplicates pays follows every sale, said by the dialog it
// opens, which keeps how many there are. With none left the dialog and its
// button go away.
function updateDuplicates(duplicates) {
	if (!duplicates) {
		return;
	}
	const button = document.querySelector('.inventory-sell-duplicates');
	const dialog = document.querySelector('#inventory-sell-duplicates-popup');
	if (duplicates.count === 0) {
		if (dialog?.open) {
			dialog.close();
		}
		button?.remove();
		dialog?.remove();
		return;
	}
	if (!dialog) {
		return;
	}
	dialog.dataset.count = String(duplicates.count);
	const text = dialog.querySelector('#inventory-sell-duplicates-text');
	if (text) {
		text.textContent = duplicates.text;
	}
	const confirm = dialog.querySelector('button[type=submit]');
	if (confirm) {
		confirm.textContent = duplicates.confirm;
	}
}

// While the sale is made the dialog is busy and its buttons do nothing.
// Marked, not disabled: the focused button keeps the focus.
function setBusy(form, isBusy) {
	form.setAttribute('aria-busy', String(isBusy));
	for (const button of form.closest('dialog').querySelectorAll('button')) {
		button.setAttribute('aria-disabled', String(isBusy));
	}
}

// Sell the duplicates of an inventory from the dialog of
// templates/pages/inventory.html, the page staying put. A sale takes every
// duplicate at once, so the dialog and its button go away with them.
export function enhanceSellDuplicates(form) {
	const dialog = form.closest('dialog');

	async function sellDuplicates(event) {
		event.preventDefault();
		if (form.getAttribute('aria-busy') === 'true') {
			return;
		}
		const error = form.querySelector('[data-sell-error]');
		// The copies the dialog says it sells, from anywhere in the inventory
		const sold = Number(dialog.dataset.count) || 0;
		setBusy(form, true);

		const result = await postSale(form);
		setBusy(form, false);
		if (result.error) {
			error.textContent = result.error;
			error.hidden = false;
			return;
		}

		dialog.close();
		applySale(result, { sold, inView: false });
		// The button that opened the dialog goes away with the last of the
		// duplicates, taking the focus the dialog gives back: the message
		// saying what the sale paid gets it instead
		focusNotice();
	}

	form.addEventListener('submit', sellDuplicates);
}
