// Server-rendered cards (templates/partials/card.html) made into buttons. A
// button's content is presentational, so its name tells all the card shows:
// title, rarity, foil, copies and how many are up in trades.

function textOf(container, selector) {
	return container.querySelector(selector)?.textContent.trim() ?? '';
}

// Rarity badge, lowercase so screen readers don't spell it out
function rarityOf(container) {
	return textOf(container, '.card-badge').toLocaleLowerCase(document.documentElement.lang || undefined);
}

// "Copies: 3" for a stack of more than one card
function copiesOf(container, labels) {
	const copies = Number(container.dataset.copies ?? labels.count ?? 0);
	return copies > 1 && labels.copies ? `${labels.copies}: ${copies}` : '';
}

// `labels`: the page's translations of `foil` and `copies`, and `count`, the
// copies when the card doesn't give them
export function cardName(container, labels = {}) {
	const isFoil = container.querySelector('.card-foil') !== null;
	return [
		textOf(container, '.card-title'),
		rarityOf(container),
		isFoil ? labels.foil : '',
		copiesOf(container, labels),
		// "1 in trades"
		textOf(container, '.card-locked'),
	].filter(Boolean).join(', ');
}

// Once what the card shows changed, e.g its copies after a sale
export function refreshCardName(container, labels = {}) {
	container.setAttribute('aria-label', cardName(container, labels));
}

// The collection link inside leaves the tab order: a link inside a button
// can't be reached reliably
export function makeCardButton(container, labels = {}) {
	container.tabIndex = 0;
	container.setAttribute('role', 'button');
	refreshCardName(container, labels);
	for (const link of container.querySelectorAll('a')) {
		link.tabIndex = -1;
	}
}
