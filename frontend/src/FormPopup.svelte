<script>
	import { onMount, tick, mount } from "svelte";
	import FormEnhancement from "./FormEnhancement.svelte";

	let { url, target, formSelector, enhancementSelector } = $props();

	let label = $state("");
	let open = $state(false);
	let loading = $state(false);
	let loaded = $state(false);

	let content;
	let popup;

	let closeTimer;

	onMount(() => {
		if (target) {
			label = target.dataset.label ?? "";
		}
	});

	async function openpopup() {
		clearTimeout(closeTimer);
		open = true;

		if (loaded || loading) {
			await tick();
			popup?.focus();
			return;
		}

		loading = true;

		try {
			const response = await fetch(url + "?fragment=1");

			if (!response.ok) {
				throw new Error(`HTTP ${response.status}`);
			}

			content.innerHTML = await response.text();

			const enhancementTarget =
				content.querySelector(enhancementSelector);

			if (enhancementTarget) {
				mount(FormEnhancement, {
					target: enhancementTarget,
					props: {
						formSelector,
						onSuccess: handleFormSuccess,
					},
				});
			}

			loaded = true;

			await tick();
			popup?.focus();
		} catch (error) {
			console.error(error);
			content.textContent = "Unable to load form.";
		} finally {
			loading = false;
		}
	}

	function closepopup() {
		clearTimeout(closeTimer);
		open = false;
	}

	function handleFormSuccess() {
		clearTimeout(closeTimer);

		closeTimer = setTimeout(() => {
			closepopup();
		}, 800);
	}

	function handleBackdropClick(event) {
		if (event.target === event.currentTarget) {
			closepopup();
		}
	}

	function handleKeydown(event) {
		if (event.key === "Escape" && open) {
			event.preventDefault();
			closepopup();
		}
	}
</script>

<button
	type="button"
	aria-haspopup="dialog"
	aria-expanded={open}
	onclick={open ? closepopup : openpopup}
>
	{label}
</button>

<div
	class="backdrop"
	class:open
	role="presentation"
	aria-hidden={!open}
	onclick={handleBackdropClick}
>
	<div
		class="popup"
		role="dialog"
		aria-modal="true"
		aria-label={label}
		tabindex="-1"
		bind:this={popup}
	>
		{#if loading}
			<p>Loading...</p>
		{/if}

		<div bind:this={content}></div>
	</div>
</div>

<svelte:window onkeydown={handleKeydown} />

<style>
	.backdrop {
		position: fixed;
		inset: 0;
		z-index: 1000;

		display: flex;
		align-items: center;
		justify-content: center;

		/* Respect notches */
		padding: max(1rem, env(safe-area-inset-top))
			max(1rem, env(safe-area-inset-right))
			max(1rem, env(safe-area-inset-bottom))
			max(1rem, env(safe-area-inset-left));

		background: rgb(0 0 0 / 55%);
		backdrop-filter: blur(3px);

		visibility: hidden;
		opacity: 0;
		pointer-events: none;

		transition:
			opacity 150ms ease,
			visibility 150ms ease;
	}

	.backdrop.open {
		visibility: visible;
		opacity: 1;
		pointer-events: auto;
	}

	.popup {
		position: relative;

		width: min(100%, 20rem);

		max-height: calc(
			100dvh - max(1rem, env(safe-area-inset-top)) -
				max(1rem, env(safe-area-inset-bottom)) - 2rem
		);

		overflow: auto;

		overscroll-behavior: contain;

		/* Makes focus outlines easier to see */
		outline: none;

		transform: translateY(0) scale(1);

		transition:
			transform 150ms ease,
			opacity 150ms ease;
	}

	.backdrop:not(.open) .popup {
		transform: translateY(0.5rem) scale(0.98);
		opacity: 0;
	}

	.backdrop.open .popup {
		transform: translateY(0) scale(1);
		opacity: 1;
	}

	/* Small screens */
	@media (max-width: 480px) {
		.backdrop {
			align-items: flex-end;
			padding: 0;
		}

		.popup {
			width: 100%;
			max-height: min(90dvh, 42rem);
		}

		.backdrop:not(.open) .popup {
			transform: translateY(1rem);
		}
	}

	/* Reduced motion */
	@media (prefers-reduced-motion: reduce) {
		.backdrop,
		.popup {
			transition: none;
		}
	}
</style>
