// Requests to the app's own endpoints.
//
// Expected failures (a wrong password, a taken username, not enough coins)
// come back as 4xx, which browsers log in red in the console. Asked with
// `X-Soft-Errors: 1`, the server answers them with a 200 and the real status
// in `X-Status`: the response is rebuilt with that status, so callers keep
// checking `response.ok` and `response.status` as usual.

// Statuses the server may report through `X-Status`
const minSoftStatus = 400;
const maxSoftStatus = 599;

export async function apiFetch(url, options = {}) {
	const headers = new Headers(options.headers);
	headers.set('X-Soft-Errors', '1');

	const response = await fetch(url, {
		credentials: 'same-origin',
		...options,
		headers,
	});

	const status = Number(response.headers.get('X-Status'));
	if (!Number.isSafeInteger(status) || status < minSoftStatus || status > maxSoftStatus) {
		return response;
	}
	return new Response(response.body, {
		status,
		statusText: '',
		headers: response.headers,
	});
}
