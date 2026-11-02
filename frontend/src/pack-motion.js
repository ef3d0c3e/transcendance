// The pack page's switch for its endless animations: the swirling
// background, the floating booster, the pulsing buttons and bars. Remembered
// in the browser; the background and the page follow it as it changes.

const storageKey = 'pack-motion';
const listeners = new Set();

export function loadStill() {
	try {
		return localStorage.getItem(storageKey) === 'still';
	} catch {
		// Storage blocked: animated
		return false;
	}
}

export function saveStill(isStill) {
	try {
		localStorage.setItem(storageKey, isStill ? 'still' : 'animated');
	} catch {
		// Not remembered, still used for this visit
	}
	for (const listener of listeners) {
		listener(isStill);
	}
}

// Calls listener with the new setting on every change, until the returned
// function is called
export function onStillChange(listener) {
	listeners.add(listener);
	return () => listeners.delete(listener);
}
