// Close codes after which reconnecting would fail the same way
const FINAL_CLOSE_CODES = new Map([
	// Unsupported data, policy violation, message too big: a client bug
	[1003, 'error'],
	[1008, 'error'],
	[1009, 'error'],
	[4401, 'unauthorized'],
	[4429, 'too-many'],
]);

const RETRY_MIN_MS = 1000;
const RETRY_MAX_MS = 30_000;

function shoutboxUrl() {
	return `${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}/api/shoutbox`;
}

// Exponential backoff with jitter, so clients don't all come back at once after a restart
function retryDelay(attempt) {
	const delay = Math.min(RETRY_MAX_MS, RETRY_MIN_MS * (2 ** attempt));
	const random = crypto.getRandomValues(new Uint32Array(1))[0] / (2 ** 32);
	return (delay / 2) + (random * delay / 2);
}

// Connection to the shoutbox and private messages that reconnects until closed
//
// `onEvent` receives every server event, `onStatus` the connection status: `connecting`, `open`, then
// `closed` with a reason (`unauthorized`, `too-many` or `error`) when it won't reconnect
export class ShoutboxConnection {
	#socket;
	#attempt = 0;
	#timer;
	#isStopped = false;
	#onEvent;
	#onStatus;

	constructor({ onEvent, onStatus }) {
		this.#onEvent = onEvent;
		this.#onStatus = onStatus;
		this.#open();
	}

	#open() {
		this.#onStatus('connecting');
		const socket = new WebSocket(shoutboxUrl());
		socket.addEventListener('open', () => {
			this.#attempt = 0;
			this.#onStatus('open');
		});
		socket.addEventListener('message', (event) => this.#handleMessage(event));
		socket.addEventListener('close', (event) => this.#handleClose(event));
		this.#socket = socket;
	}

	#handleMessage(event) {
		try {
			this.#onEvent(JSON.parse(event.data));
		} catch (error) {
			console.error('Invalid shoutbox event:', error);
		}
	}

	#handleClose(event) {
		this.#socket = undefined;
		if (this.#isStopped) {
			return;
		}
		const reason = FINAL_CLOSE_CODES.get(event.code);
		if (reason) {
			this.#isStopped = true;
			this.#onStatus('closed', reason);
			return;
		}
		this.#onStatus('connecting');
		this.#timer = setTimeout(() => this.#open(), retryDelay(this.#attempt));
		this.#attempt += 1;
	}

	// Returns whether the request could be sent
	#request(request) {
		if (this.#socket?.readyState !== WebSocket.OPEN) {
			return false;
		}
		this.#socket.send(JSON.stringify(request));
		return true;
	}

	// Post to the shoutbox, or privately to a user. `to` is left out of the request when undefined.
	send(body, to) {
		return this.#request({ type: 'send', body, to });
	}

	// Show the conversation with a user, or none with an empty username
	view(username) {
		return this.#request({ type: 'view', with: username });
	}

	// Ignore a user for an `hour`, a `day` or a `week`
	ignore(username, duration) {
		return this.#request({ type: 'ignore', username, duration });
	}

	unignore(username) {
		return this.#request({ type: 'unignore', username });
	}

	close() {
		this.#isStopped = true;
		clearTimeout(this.#timer);
		this.#socket?.close(1000);
	}
}
