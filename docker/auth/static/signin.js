const root = document.querySelector('#signin');
const waiting = document.querySelector('#waiting');
const done = document.querySelector('#done');
const command = document.querySelector('#command');
const expiry = document.querySelector('#expiry');
const message = document.querySelector('#message');

const POLL_DELAY = 2000;

// Increased by every new code, so an older polling loop stops
const attempt = { current: 0 };

// Only paths of this site, so the link cannot send the admin elsewhere
function target() {
	// The gateway appends the original request as is, it may contain its own query
	const raw = location.search.startsWith('?rd=') ? location.search.slice(4) : '';
	const local = raw.startsWith('/') && !raw.startsWith('//') && !raw.startsWith('/\\');
	return local ? raw : '/auth/whoami';
}

function finish(text) {
	attempt.current += 1;
	waiting.hidden = true;
	done.hidden = false;
	message.textContent = text;
}

function pause() {
	return new Promise((resolve) => {
		setTimeout(resolve, POLL_DELAY);
	});
}

// Wait for the admin's answer, true when this attempt is over
async function check() {
	const response = await fetch('/auth/status');
	const { status } = await response.json();
	switch (status) {
		case 'approved': {
			location.assign(target());
			return true;
		}
		case 'denied': {
			finish('The request was denied.');
			return true;
		}
		case 'expired': {
			finish('The code expired.');
			return true;
		}
		default: {
			return false;
		}
	}
}

async function poll(id) {
	await pause();
	if (id === attempt.current && !(await check())) {
		await poll(id);
	}
}

async function start() {
	attempt.current += 1;
	const id = attempt.current;
	done.hidden = true;
	const response = await fetch('/auth/challenge', { method: 'POST' });
	const body = await response.json();
	if (!response.ok) {
		finish(body.error);
		return;
	}
	// ssh takes IPv6 addresses without brackets
	const host = location.hostname.replace(/^\[(.*)\]$/, '$1');
	command.textContent = `ssh -p ${root.dataset.port} ${body.code}@${host}`;
	const time = new Date(body.expires * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	expiry.textContent = `The code is valid until ${time}.`;
	waiting.hidden = false;
	await poll(id);
}

async function run() {
	try {
		await start();
	} catch {
		finish('Lost contact with the server.');
	}
}

document.querySelector('#copy').addEventListener('click', () => {
	navigator.clipboard?.writeText(command.textContent);
});
document.querySelector('#retry').addEventListener('click', run);
await run();
