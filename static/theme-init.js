// Theme of a first visit, picked before the page is painted: the server
// renders the theme of the `theme` cookie, set here from the system's.
// Loaded synchronously from <head>, as a file rather than an inline script
// so the Content-Security-Policy can forbid those (script-src 'self').
if (!document.cookie.includes('theme=')) {
	const theme = matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
	// Read by the server on the next requests, the Cookie Store API is async
	// eslint-disable-next-line unicorn/no-document-cookie
	document.cookie = `theme=${theme}; Path=/; Max-Age=31536000; SameSite=Lax`;
	document.documentElement.setAttribute('theme', theme);
	// The browser's bars, as the server rendered them for the light theme
	document.querySelector('meta[name="theme-color"]')?.setAttribute('content', theme === 'dark' ? '#171c1a' : '#fafcfb');
}
