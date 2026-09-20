<script>
	import { onMount } from "svelte";

	let theme = "light";
	let labelLight = "";
	let labelDark = "";

	function toggleTheme() {
		theme = theme === "dark" ? "light" : "dark";

		document.documentElement.setAttribute("theme", theme);
		document.cookie = `theme=${theme}; Path=/; Max-Age=31536000; SameSite=Lax`;
	}

	onMount(() => {
		const target = document.querySelector("#theme-switcher");

		if (!target) {
			return;
		}

		theme = document.documentElement.getAttribute("theme") ?? "light";

		labelLight = target.dataset.themeLabelLight ?? "";
		labelDark = target.dataset.themeLabelDark ?? "";
	});
</script>

<button
	type="button"
	aria-label={theme === "dark" ? labelLight : labelDark}
	title={theme === "dark" ? labelLight : labelDark}
	on:click={toggleTheme}
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

		vertical-align: -0.25em;
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
