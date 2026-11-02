// Server-rendered cards (templates/partials/card.html) made into buttons.
//
// The content of a button is presentational, read through its name only:
// the name says what the card shows, its title, rarity and foil, and what
// the pills on it tell, how many copies and how many are up in trades.

// Text of an element of the card, empty when it has none
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

// Accessible name of a card. `labels` are the page's translations: `foil`,
// and `copies` for a stack of more than one card (`count` copies when the
// card doesn't tell).
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

// Makes a card a button: reachable, named, and with its collection link out
// of the tab order, as a link inside a button can't be reached reliably
export function makeCardButton(container, labels = {}) {
	container.tabIndex = 0;
	container.setAttribute('role', 'button');
	container.setAttribute('aria-label', cardName(container, labels));
	for (const link of container.querySelectorAll('a')) {
		link.tabIndex = -1;
	}
}
