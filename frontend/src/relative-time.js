// Largest unit first, with the number of seconds in it
const units = [
	['year', 60 * 60 * 24 * 365],
	['month', 60 * 60 * 24 * 30],
	['week', 60 * 60 * 24 * 7],
	['day', 60 * 60 * 24],
	['hour', 60 * 60],
	['minute', 60],
	['second', 1],
];

function startOfDay(date) {
	return new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime();
}

// Calendar days between a date and today, e.g -1 for yesterday
function daysFromToday(date) {
	return Math.round((startOfDay(date) - startOfDay(new Date())) / (1000 * 60 * 60 * 24));
}

// Format a date relative to now, e.g "5 minutes ago", in the given locale
export function relativeTime(date, locale) {
	const format = new Intl.RelativeTimeFormat(locale || undefined, { numeric: 'auto' });
	const elapsed = (new Date(date).getTime() - Date.now()) / 1000;
	for (const [unit, seconds] of units) {
		if (Math.abs(elapsed) >= seconds) {
			// Count days by calendar so they match the day groups
			const count = unit === 'day' ? daysFromToday(new Date(date)) : Math.trunc(elapsed / seconds);
			return format.format(count, unit);
		}
	}
	return format.format(0, 'second');
}

// Full date and time, e.g for tooltips
export function fullDate(date, locale) {
	return new Intl.DateTimeFormat(locale || undefined, { dateStyle: 'full', timeStyle: 'short' }).format(new Date(date));
}

// Day a date belongs to: "Today", "Yesterday", then the date itself
export function dayLabel(date, locale) {
	const day = new Date(date);
	const now = new Date();
	const days = daysFromToday(day);
	if (days >= -1) {
		const label = new Intl.RelativeTimeFormat(locale || undefined, { numeric: 'auto' }).format(days, 'day');
		return label.charAt(0).toLocaleUpperCase(locale || undefined) + label.slice(1);
	}
	const options = { weekday: 'long', day: 'numeric', month: 'long' };
	if (day.getFullYear() !== now.getFullYear()) {
		options.year = 'numeric';
	}
	const label = new Intl.DateTimeFormat(locale || undefined, options).format(day);
	return label.charAt(0).toLocaleUpperCase(locale || undefined) + label.slice(1);
}

// Format the times of a server rendered notification list and group its items by day
export function enhanceNotificationList(list) {
	const { locale } = list.dataset;
	let previous = '';
	for (const item of list.querySelectorAll('.notification-item')) {
		const time = item.querySelector('time');
		if (time) {
			time.textContent = relativeTime(time.dateTime, locale);
			time.title = fullDate(time.dateTime, locale);
		}

		const label = dayLabel(item.dataset.created, locale);
		if (label === previous) {
			continue;
		}
		const heading = document.createElement('li');
		heading.className = 'notification-day';
		heading.setAttribute('role', 'presentation');
		heading.textContent = label;
		item.before(heading);
		previous = label;
	}
}
