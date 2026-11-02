// Countdown pills (next booster, next daily bonus). At zero the page is
// updated in place, never reloaded: that would lose what the user is doing
// (a trade-up contract, a chat message being typed).

// The server rounds the time left down: under a second, it says 0 while its
// deadline hasn't passed yet. Every countdown runs for a second at least.
const minimumSeconds = 1;

// M:SS, or H:MM:SS past an hour
function formatCountdown(seconds) {
	const hours = Math.floor(seconds / 3600);
	const minutes = Math.floor((seconds % 3600) / 60);
	const rest = String(seconds % 60).padStart(2, '0');
	return hours > 0
		? `${hours}:${String(minutes).padStart(2, '0')}:${rest}`
		: `${minutes}:${rest}`;
}

// What the countdown waited for is there. Labels come from <body>. In a
// button (the daily bonus's claim) it is enabled and says so; in a link (a
// booster to open) the pill says it's ready; elsewhere the pill becomes a
// link reloading the page when the user chooses to.
function finish(element) {
	const labels = document.body.dataset;
	const button = element.closest('button');
	if (button) {
		button.disabled = false;
		button.textContent = labels.labelCountdownClaim ?? '';
		return;
	}
	if (element.closest('a, [data-booster-open]')) {
		element.textContent = labels.labelCountdownDone ?? '';
		return;
	}
	const link = document.createElement('a');
	link.href = location.href;
	link.className = 'countdown-refresh';
	link.textContent = labels.labelCountdownRefresh ?? '';
	element.replaceChildren(link);
}

function startCountdown(element) {
	const seconds = Math.max(minimumSeconds, Number(element.dataset.countdown) || 0);
	const deadline = Date.now() + (seconds * 1000);

	const tick = () => {
		const remaining = Math.ceil((deadline - Date.now()) / 1000);
		if (remaining <= 0) {
			finish(element);
			return;
		}
		element.textContent = formatCountdown(remaining);
		setTimeout(tick, 1000);
	};
	tick();
}

export function enhanceCountdowns() {
	for (const element of document.querySelectorAll('[data-countdown]')) {
		startCountdown(element);
	}
}
