// Coin top-up popup shared by the pages selling with coins (templates/partials/topup-popup.html)

// Called with the new balance once a pack is bought without leaving the page,
// for the popup opened last. Without one the form posts as usual.
let onCredited = null;

// Buy a pack in the background. A failure posts the form after all, the
// page it lands on tells what went wrong.
async function creditInPlace(event) {
	const callback = onCredited;
	if (!callback) {
		return;
	}
	event.preventDefault();

	const form = event.currentTarget;
	const dialog = form.closest("dialog");
	for (const button of dialog.querySelectorAll("button[type=submit]")) {
		button.disabled = true;
	}

	let result = null;
	try {
		const response = await fetch(form.action, {
			method: "POST",
			headers: { Accept: "application/json" },
			body: new URLSearchParams(new FormData(form)),
		});
		if (response.ok) {
			result = await response.json();
		}
	} catch {
		// Not JSON (logged out, the login page) or offline
	}

	if (typeof result?.coins !== "number") {
		form.submit();
		return;
	}
	dialog.close();
	callback(result);
}

let listening = false;

// Open the top-up popup. Packs crediting less than `missing` coins are grayed
// out, they wouldn't cover what the user is after. With credited, a bought
// pack is credited without reloading the page and the new balance handed to
// it. Returns false when the page has no popup.
export function openTopup(missing = 0, credited = null) {
	const dialog = document.querySelector("#shop-topup-popup");
	if (!dialog) {
		return false;
	}

	if (!listening) {
		listening = true;
		for (const form of dialog.querySelectorAll(".shop-topup-pack")) {
			form.addEventListener("submit", creditInPlace);
		}
	}
	onCredited = credited;

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
