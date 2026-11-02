// The pack page's switch for its endless animations (swirling background,
// floating booster, pulsing buttons and bars), remembered in the browser

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

// Returns the function that removes the listener
export function onStillChange(listener) {
	listeners.add(listener);
	return () => listeners.delete(listener);
}
