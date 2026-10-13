<script>
	import { onMount } from "svelte";

	let languages = [
		{ code: "fr", label: "Français", flag: "🇫🇷" },
		{ code: "en", label: "English", flag: "🇬🇧" },
		{ code: "es", label: "Español", flag: "🇪🇸" },
	];

	let defaultLanguage = "en";

	let isOpen = false;
	let selectedCode = normalizeCode(defaultLanguage);
	let rootElement;
	let triggerElement;
	let label;

	function normalizeCode(code) {
		return String(code || "fr")
			.trim()
			.toLowerCase()
			.split(/[-_]/)[0];
	}

	function readLanguageCookie() {
		if (typeof document === "undefined") return null;

		const entry = document.cookie
			.split(";")
			.map((part) => part.trim())
			.find((part) => part.startsWith("lang="));

		if (!entry) return null;

		try {
			return decodeURIComponent(entry.slice("lang=".length));
		} catch {
			return entry.slice("lang=".length);
		}
	}

	function supportsLanguage(code) {
		const normalized = normalizeCode(code);

		return languages.some(
			(language) => normalizeCode(language.code) === normalized,
		);
	}

	$: currentLanguage = languages.find(
		(language) => normalizeCode(language.code) === selectedCode,
	) ??
		languages[0] ?? { code: "fr", label: "Français", flag: "🇫🇷" };

	function selectLanguage(code) {
		const languageCode = normalizeCode(code);

		if (!supportsLanguage(languageCode)) return;

		const secure = window.location.protocol === "https:" ? "Secure" : "";

		document.cookie =
			`lang=${encodeURIComponent(languageCode)}; ` +
			`Path=/; Max-Age=31536000; ${secure}`;

		// Reload the current URL after setting the cookie.
		window.location.reload();
	}

	onMount(() => {
		const target = document.getElementById("language-switcher");
		if (target) {
			label = target.dataset.label ?? "";
		}
		const cookieLanguage = readLanguageCookie();

		if (cookieLanguage && supportsLanguage(cookieLanguage)) {
			selectedCode = normalizeCode(cookieLanguage);
		} else if (supportsLanguage(defaultLanguage)) {
			selectedCode = normalizeCode(defaultLanguage);
		} else if (languages.length > 0) {
			selectedCode = normalizeCode(languages[0].code);
		}

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
		aria-label={`Language: ${currentLanguage.label}. ${label}`}
		aria-haspopup="true"
		aria-expanded={isOpen}
		aria-controls="language-switcher-menu"
		on:click={() => (isOpen = !isOpen)}
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

	{#if isOpen}
		<div
			id="language-switcher-menu"
			class="language-switcher-menu"
			aria-label={label}
		>
			{#each languages as language (language.code)}
				<button
					type="button"
					class="language-option"
					aria-pressed={normalizeCode(language.code) === selectedCode}
					on:click={() => selectLanguage(language.code)}
				>
					<span class="language-option-name">
						{#if language.flag}
							<span aria-hidden="true">
								{language.flag}
							</span>
						{/if}

						<span>{language.label}</span>
					</span>

					<span class="language-option-code">
						{normalizeCode(language.code).toUpperCase()}
					</span>
				</button>
			{/each}
		</div>
	{/if}
</div>

<style>
	:global(:root[theme="light"]) {
		--color-language-switcher: #e7eeeb;
	}

	:global(:root[theme="dark"]) {
		--color-language-switcher: #2f2f2f;
	}

	.language-switcher-wrap {
		position: relative;
		display: inline-block;
		font-family: inherit;
	}

	.language-switcher,
	.language-option {
		font: inherit;
		color: var(--color-text, #202522);
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
		background: var(--color-language-switcher, #e7eeeb);
		transition: background-color 140ms ease;
	}

	.language-switcher:hover {
		background: color-mix(
			in srgb,
			var(--color-language-switcher, #e7eeeb),
			black 10%
		);
	}

	.language-switcher:active {
		background: color-mix(
			in srgb,
			var(--color-language-switcher, #e7eeeb),
			black 20%
		);
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
		border: 1px solid
			color-mix(
				in srgb,
				var(--color-language-switcher, #e7eeeb),
				var(--color-text, #202522) 18%
			);
		border-radius: 0.65rem;
		background: var(--color-language-switcher, #e7eeeb);
		color: var(--color-text, #202522);
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
		background: color-mix(
			in srgb,
			var(--color-language-switcher, #e7eeeb),
			black 8%
		);
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
		opacity: 0.7;
	}

	@media (prefers-reduced-motion: reduce) {
		.language-switcher,
		.lsw-caret {
			transition: none;
		}
	}
</style>
