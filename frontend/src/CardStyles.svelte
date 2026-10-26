<!--
    Global styles for the card layout

    Backend still needs to provide per cards:
     * --c1 (Primary color)
     * --c2 (Secondary color)
     * --badge-color (Badge color)
-->

<style>
	:global(:root) {
		--card-border-radius: 0.5rem;
		--collection-line: var(--color-text);
	}

	:global(.card-container) {
		width: 30ch;
		height: 30em;

		position: relative;

		user-select: none;

		--mouse-x: 50%;
		--mouse-y: 50%;
		--mouse-distance: 0;
		--tilt-x: 0deg;
		--tilt-y: 0deg;
	}

	:global(.card-container[data-tilt]) {
		perspective: 1000px;
	}

	/* Tilting is mouse-only, touch devices keep scrolling over cards */
	@media (hover: hover) and (pointer: fine) {
		:global(.card-container[data-tilt]) {
			touch-action: none;
			cursor: grab;
		}

		:global(.card-container[data-tilt]:active) {
			cursor: grabbing;
		}
	}

	:global(.card-container[data-tilt] .card) {
		transform: rotateX(var(--tilt-x)) rotateY(var(--tilt-y));
		transform-style: preserve-3d;

		transition:
			transform 0.15s ease-out,
			box-shadow 0.3s ease;

		will-change: transform;
	}

	:global(.card-container[data-tilt] .collection) {
		transform: rotateX(var(--tilt-x)) rotateY(var(--tilt-y));
		transform-style: preserve-3d;

		transition:
			transform 0.15s ease-out,
			box-shadow 0.3s ease;

		will-change: transform;
	}

	:global(.card) {
		width: 100%;
		height: 100%;
		position: absolute;
		inset: 0;

		display: flex;
		flex-direction: column;
		box-sizing: border-box;
		overflow: hidden;

		/* For styles */
		isolation: isolate;

		border-radius: var(--card-border-radius);
		background: radial-gradient(
				120% 90% at 0% 40%,
				color-mix(in srgb, var(--c1), transparent 50%),
				transparent 60%
			),
			radial-gradient(
				120% 90% at 100% 100%,
				color-mix(in srgb, var(--c2), transparent 50%),
				transparent 60%
			),
			linear-gradient(
				160deg,
				color-mix(
					in srgb,
					color-mix(in srgb, var(--c1), transparent 80%),
					black 20%
				),
				color-mix(
					in srgb,
					color-mix(in srgb, var(--c2), transparent 80%),
					black 20%
				)
			),
			var(--color-bg-1);
	}

	/* Card components */
	:global(span.card-badge) {
		z-index: 12;
		position: absolute;
		top: 0.5em;
		left: 1ch;
		align-self: flex-start;
		padding: 0.5ch 0.8ch;

		font-size: 0.7rem;
		letter-spacing: 0.1ch;
		font-weight: 800;
		color: white;
		text-shadow: 0 0 2px color-mix(in srgb, var(--badge-color), black 50%);

		border-radius: 10ch;
		background: linear-gradient(
			160deg,
			color-mix(
				in srgb,
				color-mix(in srgb, var(--badge-color), transparent 50%),
				black 20%
			),
			color-mix(
				in srgb,
				color-mix(in srgb, var(--badge-color), transparent 50%),
				white 50%
			)
		);

		backdrop-filter: blur(1px);
	}

	:global(.card img.card-image) {
		z-index: 11;
		border-radius: var(--card-border-radius) var(--card-border-radius) 0 0;

		width: 100%;
		flex: 0 0 40%;
		height: 40%;
		object-fit: cover;
		object-position: center;
	}

	:global(.card img.collection-image) {
		z-index: 11;
		border-radius: var(--card-border-radius) var(--card-border-radius) 0 0;

		width: 100%;
		flex: 0 0 60%;
		height: 60%;
		object-fit: cover;
		object-position: center;
	}

	:global(h2.card-title) {
		z-index: 10;
		margin: 0.3ch 0.5ch 0.1ch 0.5ch;
		flex: 0 0 auto;
	}

	:global(p.card-description) {
		z-index: 10;
		flex: 1 1 auto;
		min-height: 0;
		overflow: hidden;
		margin: 1ch 1ch;

		text-align: justify;
		line-height: 1.1;
		color: color-mix(in srgb, var(--color-text) 75%, transparent);
	}

	:global(div.card-footer) {
		z-index: 10;
		position: static;
		flex: 0 0 auto;
		display: inline-flex;
		width: 90%;
		margin: 0 auto 0.4ch;
		border-top: 1px solid rgba(127, 127, 127, 0.5);
	}

	:global(a.card-collection) {
		z-index: 10;
		font-weight: 600;
		letter-spacing: -0.05ch;
		margin-right: auto;
		text-decoration: none;
		color: var(--color-text);
	}

	:global(span.card-number) {
		z-index: 10;
		color: color-mix(in srgb, var(--color-text) 30%, transparent);
	}

	/* Collections */
	:global(.collection) {
		background: linear-gradient(
			124deg,
			var(--c1) 0%,
			color-mix(in srgb, var(--c2), var(--color-bg-1) 50%) 19%,
			color-mix(in srgb, var(--c1), var(--color-bg-1) 30%) 42%,
			color-mix(in srgb, var(--c2), var(--color-bg-1) 50%) 70%,
			color-mix(in srgb, var(--c1), var(--color-bg-1) 50%) 83%,
			var(--c2) 100%
		);
		border-radius: 0.5rem;

		border: 1px solid
			color-mix(in srgb, var(--collection-line), transparent 65%);

		width: 100%;
		height: 100%;
		position: absolute;
		inset: 0;

		box-sizing: border-box;
		overflow: hidden;

		isolation: isolate;
	}

	:global(.collection::before) {
		position: absolute;
		inset: 0;
		z-index: 9;

		content: "";
		background: linear-gradient(
				115deg,
				transparent 13%,
				rgba(255, 255, 255, 0.19) 21%,
				transparent 29%,
				transparent 58%,
				rgba(255, 255, 255, 0.12) 62%,
				transparent 68%
			),
			linear-gradient(
				90deg,
				rgba(255, 255, 255, 0.12),
				transparent 9%,
				transparent 91%,
				rgba(0, 0, 0, 0.22)
			);
		mix-blend-mode: screen;
	}

	:global(.collection-crimp) {
		position: absolute;
		left: 0;
		z-index: 12;
		width: 100%;
		height: 13px;
		background: repeating-linear-gradient(
			90deg,
			rgba(255, 255, 255, 0.55) 0 1px,
			rgba(8, 12, 40, 0.45) 1px 3px,
			rgba(202, 218, 255, 0.34) 3px 4px
		);
		opacity: 0.58;
	}

	:global(.collection-content) {
		position: absolute;

		top: 2%;
		left: 3%;
		width: 94%;
		height: 96%;

		border: 1px solid
			color-mix(in srgb, var(--collection-line), transparent 65%);
		border-radius: 0.3rem;
	}

	:global(.collection-rays) {
		position: absolute;
		z-index: -1;
		inset: 12% -30% 22%;
		opacity: 0.6;
		background: repeating-conic-gradient(
			from 8deg at 50% 53%,
			rgba(255, 255, 255, 0.2) 0deg 1.2deg,
			transparent 1.2deg 13deg
		);
		mask-image: radial-gradient(
			ellipse at center,
			black 0%,
			transparent 70%
		);
	}

	:global(.collection-image) {
		z-index: 11;
		width: 95%;
		height: 60%;
		margin-left: 2.5%;
		margin-top: 0.5em;
		object-fit: cover;
		object-position: center;
		border-radius: 50%;

		mask-image: radial-gradient(
			circle closest-side,
			black calc(100% - 8px),
			transparent 100%
		);
	}

	:global(.collection-title) {
		text-align: center;
		margin: 0.5rem 0 0 0;

		font-size: 2rem;
		font-weight: 800;
		letter-spacing: -0.015em;
		text-shadow: 1px 2px 4px var(--color-bg-1);
	}

	:global(.collection-description) {
		z-index: 11;
		width: fit-content;
		padding: 0 0.8ch;
		overflow: hidden;

		text-align: center;
		font-weight: 600;
		margin: auto;

		background: linear-gradient(45deg, var(--c1), var(--c2));
		opacity: 0.9;
		text-shadow: 1px 2px 4px color-mix(in srgb, var(--c1), black 50%);
		clip-path: polygon(4% 0, 100% 0, 96% 100%, 0 100%);
	}

	:global(.collection-footer) {
		z-index: 10;
		position: absolute;
		bottom: 0.3em;
		left: 5%;

		width: 90%;
		border-top: solid 1px
			color-mix(in srgb, var(--collection-line), transparent 50%);
		font-weight: 600;
		font-size: 0.7rem;
		letter-spacing: -0.055em;

		opacity: 0.9;

		display: inline-flex;
	}
</style>
