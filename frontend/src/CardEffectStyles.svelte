<!--
    Stackable effects for cards
-->

<style>
	:global {
        /* Base effects data */
		.card-effect {
			--card-effect-z: 10;
			--card-effect-opacity: 0.5;
			--card-effect-rest: 1; /* opacity multiplier when not hovered */
			--card-effect-scale: 1; /* texture / mask tile scale */
			--card-effect-angle: 125deg;
			--card-effect-hue: 0deg;
			--card-effect-color: #ffffff;
			--card-effect-mask: none;

			--fx-x: var(--mouse-x, 50%);
			--fx-y: var(--mouse-y, 50%);
			--fx-ix: calc(100% - var(--fx-x));
			--fx-iy: calc(100% - var(--fx-y));

			position: absolute;
			inset: 0;
			z-index: var(--card-effect-z);

			width: 100%;
			height: 100%;

			pointer-events: none;
			border-radius: inherit;
		}

		:is(.card-foil, .card-rainbow, .card-sparkle, .card-prism) {
			--card-effect-rest: 0.35;
			opacity: calc(var(--card-effect-opacity) * var(--card-effect-rest));
			transition: opacity 0.3s ease;
		}

		.card:hover :is(.card-foil, .card-rainbow, .card-sparkle, .card-prism) {
			--card-effect-rest: 1;
		}

		:is(.card-foil, .card-rainbow, .card-prism) {
			mask-image: var(--card-effect-mask);
			mask-mode: luminance;
			mask-size: calc(200% * var(--card-effect-scale));
			mask-position: center;
			mask-repeat: repeat;
		}

        /* Effect: glare */
		.card-glare {
			background:
				radial-gradient(
					circle at var(--fx-ix) var(--fx-iy),
					rgba(255, 255, 255, 0.55),
					transparent 70%
				),
				radial-gradient(
					circle at var(--fx-ix) var(--fx-iy),
					rgba(255, 255, 255, 0.08),
					transparent max(0%, calc(var(--mouse-distance, 60%) - 10%))
				);

			mix-blend-mode: var(--card-effect-blend, overlay);
			opacity: var(--card-effect-opacity);
			transition: opacity 0.3s ease;
		}

        /* Effect: Watermark */
		.card-watermark {
			--card-effect-opacity: 0.2;
			--card-effect-transform: matrix(1, 0, 0, 1, 0, 0);

			background-image: var(--card-effect-watermark-image);
			background-size: 64px 64px;
			background-repeat: repeat;
			background-position: center;

			opacity: var(--card-effect-opacity, 1);
			transform: var(--card-effect-transform, none);

			mix-blend-mode: overlay;
		}

        /* Effect: Foil */
		.card-foil {
			--card-effect-opacity: 0.5;

			background:
				/* sheen band following the pointer */
				linear-gradient(
					var(--card-effect-angle),
					transparent calc(var(--fx-x) - 30%),
					rgba(255, 255, 255, 0.95) var(--fx-x),
					transparent calc(var(--fx-x) + 30%)
				),
				/* brushed streaks, opposite parallax */
				repeating-linear-gradient(
					90deg,
					rgba(255, 255, 255, 0.22) 0 1px,
					rgba(0, 0, 0, 0.22) 1px 3px
				),
				/* base metal color */
				var(--card-effect-color);

			background-size: 100% 100%, 140% 140%, auto;
			background-position: center, var(--fx-ix) var(--fx-iy), center;
			background-blend-mode: screen, overlay, normal;

			mix-blend-mode: var(--card-effect-blend, overlay);
			filter: brightness(1.15) contrast(1.35) saturate(1.2) hue-rotate(var(--card-effect-hue));
		}

        /* Effect: Rainbow */
		.card-rainbow {
			--card-effect-opacity: 0.45;
			--card-effect-band: 0.3;
			--card-effect-spectrum:
				#ff2d95 0%,
				#ffb400 calc(7% * var(--card-effect-band)),
				#f4ff3a calc(14% * var(--card-effect-band)),
				#2dff9a calc(21% * var(--card-effect-band)),
				#2dc8ff calc(28% * var(--card-effect-band)),
				#7a4dff calc(35% * var(--card-effect-band)),
				#ff2d95 calc(42% * var(--card-effect-band));

			background:
				repeating-linear-gradient(var(--card-effect-angle), var(--card-effect-spectrum)),
				radial-gradient(
					farthest-corner circle at var(--fx-ix) var(--fx-iy),
					rgb(235, 235, 235) 0%,
					rgb(70, 70, 70) 70%
				);

			background-size: 300% 300%, 100% 100%;
			background-position: var(--fx-x) var(--fx-y), center;
			background-blend-mode: hard-light, normal;

			mix-blend-mode: var(--card-effect-blend, color-dodge);
			filter: brightness(0.9) contrast(1.4) saturate(1.5) hue-rotate(var(--card-effect-hue));
		}

        /* Effect: Point sparkle */
		.card-sparkle {
			--card-effect-opacity: 0.7;
			--card-effect-rest: 1; /* always on, no hover dependency */
			--card-effect-speed: 2;

			/* Wandering visibility window */
			mask-image: radial-gradient(
				circle at 50% 50%,
				#000 0%,
				rgba(0, 0, 0, 0.35) 40%,
				transparent 70%
			);
			mask-size: 220% 220%;
			mask-repeat: no-repeat;
			animation: sparkle-sweep calc(18s / var(--card-effect-speed)) ease-in-out infinite;

			mix-blend-mode: var(--card-effect-blend, plus-lighter);
			filter: hue-rotate(var(--card-effect-hue));
		}

		.card-sparkle::before,
		.card-sparkle::after {
			content: '';
			position: absolute;
			inset: 0;
			border-radius: inherit;
			background-repeat: repeat;
		}

		.card-sparkle::before {
			background-image:
				radial-gradient(circle at 23% 61%, #fff 0 1px, transparent 1.6px),
				radial-gradient(circle at 71% 18%, #ffe9a8 0 1px, transparent 1.6px);
			background-size: 31px 37px, 53px 47px;
			animation:
				sparkle-drift-a calc(26s / var(--card-effect-speed)) linear infinite,
				sparkle-twinkle calc(3.1s / var(--card-effect-speed)) ease-in-out infinite;
		}

		.card-sparkle::after {
			background-image:
				radial-gradient(circle at 44% 83%, #a8dcff 0 1.2px, transparent 2px),
				radial-gradient(circle at 12% 27%, #fff 0 0.8px, transparent 1.3px);
			background-size: 71px 67px, 23px 29px;
			animation:
				sparkle-drift-b calc(34s / var(--card-effect-speed)) linear infinite,
				sparkle-twinkle calc(4.3s / var(--card-effect-speed)) ease-in-out -1.9s infinite;
		}
	}

	@keyframes -global-sparkle-drift-a {
		to {
			background-position: 31px 37px, -53px 47px;
		}
	}

	@keyframes -global-sparkle-drift-b {
		to {
			background-position: -71px 67px, 23px -29px;
		}
	}

	@keyframes -global-sparkle-twinkle {
		0%,
		100% {
			opacity: 0.2;
		}

		50% {
			opacity: 1;
		}
	}

	@keyframes -global-sparkle-sweep {
		0%,
		100% {
			mask-position: 0% 0%;
		}

		25% {
			mask-position: 100% 30%;
		}

		50% {
			mask-position: 70% 100%;
		}

		75% {
			mask-position: 10% 60%;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		:global(.card-sparkle),
		:global(.card-sparkle::before),
		:global(.card-sparkle::after) {
			animation: none;
		}

		:global(.card-sparkle) {
			mask-position: 50% 50%;
		}
	}
</style>
