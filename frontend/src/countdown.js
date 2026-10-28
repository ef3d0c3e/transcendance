// Format a number of seconds as M:SS, e.g "9:05"
function formatCountdown(seconds) {
	return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`;
}

// Tick one countdown pill down to zero, then reload the page so the server
// can hand out what the pill counts down to
function startCountdown(element) {
	const deadline = Date.now() + (Number(element.dataset.countdown) * 1000);

	const tick = () => {
		const remaining = Math.ceil((deadline - Date.now()) / 1000);
		if (remaining <= 0) {
			location.reload();
			return;
		}
		element.textContent = formatCountdown(remaining);
		setTimeout(tick, 1000);
	};
	tick();
}

// Count down every [data-countdown] pill of the page
export function enhanceCountdowns() {
	for (const element of document.querySelectorAll('[data-countdown]')) {
		startCountdown(element);
	}
}
