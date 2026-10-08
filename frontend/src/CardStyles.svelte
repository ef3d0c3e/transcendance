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
	}

	:global(.card-container) {
		width: 30ch;
		height: 30em;

		position: relative;

		--mouse-x: 50%;
		--mouse-y: 50%;
		--mouse-distance: 0%;
		--tilt-x: 0deg;
		--tilt-y: 0deg;
	}

	:global(.card-container[data-tilt]) {
		perspective: 1000px;

		cursor: grab;
	}

	:global(.card-container[data-tilt]:active) {
		cursor: grabbing;
	}

	:global(.card-container[data-tilt] .card) {
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

	:global(.card img) {
		z-index: 11;
		border-radius: var(--card-border-radius) var(--card-border-radius) 0 0;

		width: 100%;
		flex: 0 0 40%;
		height: 40%;
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
</style>
