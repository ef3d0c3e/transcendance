// Coin top-up popup shared by the pages selling with coins (templates/partials/topup-popup.html)

// Called with the new balance once a pack is bought without leaving the page,
// for the popup opened last. Without one the form posts as usual.
const popup = {
	onCredited: undefined,
	isListening: false,
};

// Post a top-up form in the background: the new balance if it went through,
// an empty result otherwise
async function postTopup(form) {
	try {
		const response = await fetch(form.action, {
			method: 'POST',
			headers: { Accept: 'application/json' },
			body: new URLSearchParams(new FormData(form)),
		});
		return response.ok ? await response.json() : {};
	} catch {
		// Not JSON (logged out, the login page) or offline
		return {};
	}
}

// Buy a pack in the background
async function creditInPlace(event) {
	const callback = popup.onCredited;
	if (!callback) {
		return;
	}
	event.preventDefault();

	const form = event.currentTarget;
	const dialog = form.closest('dialog');
	for (const button of dialog.querySelectorAll('button[type=submit]')) {
		button.disabled = true;
	}

	const result = await postTopup(form);
	if (typeof result?.coins !== 'number') {
		dialog.querySelector('[data-shop-topup-error]').hidden = false;
		return;
	}
	dialog.close();
	callback(result);
}

// Open the top-up popup. Packs crediting less than `missing` coins are grayed
// out, they wouldn't cover what the user is after. With credited, a bought
// pack is credited without reloading the page and the new balance handed to
// it. Returns false when the page has no popup.
export function openTopup(missing, credited) {
	const dialog = document.querySelector('#shop-topup-popup');
	if (!dialog) {
		return false;
	}

	if (!popup.isListening) {
		popup.isListening = true;
		for (const form of dialog.querySelectorAll('.shop-topup-pack')) {
			form.addEventListener('submit', creditInPlace);
		}
	}
	popup.onCredited = credited;
	dialog.querySelector('[data-shop-topup-error]').hidden = true;

	for (const pack of dialog.querySelectorAll('.shop-topup-pack')) {
		const isShort = Number(pack.dataset.amount) < (missing ?? 0);
		pack.classList.toggle('is-short', isShort);
		const button = pack.querySelector('button[type=submit]');
		if (button) {
			button.disabled = isShort;
		}
	}

	dialog.showModal();
	return true;
}
