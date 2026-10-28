<!--
    Asks to open a booster when clicking it in the inventory

    Boosters must provide `data-booster-open` (opening page URL) and
    `data-booster-confirm` (the question to ask)
-->

<script>
	import { onMount } from "svelte";

	let { target } = $props();

	const BOOSTER_SELECTOR = ".inventory-booster[data-booster-open]";
	const TILE_SELECTOR = `${BOOSTER_SELECTOR} .card-container`;
	// Pointer travel (px) after which a press counts as a tilt drag, not a click
	const DRAG_THRESHOLD = 6;

	let dialog;
	let question = $state("");
	let openUrl = $state("");
	let returnFocus = null;

	const labelYes = $derived(target.dataset.labelYes ?? "");
	const labelNo = $derived(target.dataset.labelNo ?? "");

	function show(tile) {
		const booster = tile.closest(BOOSTER_SELECTOR);
		question = booster.dataset.boosterConfirm;
		openUrl = booster.dataset.boosterOpen;
		returnFocus = tile;
		dialog.showModal();
	}

	function close() {
		dialog.close();
	}

	function onDialogClose() {
		returnFocus?.focus();
		returnFocus = null;
	}

	// Clicking the backdrop targets the dialog itself
	function onDialogClick(e) {
		if (e.target === dialog) close();
	}

	onMount(() => {
		let press = null;

		function tileFrom(e) {
			if (!(e.target instanceof Element)) return null;
			return e.target.closest(TILE_SELECTOR);
		}

		function onPointerDown(e) {
			press = { x: e.clientX, y: e.clientY };
		}

		function onClick(e) {
			const tile = tileFrom(e);
			if (!tile) return;

			// Dragging to tilt the booster isn't a click
			if (press && Math.hypot(e.clientX - press.x, e.clientY - press.y) > DRAG_THRESHOLD) return;
			show(tile);
		}

		function onKeyDown(e) {
			if (e.key !== "Enter" && e.key !== " ") return;
			const tile = tileFrom(e);
			if (!tile || e.target !== tile) return;

			e.preventDefault();
			show(tile);
		}

		// Boosters open a dialog, make them reachable and announced as buttons
		for (const tile of document.querySelectorAll(TILE_SELECTOR)) {
			tile.tabIndex = 0;
			tile.setAttribute("role", "button");
			tile.setAttribute("aria-haspopup", "dialog");
			tile.setAttribute("aria-label", tile.closest(BOOSTER_SELECTOR).dataset.boosterConfirm);
		}

		document.addEventListener("pointerdown", onPointerDown, { passive: true });
		document.addEventListener("click", onClick);
		document.addEventListener("keydown", onKeyDown);

		return () => {
			document.removeEventListener("pointerdown", onPointerDown);
			document.removeEventListener("click", onClick);
			document.removeEventListener("keydown", onKeyDown);
		};
	});
</script>

<dialog
	class="booster-popup"
	aria-labelledby="booster-popup-question"
	bind:this={dialog}
	onclick={onDialogClick}
	onclose={onDialogClose}
>
	<div class="booster-popup-content">
		<h3 id="booster-popup-question">{question}</h3>
		<div class="booster-popup-actions">
			<button type="button" class="booster-popup-no" onclick={close}>{labelNo}</button>
			<!-- svelte-ignore a11y_autofocus -->
			<a class="booster-popup-yes" href={openUrl} autofocus>{labelYes}</a>
		</div>
	</div>
</dialog>

<style>
	.booster-popup {
		width: min(100% - 2rem, 24rem);
		padding: 0;

		color: var(--color-text);
		background: var(--color-bg-2);
		border: 1px solid var(--color-border-1);
		border-radius: 1rem;
		box-shadow: 0 10px 30px var(--color-shadow);
	}

	.booster-popup::backdrop {
		background: rgb(0 0 0 / 55%);
		backdrop-filter: blur(3px);
	}

	.booster-popup[open] {
		animation: popup-in 150ms ease;
	}

	.booster-popup-content {
		display: flex;
		flex-direction: column;
		gap: 1.25rem;

		padding: 1.25rem;
	}

	h3 {
		margin: 0;

		font-size: 1.25rem;
		font-weight: 650;
		letter-spacing: -0.01em;
	}

	.booster-popup-actions {
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
	}

	/* No looks like the neutral buttons, yes like the page's open button */
	.booster-popup-no,
	.booster-popup-yes {
		display: inline-flex;
		align-items: center;
		justify-content: center;

		min-width: 5.5rem;
		min-height: 2.5rem;
		padding: 0.45rem 1rem;

		border: 0;
		border-radius: 0.55rem;

		font-weight: 600;
		line-height: 1.3;
		text-decoration: none;

		transition: background-color 120ms ease;
	}

	.booster-popup-no {
		color: var(--color-text);
		background: var(--color-bg-4);

		&:hover {
			background: color-mix(in srgb, var(--color-bg-4), black 10%);
		}
	}

	.booster-popup-yes {
		color: var(--color-text-alt);
		background: var(--color-accent-primary);

		&:hover {
			color: var(--color-text-alt);
			background: var(--color-accent-primary-dark);
		}

		&:focus-visible {
			outline: 3px solid color-mix(in srgb, var(--color-accent-secondary) 55%, transparent);
			outline-offset: 2px;
		}
	}

	@keyframes popup-in {
		from {
			opacity: 0;
			transform: translateY(0.5rem) scale(0.98);
		}
	}

	/* Small screens: bottom sheet, like the other popups */
	@media (max-width: 480px) {
		.booster-popup {
			width: 100%;
			max-width: 100%;
			margin: auto 0 0;

			border-radius: 1rem 1rem 0 0;
			border-bottom: 0;
		}

		.booster-popup-content {
			padding-bottom: max(1.25rem, env(safe-area-inset-bottom));
		}

		.booster-popup-no,
		.booster-popup-yes {
			flex: 1;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.booster-popup[open] {
			animation: none;
		}
	}
</style>
