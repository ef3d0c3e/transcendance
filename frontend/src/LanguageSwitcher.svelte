<script>
	import { onMount } from "svelte";

	// `label` names the menu, `labelCurrent` the language of the page
	let { label = "", labelCurrent = "" } = $props();

	const languages = [
		{ code: "fr", label: "Français", flag: "🇫🇷" },
		{ code: "en", label: "English", flag: "🇬🇧" },
	];

	const defaultLanguage = "en";

	let isOpen = $state(false);
	let rootElement = $state();
	let triggerElement = $state();

	function normalizeCode(code) {
		return String(code || defaultLanguage)
			.trim()
			.toLowerCase()
			.split(/[-_]/)[0];
	}

	function supportsLanguage(code) {
		const normalized = normalizeCode(code);

		return languages.some(
			(language) => normalizeCode(language.code) === normalized,
		);
	}

	// The language the server rendered the page in
	const pageCode = normalizeCode(document.documentElement.lang);
	const selectedCode = supportsLanguage(pageCode) ? pageCode : defaultLanguage;
	const currentLanguage =
		languages.find((language) => normalizeCode(language.code) === selectedCode) ?? languages[0];

	function selectLanguage(code) {
		const languageCode = normalizeCode(code);

		if (!supportsLanguage(languageCode)) return;

		const secure = window.location.protocol === "https:" ? "; Secure" : "";

		document.cookie =
			`lang=${encodeURIComponent(languageCode)}; ` +
			`Path=/; Max-Age=31536000; SameSite=Lax${secure}`;

		// Reload the current URL after setting the cookie.
		window.location.reload();
	}

	onMount(() => {
		function handleOutsidePointer(event) {
			if (rootElement && !rootElement.contains(event.target)) {
				isOpen = false;
			}
		}

		function handleKeydown(event) {
			if (event.key === "Escape" && isOpen) {
				isOpen = false;
				triggerElement?.focus();
			}
		}

		document.addEventListener("pointerdown", handleOutsidePointer);
		document.addEventListener("keydown", handleKeydown);

		return () => {
			document.removeEventListener("pointerdown", handleOutsidePointer);
			document.removeEventListener("keydown", handleKeydown);
		};
	});
</script>

<div class="language-switcher-wrap" bind:this={rootElement}>
	<button
		bind:this={triggerElement}
		type="button"
		class="language-switcher"
		aria-label={labelCurrent ? `${labelCurrent}. ${label}` : label}
		aria-expanded={isOpen}
		aria-controls="language-switcher-menu"
		onclick={() => (isOpen = !isOpen)}
	>
		{#if currentLanguage.flag}
			<span aria-hidden="true">
				{currentLanguage.flag}
			</span>
		{/if}

		<span>
			{normalizeCode(currentLanguage.code).toUpperCase()}
		</span>

		<svg
			class="lsw-caret"
			class:open={isOpen}
			width="14"
			height="14"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2.4"
			stroke-linecap="round"
			stroke-linejoin="round"
			aria-hidden="true"
		>
			<polyline points="6 11 12 17 18 11" />
		</svg>
	</button>

	<div
		id="language-switcher-menu"
		class="language-switcher-menu"
		role="group"
		aria-label={label}
		hidden={!isOpen}
	>
		{#each languages as language (language.code)}
			<button
				type="button"
				class="language-option"
				aria-pressed={normalizeCode(language.code) === selectedCode}
				onclick={() => selectLanguage(language.code)}
			>
				<span class="language-option-name">
					{#if language.flag}
						<span aria-hidden="true">
							{language.flag}
						</span>
					{/if}

					<!-- Each language's own name, read in that language -->
					<span lang={language.code}>{language.label}</span>
				</span>

				<span class="language-option-code" aria-hidden="true">
					{normalizeCode(language.code).toUpperCase()}
				</span>
			</button>
		{/each}
	</div>
</div>

<style>
	.language-switcher-wrap {
		position: relative;
		display: inline-block;
		font-family: inherit;
	}

	.language-switcher,
	.language-option {
		font: inherit;
		color: var(--color-text);
		cursor: pointer;
	}

	.language-switcher {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.45rem;
		padding: 0.5rem 0.65rem;
		border: 1px solid transparent;
		border-radius: 0.5rem;
		background: var(--color-bg-4);
		transition: background-color 140ms ease;
	}

	.language-switcher:hover {
		background: color-mix(in srgb, var(--color-bg-4), black 10%);
	}

	.language-switcher:active {
		background: color-mix(in srgb, var(--color-bg-4), black 20%);
	}

	.language-switcher:focus-visible,
	.language-option:focus-visible {
		outline: 2px solid currentColor;
		outline-offset: 2px;
	}

	.lsw-caret {
		flex: none;
		transition: transform 140ms ease;
	}

	.lsw-caret.open {
		transform: rotate(180deg);
	}

	.language-switcher-menu {
		position: fixed;
		z-index: 1000;
		top: auto;
		right: auto;
		min-width: 11rem;
		padding: 0.3rem;
		border: 1px solid var(--color-border-2);
		border-radius: 0.65rem;
		background: var(--color-bg-4);
		color: var(--color-text);
		box-shadow: 0 8px 24px rgb(0 0 0 / 16%);
	}

	.language-option {
		display: flex;
		width: 100%;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		padding: 0.55rem 0.6rem;
		border: 0;
		border-radius: 0.4rem;
		background: transparent;
		text-align: left;
	}

	.language-option:hover,
	.language-option[aria-pressed="true"] {
		background: color-mix(in srgb, var(--color-bg-4), black 8%);
	}

	.language-option[aria-pressed="true"] {
		font-weight: 650;
	}

	.language-option-name {
		display: inline-flex;
		align-items: center;
		gap: 0.55rem;
	}

	.language-option-code {
		font-size: 0.8em;
		color: var(--color-text-muted);
	}

	@media (prefers-reduced-motion: reduce) {
		.language-switcher,
		.lsw-caret {
			transition: none;
		}
	}
</style>
