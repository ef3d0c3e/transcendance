<script>
	import { onMount } from "svelte";
	import CardStyles from "./CardStyles.svelte";
	import CardEffectStyles from "./CardEffectStyles.svelte";

	onMount(() => initCards());

	const CARD_SELECTOR = ".card-container";
	const DEFAULT_MAX_TILT = 10; // degrees, overriden by data-tilt-max

	// Data attributes
	const EFFECT_VARS = {
		z: "--card-effect-z",
		opacity: "--card-effect-opacity",
		rest: "--card-effect-rest", // opacity multiplier when not hovered
		transform: "--card-effect-transform",
		scale: "--card-effect-scale", // mask tile scale
		angle: "--card-effect-angle", // e.g. "125deg"
		hue: "--card-effect-hue", // e.g. "90deg" (shifts the palette)
		color: "--card-effect-color",
		band: "--card-effect-band", // rainbow: color band width
		speed: "--card-effect-speed", // sparkle: animation speed multiplier
		blend: "--card-effect-blend", // any mix-blend-mode value
		spectrum: "--card-effect-spectrum", // rainbow: "#f00 0%, #0f0 10%, ..."
	};

	// Data attributes wrapped in url(...)
	const EFFECT_URLS = {
		mask: "--card-effect-mask",
		watermark: "--card-effect-watermark-image",
	};

	const cssUrl = (path) => `url("${String(path).replace(/"/g, "%22")}")`;

	/* Effects */
	const prepared = new WeakSet();

	function prepareEffect(effect) {
		if (prepared.has(effect)) return;
		prepared.add(effect);

		const data = effect.dataset;

		for (const [key, variable] of Object.entries(EFFECT_VARS)) {
			if (data[key] !== undefined)
				effect.style.setProperty(variable, data[key]);
		}

		for (const [key, variable] of Object.entries(EFFECT_URLS)) {
			if (data[key] !== undefined)
				effect.style.setProperty(variable, cssUrl(data[key]));
		}
	}

	function prepareEffectsIn(node) {
		if (!(node instanceof Element)) return;

		if (node.matches(".card-effect")) prepareEffect(node);
		node.querySelectorAll(".card-effect").forEach(prepareEffect);
	}

	/* Tilt and mouse cursor handing */
	function setPointer(card, x, y, maxTilt) {
		const dist = Math.hypot(x / 100 - 0.5, y / 100 - 0.5) * 100;
		const nx = (x - 50) / 50;
		const ny = (y - 50) / 50;

		card.style.setProperty("--mouse-x", `${x}%`);
		card.style.setProperty("--mouse-y", `${y}%`);
		card.style.setProperty("--mouse-distance", `${dist}%`);
		card.style.setProperty("--tilt-x", `${ny * maxTilt}deg`);
		card.style.setProperty("--tilt-y", `${-nx * maxTilt}deg`);
	}

	function resetCard(card) {
		card.dataset.hovered = "false";

		card.style.setProperty("--mouse-distance", "0%");
		card.style.setProperty("--mouse-x", "50%");
		card.style.setProperty("--mouse-y", "50%");
		card.style.setProperty("--tilt-x", "0deg");
		card.style.setProperty("--tilt-y", "0deg");
	}

	export function initCards() {
		let active = null; // card currently hovered
		let drag = null; // { card, pointerId } while pressed
		let pending = null;
		let frame = 0;

		function tiltFrom(card, clientX, clientY) {
			const rect = card.getBoundingClientRect();
			const rawX = ((clientX - rect.left) / rect.width) * 100;
			const rawY = ((clientY - rect.top) / rect.height) * 100;

			// Clamp so dragging past the card edge pins to max tilt
			// instead of over-rotating.
			const x = Math.max(0, Math.min(100, rawX));
			const y = Math.max(0, Math.min(100, rawY));

			const maxTilt = Number(card.dataset.tiltMax) || DEFAULT_MAX_TILT;
			setPointer(card, x, y, maxTilt);
		}

		function flush() {
			frame = 0;
			if (!pending) return;

			const { target, clientX, clientY } = pending;
			pending = null;

			// While dragging, pointermove drives tilt directly; skip hover.
			if (drag) return;

			const card =
				target instanceof Element
					? target.closest(CARD_SELECTOR)
					: null;

			if (card !== active) {
				if (active) resetCard(active);
				active = card;
				if (active) active.dataset.hovered = "true";
			}
			if (!active) return;

			tiltFrom(active, clientX, clientY);
		}

		function release() {
			pending = null;
			if (active) resetCard(active);
			active = null;
		}

		function onPointerDown(e) {
			if (e.pointerType === "mouse" && e.button !== 0) return;

			// Allow clicking on the link
			if (e.target instanceof Element && e.target.closest(".card-collection"))
			{
				return;
			}

			const card =
				e.target instanceof Element
					? e.target.closest(CARD_SELECTOR)
					: null;
			if (!card) return;

			if (active && active !== card) resetCard(active);
			active = card;
			card.dataset.hovered = "true";
			card.dataset.dragging = "true";

			drag = { card, pointerId: e.pointerId };

			card.setPointerCapture(e.pointerId);

			// Kill any in-flight selection that started elsewhere.
			window.getSelection()?.removeAllRanges();

			tiltFrom(card, e.clientX, e.clientY);
		}

		function onPointerMove(e) {
			if (drag && e.pointerId === drag.pointerId) {
				tiltFrom(drag.card, e.clientX, e.clientY);
				return;
			}

			pending = {
				target: e.target,
				clientX: e.clientX,
				clientY: e.clientY,
			};
			if (!frame) frame = requestAnimationFrame(flush);
		}

		function endDrag(e) {
			if (!drag || e.pointerId !== drag.pointerId) return;

			const { card } = drag;
			drag = null;

			try {
				card.releasePointerCapture(e.pointerId);
			} catch {}
			delete card.dataset.dragging;

			// If the pointer was released outside the card, clear hover state.
			// If inside, keep hover as-is so it transitions back smoothly.
			const rect = card.getBoundingClientRect();
			const inside =
				e.clientX >= rect.left &&
				e.clientX <= rect.right &&
				e.clientY >= rect.top &&
				e.clientY <= rect.bottom;

			if (!inside) {
				resetCard(card);
				if (active === card) active = null;
			}
		}

		function onPointerOut(e) {
			if (drag) return; // don't reset while captured
			if (!e.relatedTarget) release();
		}

		function onPointerEnd(e) {
			if (drag) return; // pointerup is handled by endDrag
			if (e.pointerType !== "mouse") release();
		}

		document.addEventListener("pointerdown", onPointerDown);
		document.addEventListener("pointermove", onPointerMove, {
			passive: true,
		});
		document.addEventListener("pointerup", endDrag);
		document.addEventListener("pointercancel", endDrag);
		document.addEventListener("pointerout", onPointerOut);
		document.addEventListener("pointerup", onPointerEnd);
		document.addEventListener("pointercancel", onPointerEnd);

		// Effects: prepare what's on the page now, then anything added later
		prepareEffectsIn(document.body);

		const observer = new MutationObserver((mutations) => {
			for (const mutation of mutations) {
				mutation.addedNodes.forEach(prepareEffectsIn);
			}
		});
		observer.observe(document.body, { childList: true, subtree: true });

		return () => {
			document.removeEventListener("pointerdown", onPointerDown);
			document.removeEventListener("pointermove", onPointerMove);
			document.removeEventListener("pointerup", endDrag);
			document.removeEventListener("pointercancel", endDrag);
			document.removeEventListener("pointerout", onPointerOut);
			document.removeEventListener("pointerup", onPointerEnd);
			document.removeEventListener("pointercancel", onPointerEnd);
			observer.disconnect();
			if (frame) cancelAnimationFrame(frame);
			release();
			drag = null;
		};
	}
</script>

<!-- Inject CSS -->
<CardStyles />
<CardEffectStyles />
