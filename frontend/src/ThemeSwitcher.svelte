<script>
	// Color of the browser's bars in each theme, the header's (--color-bg-2)
	const THEME_COLORS = { light: "#fafcfb", dark: "#171c1a" };

	let { labelLight = "", labelDark = "" } = $props();

	let theme = $state(document.documentElement.getAttribute("theme") ?? "light");
	// The button names what it does: switching to the other theme
	const label = $derived(theme === "dark" ? labelLight : labelDark);

	function toggleTheme() {
		theme = theme === "dark" ? "light" : "dark";

		document.documentElement.setAttribute("theme", theme);
		document.cookie = `theme=${theme}; Path=/; Max-Age=31536000; SameSite=Lax`;
		// The installed app's and the mobile browser's bars follow
		document.querySelector('meta[name="theme-color"]')?.setAttribute("content", THEME_COLORS[theme]);
	}
</script>

<button
	type="button"
	aria-label={label}
	title={label}
	onclick={toggleTheme}
>
	{#if theme === "dark"}
		<!-- Sun -->
		<svg viewBox="0 0 24 24" aria-hidden="true">
			<circle
				cx="12"
				cy="12"
				r="4"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
			/>
			<path
				d="M12 2v2M12 20v2M4.93 4.93l1.42 1.42M17.65 17.65l1.42 1.42
				   M2 12h2M20 12h2M4.93 19.07l1.42-1.42M17.65 6.35l1.42-1.42"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
			/>
		</svg>
	{:else}
		<!-- Moon -->
		<svg viewBox="0 0 24 24" aria-hidden="true">
			<path
				d="M21 12.79A9 9 0 1 1 11.21 3
				   7 7 0 0 0 21 12.79Z"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
			/>
		</svg>
	{/if}
</button>

<style>
	button {
		display: inline-flex;
		align-items: center;
		justify-content: center;

		line-height: 0;

		border: 0;
		background: none;
		color: inherit;
		cursor: pointer;
	}

	svg {
		width: 1.5rem;
		height: 1.5rem;
	}
</style>
