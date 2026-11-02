// Selling inventory cards without reloading the page: the card popup
// (InventoryCardPopup.svelte) and the duplicates dialog of
// templates/pages/inventory.html both post their sale here, then apply what
// it changed to the page behind them.

import { updateNavBalance } from './topup.js';

// Post a sale form in the background: what the sale did if it went through,
// why it failed otherwise
export async function postSale(form) {
	try {
		const response = await fetch(form.action, {
			method: 'POST',
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

// Show a sale's outcome in the page that made it: the balance, the stacks it
// took copies from and what selling the duplicates left would pay
export function applySale(result) {
	updateNavBalance(result.navBalance);

	const wallet = document.querySelector('#shop-coins');
	if (wallet && result.coinsLabel) {
		wallet.textContent = result.coinsLabel;
	}

	const stacks = result.stacks ?? [];
	for (const stack of stacks) {
		updateStack(stack);
	}
	updateDuplicates(result.duplicates);

	const notice = document.querySelector('[data-inventory-notice]');
	if (!notice || !result.message) {
		return;
	}
	notice.className = 'shop-message shop-message-success';
	notice.textContent = result.message;
	notice.hidden = false;
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
// its tile and the copy badge, or the whole tile when its last copy went
function updateStack({ card, foil, copies, owned, acquiredLast }) {
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
		return;
	}
}

// What selling the duplicates pays follows every sale, said by the dialog it
// opens. With none left the dialog and its button go away.
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
	const text = dialog.querySelector('#inventory-sell-duplicates-text');
	if (text) {
		text.textContent = duplicates.text;
	}
	const confirm = dialog.querySelector('button[type=submit]');
	if (confirm) {
		confirm.textContent = duplicates.confirm;
	}
}

// Sell the duplicates of an inventory from the dialog of
// templates/pages/inventory.html, the page staying put. A sale takes every
// duplicate at once, so the dialog and its button go away with them.
export function enhanceSellDuplicates(form) {
	const dialog = form.closest('dialog');

	async function sellDuplicates(event) {
		event.preventDefault();
		const error = form.querySelector('[data-sell-error]');
		for (const button of dialog.querySelectorAll('button')) {
			button.disabled = true;
		}

		const result = await postSale(form);
		if (result.error) {
			error.textContent = result.error;
			error.hidden = false;
			for (const button of dialog.querySelectorAll('button')) {
				button.disabled = false;
			}
			return;
		}

		dialog.close();
		applySale(result);
		// The button that opened the dialog goes away with the last of the
		// duplicates, taking the focus the dialog gives back: the message
		// saying what the sale paid gets it instead
		focusNotice();
	}

	form.addEventListener('submit', sellDuplicates);
}
