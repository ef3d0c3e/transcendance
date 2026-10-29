<script>
	import { onMount, tick } from "svelte";

	let {
		formSelector,
		openLabel,
		retryLabel,
		errorLabel,
		autostart = false,
	} = $props();

	let reels = $state([]);
	let message = $state("");
	let failed = $state(false);
	let colors = $state({ primary: "#407fff", secondary: "#7f2fff" });
	let busy = $state(false);
	// Card scale in the reels, fitted to the viewport once they are mounted
	let reelScale = $state(null);

	let stageElement = $state();
	let timers = [];

	// Slot machine: five vertical reels side by side. Each drops past its
	// window on its own clock, overshoots the payline, then springs back up
	// onto the card it landed on.
	const SPIN_MIN_MS = 5000;
	const SPIN_MAX_MS = 9000;
	// Starts from rest, peaks at about twice the average speed a third of the
	// way in, then slows down onto the payline
	const SPIN_EASE = "cubic-bezier(0.4, 0.15, 0.3, 1)";
	const BOUNCE_MS = 280;
	const BOUNCE_EASE = "cubic-bezier(0.34, 1.3, 0.64, 1)";
	// How far the reel drops past the won card before springing back, in cards.
	// Must stay under half a card, so the won card never leaves the payline.
	const OVERSHOOT_CELLS = 0.3;
	// How far the stack is pulled up before it starts its drop, in cards.
	// Matches reelBelow on the server, the cards stacked beneath the winner
	const RUN_CELLS = 12;
	// Share of the viewport the reels may use
	const REELS_MAX_WIDTH = 0.92;
	const REELS_MAX_HEIGHT = 0.7;
	const REELS_MAX_SCALE = 1;
	// Time to look at the landed cards before going back to the open button
	const REVEAL_MS = 1200;
	// Period of the cards' sway, picked per reel so they never move in sync
	const SWAY_MIN_S = 2.6;
	const SWAY_MAX_S = 4.2;

	// Scale the cards so the reels fill the viewport width, while a whole
	// card stays visible in height
	function fitReels() {
		const inner = stageElement.querySelector(".pack-cell-inner");
		if (!inner) {
			return;
		}

		// offsetWidth/Height ignore the transform: the unscaled card size
		const cardWidth = inner.offsetWidth;
		const cardHeight = inner.offsetHeight;
		const gap = Number.parseFloat(getComputedStyle(stageElement).columnGap) || 0;
		const count = reels.length;

		const width = Math.min(window.innerWidth * REELS_MAX_WIDTH, window.innerWidth - 32);
		const byWidth = (width - (count - 1) * gap) / count / cardWidth;
		const byHeight = (window.innerHeight * REELS_MAX_HEIGHT) / cardHeight;

		reelScale = Math.min(byWidth, byHeight, REELS_MAX_SCALE);
	}

	function scrollBehavior() {
		return window.matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth";
	}

	function scrollToReels() {
		stageElement.scrollIntoView({ behavior: scrollBehavior(), block: "center" });
	}

	// Back to the top of the page, where the open button is
	function scrollToForm(form, button) {
		window.scrollTo({ top: 0, behavior: scrollBehavior() });
		const target = button.hidden ? form.querySelector("a") : button;
		target?.focus({ preventScroll: true });
	}

	// Failure reported by the server, its message is shown as is
	class PackError extends Error {}

	function showFailure(text) {
		failed = true;
		message = text;
		reels = [];
	}

	function clearTimers() {
		for (const timer of timers) {
			clearTimeout(timer);
		}
		timers = [];
	}

	function delay(ms) {
		return new Promise((resolve) => {
			setTimeout(resolve, ms);
		});
	}

	async function requestPack(form) {
		const response = await fetch(form.action, {
			method: form.method,
			body: new FormData(form),
		});
		const data = await response.json();

		if (!response.ok) {
			throw new PackError(data.message ?? errorLabel);
		}

		const { reels: stacks, colors: palette, next, remaining: left, remainingLabel: leftLabel, owned } = data;

		return {
			reels: stacks ?? [],
			palette: palette ?? colors,
			next: next ?? 0,
			remaining: left ?? 0,
			remainingLabel: leftLabel ?? "",
			owned,
		};
	}

	// One reel per drawn card, tagged with its rarity for the landing flare
	function buildReels(stacks) {
		return stacks.map((slots) => {
			const winner = slots.find((cell) => cell.win);

			const sway = SWAY_MIN_S + Math.random() * (SWAY_MAX_S - SWAY_MIN_S);

			return {
				slots,
				rarity: winner?.rarity ?? "common",
				landed: false,
				// Negative delay: the sway starts mid-swing
				sway: `--sway-duration: ${sway.toFixed(2)}s; --sway-delay: -${(Math.random() * sway).toFixed(2)}s`,
			};
		});
	}

	// Distance between the top edges of two neighbouring cards
	function cellPitch(element) {
		const first = element.querySelector(".pack-cell");

		if (!first) {
			return 0;
		}

		const style = getComputedStyle(element);
		const gap = Number.parseFloat(style.rowGap || style.gap) || 0;

		return first.getBoundingClientRect().height + gap;
	}

	// The window is one card tall, so the won card lands at translateY(-i * pitch)
	function restOffset(element, winnerIndex) {
		return -winnerIndex * cellPitch(element);
	}

	// Drop from above the stack, past the payline, and stop just short of it
	function spinReel(element, winnerIndex, duration) {
		const pitch = cellPitch(element);
		const rest = restOffset(element, winnerIndex);
		const start = rest - pitch * RUN_CELLS;
		const overshoot = rest + pitch * OVERSHOOT_CELLS;

		element.style.transition = "none";
		element.style.transform = `translateY(${start}px)`;
		// Flush the start position so the transition actually runs
		element.getBoundingClientRect();
		element.style.transition = `transform ${duration}ms ${SPIN_EASE}`;
		element.style.transform = `translateY(${overshoot}px)`;
	}

	// Settle back up onto the card the reel stopped on
	function bounceReel(element, winnerIndex) {
		element.style.transition = `transform ${BOUNCE_MS}ms ${BOUNCE_EASE}`;
		element.style.transform = `translateY(${restOffset(element, winnerIndex)}px)`;
	}

	// Every reel drops at once on its own randomised clock
	function spinReels(stacks) {
		const elements = stageElement.querySelectorAll(".pack-strip");

		return stacks.map((slots, index) => {
			const duration = SPIN_MIN_MS + Math.random() * (SPIN_MAX_MS - SPIN_MIN_MS);
			const winnerIndex = Math.max(0, slots.findIndex((cell) => cell.win));
			const element = elements[index];

			spinReel(element, winnerIndex, duration);

			timers.push(setTimeout(() => {
				bounceReel(element, winnerIndex);
			}, duration));

			timers.push(setTimeout(() => {
				reels[index].landed = true;
			}, duration + BOUNCE_MS));

			return duration;
		});
	}

	async function openPack(form, button) {
		if (busy) {
			return;
		}

		busy = true;
		clearTimers();
		// Clear the stage so the previous reels cannot flash back into view
		reels = [];
		failed = false;
		message = "";
		button.disabled = true;

		try {
			const result = await requestPack(form);

			if (result.reels.length === 0) {
				showFailure(errorLabel);
				return;
			}

			// The booster is consumed: chain into the next one of the stack, if any
			const input = form.querySelector("input[name=booster]");
			if (result.next && input) {
				input.value = result.next;
			} else {
				button.hidden = true;
			}

			reels = buildReels(result.reels);
			colors = result.palette;

			// Wait for the stacks to mount, size them, then measure them
			await tick();
			fitReels();
			await tick();
			scrollToReels();
			const durations = spinReels(result.reels);

			await delay(Math.max(...durations) + BOUNCE_MS + 80);
			// Updated once the cards are revealed, not to spoil them
			updatePageInfo(result);
			button.textContent = retryLabel ?? openLabel;

			// The won cards stay on the reels, the open button is ready for the next one
			await delay(REVEAL_MS);
			scrollToForm(form, button);
		} catch (error) {
			showFailure(error instanceof PackError ? error.message : errorLabel);
		} finally {
			busy = false;
			button.disabled = false;
		}
	}

	// Refresh the stack and collection progress shown on the page
	function updatePageInfo(result) {
		const left = document.querySelector("#pack-remaining");
		if (left && result.remainingLabel) {
			left.textContent = result.remainingLabel;
		}

		const owned = document.querySelector("#pack-owned");
		const total = Number(document.querySelector("#pack-total")?.textContent);
		const bar = document.querySelector("#pack-progress-bar");
		if (owned && typeof result.owned === "number") {
			owned.textContent = String(result.owned);
			if (bar && total > 0) {
				bar.style.width = `${Math.floor((result.owned * 100) / total)}%`;
			}
		}
	}

	onMount(() => {
		const form = document.querySelector(formSelector);

		if (!form) {
			return;
		}

		function handleSubmit(event) {
			event.preventDefault();
			openPack(form, form.querySelector("button[type=submit]"));
		}

		form.addEventListener("submit", handleSubmit);

		// Coming from the inventory's open button: no need to click again
		if (autostart) {
			openPack(form, form.querySelector("button[type=submit]"));
		}

		return () => {
			form.removeEventListener("submit", handleSubmit);
			clearTimers();
		};
	});
