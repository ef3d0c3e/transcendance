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
        let active = null; // card currently under cursor
        let pending = null; // latest pointermove, flushed once per frame
        let frame = 0;

        function flush() {
            frame = 0;
            if (!pending) return;

            const { target, clientX, clientY } = pending;
            pending = null;

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

            const rect = active.getBoundingClientRect();
            const x = ((clientX - rect.left) / rect.width) * 100;
            const y = ((clientY - rect.top) / rect.height) * 100;
            const maxTilt = Number(active.dataset.tiltMax) || DEFAULT_MAX_TILT;

            setPointer(active, x, y, maxTilt);
        }

        function release() {
            pending = null;
            if (active) resetCard(active);
            active = null;
        }

        document.addEventListener(
            "pointermove",
            (e) => {
                pending = {
                    target: e.target,
                    clientX: e.clientX,
                    clientY: e.clientY,
                };
                if (!frame) frame = requestAnimationFrame(flush);
            },
            {
                passive: true,
            },
        );
        document.addEventListener("pointerout", (e) => {
            // Pointer left the window
            if (!e.relatedTarget) release();
        });
        document.addEventListener("pointerup", (e) => {
                if (e.pointerType !== "mouse") release();
        });
        document.addEventListener("pointercancel", (e) => {
                if (e.pointerType !== "mouse") release();
        });

        // Effects: prepare what's on the page now, then anything added later
        prepareEffectsIn(document.body);

        const observer = new MutationObserver((mutations) => {
            for (const mutation of mutations) {
                mutation.addedNodes.forEach(prepareEffectsIn);
            }
        });
        observer.observe(document.body, { childList: true, subtree: true });

        return () => {
            document.removeEventListener("pointermove", onPointerMove);
            document.removeEventListener("pointerout", onPointerOut);
            document.removeEventListener("pointerup", onPointerEnd);
            document.removeEventListener("pointercancel", onPointerEnd);
            observer.disconnect();
            if (frame) cancelAnimationFrame(frame);
            release();
        };
    }
</script>

<!-- Inject CSS -->
<CardStyles />
<CardEffectStyles />
