// Coin top-up popup shared by the pages selling with coins (templates/partials/topup-popup.html)

// Open the top-up popup. Packs crediting less than `missing` coins are grayed
// out, they wouldn't cover what the user is after. Returns false when the page
// has no popup.
export function openTopup(missing = 0) {
	const dialog = document.querySelector("#shop-topup-popup");
	if (!dialog) {
		return false;
	}

	for (const pack of dialog.querySelectorAll(".shop-topup-pack")) {
		const short = Number(pack.dataset.amount) < missing;
		pack.classList.toggle("is-short", short);
		const button = pack.querySelector("button[type=submit]");
		if (button) {
			button.disabled = short;
		}
	}

	dialog.showModal();
	return true;
}
