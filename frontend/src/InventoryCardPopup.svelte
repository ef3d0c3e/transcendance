<!--
    Popup with instance details (acquisition date) when clicking an inventory card

    Cards must provide `data-acquired` (RFC 3339 date)
-->

<script>
	import { onMount } from "svelte";

	let { target } = $props();

	const CARD_SELECTOR = ".inventory-cards .card-container[data-acquired]";
	// Pointer travel (px) after which a press counts as a tilt drag, not a click
	const DRAG_THRESHOLD = 6;

	let dialog;
	let title = $state("");
	let acquired = $state("");
	let acquiredIso = $state("");
	let returnFocus = null;

	const labelAcquired = $derived(target.dataset.labelAcquired ?? "");
	const labelClose = $derived(target.dataset.labelClose ?? "");
	const dateFormat = $derived(
		new Intl.DateTimeFormat(target.dataset.locale || undefined, {
			dateStyle: "long",
			timeStyle: "short",
		}),
	);

	function show(card) {
		title = card.querySelector(".card-title")?.textContent ?? "";
		acquiredIso = card.dataset.acquired;
		acquired = dateFormat.format(new Date(acquiredIso));
		returnFocus = card;
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

		function cardFrom(e) {
			if (!(e.target instanceof Element)) return null;
			// Let the collection link work
			if (e.target.closest(".card-collection")) return null;
			return e.target.closest(CARD_SELECTOR);
		}

		function onPointerDown(e) {
			press = { x: e.clientX, y: e.clientY };
		}

		function onClick(e) {
			const card = cardFrom(e);
			if (!card) return;

			// Dragging to tilt the card isn't a click
			if (press && Math.hypot(e.clientX - press.x, e.clientY - press.y) > DRAG_THRESHOLD) return;
			show(card);
		}

		function onKeyDown(e) {
			if (e.key !== "Enter" && e.key !== " ") return;
			const card = cardFrom(e);
			if (!card || e.target !== card) return;

			e.preventDefault();
			show(card);
		}

		// Cards open a dialog, make them reachable and announced as buttons
		for (const card of document.querySelectorAll(CARD_SELECTOR)) {
			card.tabIndex = 0;
			card.setAttribute("role", "button");
			card.setAttribute("aria-haspopup", "dialog");
			card.setAttribute("aria-label", card.querySelector(".card-title")?.textContent ?? "");
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
	class="inventory-popup"
	aria-labelledby="inventory-popup-title"
	bind:this={dialog}
	onclick={onDialogClick}
	onclose={onDialogClose}
>
	<div class="inventory-popup-content">
		<h3 id="inventory-popup-title">{title}</h3>
		<dl>
			<dt>{labelAcquired}</dt>
			<dd><time datetime={acquiredIso}>{acquired}</time></dd>
		</dl>
		<button type="button" onclick={close}>{labelClose}</button>
	</div>
</dialog>

<style>
	.inventory-popup {
		width: min(100% - 2rem, 22rem);
		padding: 0;

		color: var(--color-text);
		background: var(--color-bg-2);
		border: 1px solid var(--color-border-1);
		border-radius: 1rem;
		box-shadow: 0 10px 30px var(--color-shadow);
	}

	.inventory-popup::backdrop {
		background: rgb(0 0 0 / 55%);
		backdrop-filter: blur(3px);
	}

	.inventory-popup[open] {
		animation: popup-in 150ms ease;
	}

	.inventory-popup-content {
		display: flex;
		flex-direction: column;
		gap: 1rem;

		padding: 1.25rem 1.25rem 1rem;
	}

	h3 {
		margin: 0;

		font-size: 1.25rem;
		font-weight: 650;
		letter-spacing: -0.01em;
	}

	dl {
		margin: 0;
	}

	dt {
		color: var(--color-text-muted);
		font-size: 0.875rem;
		font-weight: 600;
	}

	dd {
		margin: 0.15rem 0 0;

		font-size: 1.05rem;
		font-variant-numeric: tabular-nums;
	}

	button {
		align-self: flex-end;
	}

	@keyframes popup-in {
		from {
			opacity: 0;
			transform: translateY(0.5rem) scale(0.98);
		}
	}

	/* Small screens: bottom sheet, like the other popups */
	@media (max-width: 480px) {
		.inventory-popup {
			width: 100%;
			max-width: 100%;
			margin: auto 0 0;

			border-radius: 1rem 1rem 0 0;
			border-bottom: 0;
		}

		.inventory-popup-content {
			padding-bottom: max(1rem, env(safe-area-inset-bottom));
		}

		button {
			align-self: stretch;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.inventory-popup[open] {
			animation: none;
		}
	}
</style>