</script>

{#if message}
	<section class={failed ? "section-error" : "section-success"}>{message}</section>
{/if}

{#if reels.length > 0}
	<div class="pack-reels" bind:this={stageElement}
		style="--c1: {colors.primary}; --c2: {colors.secondary}; --pack-count: {reels.length}; {reelScale ? `--pack-scale: ${reelScale}` : ''}">
		{#each reels as reel}
			<div class="pack-reel" class:is-landed={reel.landed} data-rarity={reel.rarity} style={reel.sway}>
				<div class="pack-strip">
					{#each reel.slots as cell}
						<div class="pack-cell" data-rarity={cell.rarity} data-win={cell.win}>
							<div class="pack-cell-inner">
								{@html cell.html}
							</div>
						</div>
					{/each}
				</div>
				<div class="pack-flash" aria-hidden="true"></div>
			</div>
		{/each}
		<div class="pack-payline" aria-hidden="true"></div>
	</div>
{/if}


<style>
	.pack-reels {
		--pack-scale: 0.5;
		--pack-cell: calc(30ch * var(--pack-scale));
		--pack-cell-height: calc(30em * var(--pack-scale));
		--pack-gap: 0.75rem;
		--pack-width: calc(var(--pack-count) * var(--pack-cell) + (var(--pack-count) - 1) * var(--pack-gap));

		position: relative;
		display: flex;
		gap: var(--pack-gap);
		width: var(--pack-width);
		/* Centered on the page column, overflowing it on both sides when wider */
		margin: 1.5rem calc((100% - var(--pack-width)) / 2);
	}

	/* Each column is a reel window: one card tall, the stack turning behind it */
	.pack-reel {
		position: relative;
		flex: 0 0 auto;
		width: var(--pack-cell);
		height: var(--pack-cell-height);
		/* The swaying card may spill a little out of its window, less than
			the gap so the neighbouring cards stay hidden */
		overflow: hidden;
		overflow: clip;
		overflow-clip-margin: calc(var(--pack-gap) / 2);
		border-radius: 0.5rem;
	}

	.pack-strip {
		display: flex;
		flex-direction: column;
		gap: var(--pack-gap);
		transform: translateY(0);
		will-change: transform;
	}

	.pack-cell {
		position: relative;
		flex: 0 0 auto;
		width: var(--pack-cell);
		height: var(--pack-cell-height);
	}

	/* The card markup is a fixed 30ch x 30em box; scale it down to the window */
	.pack-cell-inner {
		width: 30ch;
		height: 30em;
		transform: scale(var(--pack-scale));
		transform-origin: top left;
	}

	/* Reels are the machine, not the prize: cards on them never take
		pointer input and never wear their effects. Only the won card, once
		its reel locked, shows its foil and can be tilted. */
	.pack-reel .pack-cell {
		pointer-events: none;
	}

	.pack-reel.is-landed .pack-cell[data-win="true"] {
		pointer-events: auto;
	}

	/* A locked reel only keeps its won card, which can then sway and tilt
		freely without being cut by the window */
	.pack-reel.is-landed {
		overflow: visible;
	}

	.pack-reel.is-landed .pack-cell[data-win="false"] {
		visibility: hidden;
	}

	.pack-reel:not(.is-landed) .pack-cell :global(.card-effect),
	.pack-reel .pack-cell[data-win="false"] :global(.card-effect) {
		display: none;
	}

	/* Cards sway in their window on their reel's clock. translate and rotate
		leave transform to the landing pop. */
	.pack-cell {
		animation: pack-sway var(--sway-duration, 3.4s) ease-in-out var(--sway-delay, 0s) infinite alternate;
	}

	@media (prefers-reduced-motion: reduce) {
		.pack-cell {
			animation: none;
		}
	}

	/* The badge's backdrop blur renders broken inside a rotating card */
	.pack-cell :global(.card-badge) {
		backdrop-filter: none;
	}

	/* The single payline every reel stops against */
	.pack-payline {
		position: absolute;
		left: 0;
		right: 0;
		top: 50%;
		height: 3px;
		transform: translateY(-50%);
		z-index: 4;
		pointer-events: none;
		background: linear-gradient(90deg, transparent, #ffd76a 8%, #ffd76a 92%, transparent);
		box-shadow:
			0 0 14px rgb(255 215 106 / 80%),
			0 0 34px rgb(255 215 106 / 45%);
	}

	.pack-payline::before,
	.pack-payline::after {
		content: "";
		position: absolute;
		top: 50%;
		width: 0;
		height: 0;
		transform: translateY(-50%);
		border-top: 9px solid transparent;
		border-bottom: 9px solid transparent;
	}

	.pack-payline::before {
		left: 0;
		border-right: 13px solid #ffd76a;
	}

	.pack-payline::after {
		right: 0;
		border-left: 13px solid #ffd76a;
	}

	/* The flare lights the window the reel's card lands in */
	.pack-flash {
		position: absolute;
		inset: -8%;
		pointer-events: none;
		opacity: 0;
		z-index: 3;
		border-radius: 0.75rem;
	}

	.pack-reel[data-rarity="common"] .pack-flash {
		background: radial-gradient(circle, #407fff 0%, transparent 68%);
	}

	.pack-reel[data-rarity="rare"] .pack-flash {
		background: radial-gradient(circle, #ff1440 0%, transparent 68%);
	}

	.pack-reel[data-rarity="legendary"] .pack-flash {
		background: radial-gradient(circle, #ffdc50 0%, transparent 70%);
	}

	/* Only the landing card pops, and only once its reel locked */
	.is-landed .pack-cell[data-win="true"] {
		z-index: 1;
		animation:
			pack-pop 0.75s ease-out both,
			pack-sway var(--sway-duration, 3.4s) ease-in-out var(--sway-delay, 0s) infinite alternate;
	}

	@media (prefers-reduced-motion: reduce) {
		.is-landed .pack-cell[data-win="true"] {
			animation: pack-pop 0.75s ease-out both;
		}
	}

	.is-landed .pack-flash {
		animation: pack-flash 1s ease-out both;
	}

	.is-landed[data-rarity="legendary"] .pack-flash {
		animation-duration: 1.3s;
	}

	@media (width <= 1100px) {
		.pack-reels {
			--pack-scale: 0.42;
		}
	}

	@media (width <= 820px) {
		.pack-reels {
			--pack-scale: 0.32;
		}
	}

	@media (width <= 560px) {
		.pack-reels {
			--pack-scale: 0.24;
		}
	}

	@keyframes pack-sway {
		from {
			translate: 0 -0.2rem;
			rotate: -0.8deg;
		}

		to {
			translate: 0 0.2rem;
			rotate: 0.8deg;
		}
	}

	@keyframes pack-pop {
		0% {
			transform: scale(1);
		}

		45% {
			transform: scale(1.07);
		}

		100% {
			transform: scale(1);
		}
	}

	@keyframes pack-flash {
		0% {
			opacity: 0.95;
			transform: scale(0.7);
		}

		55% {
			opacity: 0.45;
		}

		100% {
			opacity: 0;
			transform: scale(1.3);
		}
	}
</style>
