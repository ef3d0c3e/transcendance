<script>
	import { onMount, tick } from "svelte";
	import { openTopup } from "./topup.js";
	import { loadMuted, packSounds, saveMuted, soundsAllowed, unlockSounds } from "./pack-sounds.js";

	let {
		formSelector,
		openLabel,
		retryLabel,
		errorLabel,
		againLabel,
		coins: initialCoins = 0,
		autostart = false,
		// Labels of the sound switch, while the sound is on and off
		soundOnLabel = "Sound on",
		soundOffLabel = "Sound off",
	} = $props();

	let muted = $state(loadMuted());
	let sounds = $derived(packSounds(muted));
	// How far each reel rolled past the payline, in ticks: a tick plays when
	// it changes
	let paylineTicks = [];

	let reels = $state([]);
	let message = $state("");
	let failed = $state(false);
	let colors = $state({ primary: "#407fff", secondary: "#7f2fff" });
	let busy = $state(false);
	// The reels are turning: the payline blinks faster
	let spinning = $state(false);
	// Every reel landed: the payline flashes
	let settled = $state(false);
	// Card scale in the reels, fitted to the viewport once they are mounted
	let reelScale = $state(null);

	let stageElement = $state();
	let againButton = $state();
	// Height of the previous and next cards peeking around the won one, in px
	let reelPeek = $state(0);
	// What the stage's button does once the reels stopped: "open" the next
	// booster of the stack, or "buy" one. null while spinning.
	let stageAction = $state(null);
	// Price shown on the stage's button when it buys, the page's buy button markup
	let priceHtml = $state("");
	// Rare and legendary landings flash the whole stage
	let stageFlash = $state({ key: 0, rarity: "" });
	// The page's form, with its open button for boosters left in the stack,
	// and its buy button once they are all opened
	let pageForm;
	let openButton;
	let buyButton;
	// Balance, kept up to date by every purchase
	let coins = initialCoins;
	let timers = [];

	// Slot machine: five vertical reels side by side. Each drops past its
	// window on its own clock, overshoots the payline, then springs back up
	// onto the card it landed on.
	const SPIN_MIN_MS = 6000;
	const SPIN_MAX_MS = 7000;
	// Starts at twice the average speed and slows down linearly to a stop on the
	// payline: the exact quadratic ease-out
	const SPIN_EASE = "cubic-bezier(0.333, 0.667, 0.667, 1)";
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
	// Share of the screen under the header for the won cards, the rest is for
	// the cards peeking around them and the button under the reels
	const REELS_MAX_HEIGHT = 0.6;
	// Room kept around the reels for the button under them and the stage's
	// spacing, in px
	const ACTIONS_HEIGHT = 190;
	// The peeking cards show at most this share of themselves
	const PEEK_MAX = 0.6;
	const REELS_MAX_SCALE = 1;
	// Buys a booster of a collection and opens it at once
	const BUY_URL = "/opening/buy";
	// Time for the owned cards count to roll up to its new value
	const COUNT_UP_MS = 700;
	// Cards fade the further they are from the middle of their reel: fully
	// opaque within FADE_FULL of a card from it, then opacity drops by
	// FADE_RATE per card of distance, down to FADE_MIN
	const FADE_FULL = 0.2;
	const FADE_RATE = 0.7;
	const FADE_MIN = 0.12;
	// The cards keep being followed this long after the last reel locked, its
	// bounce included
	const FADE_SETTLE_MS = 500;
	// Ticks the rolling sound plays for each card passing the payline
	const TICKS_PER_CARD = 3;
	// Period of the reels' sway, picked per reel so they never move in sync
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
		const row = stageElement.querySelector(".pack-reels");
		const gap = Number.parseFloat(getComputedStyle(row).columnGap) || 0;
		const count = reels.length;

		const width = Math.min(window.innerWidth * REELS_MAX_WIDTH, window.innerWidth - 32);
		const header = document.querySelector("body > header")?.offsetHeight ?? 0;
		const height = window.innerHeight - header - ACTIONS_HEIGHT;
		const byWidth = (width - (count - 1) * gap) / count / cardWidth;
		const byHeight = (height * REELS_MAX_HEIGHT) / cardHeight;

		reelScale = Math.min(byWidth, byHeight, REELS_MAX_SCALE);
		// The rest of the height shows the cards before and after the won one
		const scaled = cardHeight * reelScale;
		reelPeek = Math.floor(Math.max(0, Math.min(scaled * PEEK_MAX, (height - scaled) / 2)));
	}

	// The page around the reels: the booster and its information panel
	function packPage() {
		return document.querySelector(".pack-page");
	}

	// While opening, the reels have the screen to themselves
	function hidePage() {
		const page = packPage();
		if (page && !page.classList.contains("is-opening")) {
			page.classList.add("is-opening");
			page.classList.remove("is-revealed");
			window.scrollTo({ top: 0, behavior: "instant" });
		}
	}

	// Once opened, the booster and its panel come back above the reels. They
	// push the reels down, the page scrolls along so the cards stay in place.
	function revealPage() {
		const page = packPage();
		if (!page?.classList.contains("is-opening")) {
			return;
		}
		// The reels when shown, else the stage holding the message. Measured
		// before the header and footer come back too.
		const anchor = stageElement?.querySelector(".pack-reels") ?? stageElement;
		const before = anchor?.getBoundingClientRect().top;
		page.classList.remove("is-opening");
		page.classList.add("is-revealed");
		if (before !== undefined) {
			window.scrollBy({ top: anchor.getBoundingClientRect().top - before, behavior: "instant" });
		}
	}

	// The stage's button: same as the page's open or buy button
	function openAgain() {
		if (stageAction === "open") {
			openPack(false, true);
		} else if (stageAction === "buy") {
			buyOrTopUp();
		}
	}

	// Failure reported by the server, its message is shown as is
	class PackError extends Error {}

	function showFailure(text) {
		failed = true;
		message = text;
		reels = [];
		revealPage();
	}

	function clearTimers() {
		for (const timer of timers) {
			clearTimeout(timer);
		}
		timers = [];
		stopFading();
	}

	// Where the cards sit in their reel, measured once per opening: while
	// spinning, they only move with their strip's transform
	let fadeGeometry = null;

	function measureFade() {
		return [...stageElement.querySelectorAll(".pack-window")].map((reelWindow) => {
			const strip = reelWindow.querySelector(".pack-strip");
			const cells = [...strip.querySelectorAll(".pack-cell")].map((element) => ({
				element,
				// Offset in the window, transforms ignored
				top: element.offsetTop,
				opacity: null,
			}));
			return {
				strip,
				cells,
				height: reelWindow.clientHeight,
				cellHeight: cells[0]?.element.offsetHeight ?? 1,
				pitch: cells.length > 1 ? cells[1].top - cells[0].top : 0,
			};
		});
	}

	// Set each card's opacity from its distance to the middle of its reel.
	// Only the strip's transform is read per frame, and only the opacities that
	// changed are written: every write repaints the reel.
	function fadeCells() {
		fadeGeometry ??= measureFade();
		fadeGeometry.forEach((reel, index) => {
			const offset = new DOMMatrixReadOnly(getComputedStyle(reel.strip).transform).m42;
			const middle = reel.height / 2;
			for (const cell of reel.cells) {
				const top = cell.top + offset;
				// Out of the window, it is cut anyway
				let opacity = 0;
				if (top + reel.cellHeight >= 0 && top <= reel.height) {
					const distance = Math.abs(top + reel.cellHeight / 2 - middle) / reel.cellHeight;
					opacity = Math.max(FADE_MIN, 1 - Math.max(0, distance - FADE_FULL) * FADE_RATE);
				}
				const value = opacity.toFixed(2);
				if (value !== cell.opacity) {
					cell.opacity = value;
					cell.element.style.opacity = value;
					// Hidden cards are skipped by the painting altogether
					cell.element.style.visibility = opacity === 0 ? "hidden" : "";
				}
			}
			// The rolling reel clicks a few times per card, until it landed
			if (reel.pitch > 0) {
				const ticks = Math.floor(((middle - reel.cells[0].top - offset) / reel.pitch) * TICKS_PER_CARD);
				if (paylineTicks[index] !== undefined && ticks !== paylineTicks[index] && spinning && !reels[index]?.landed) {
					sounds.tick(index);
				}
				paylineTicks[index] = ticks;
			}
		});
	}

	let fadeFrame = 0;

	// Follow the cards while the reels turn
	function startFading() {
		cancelAnimationFrame(fadeFrame);
		const step = () => {
			fadeCells();
			fadeFrame = requestAnimationFrame(step);
		};
		step();
	}

	// The reels stopped: a last pass for where the cards rest
	function stopFading() {
		if (!fadeFrame) {
			return;
		}
		cancelAnimationFrame(fadeFrame);
		fadeFrame = 0;
		fadeCells();
	}

	function delay(ms) {
		return new Promise((resolve) => {
			setTimeout(resolve, ms);
		});
	}

	// Opens the booster of the form, or buys one of the collection with coins
	async function requestPack(buy) {
		const response = buy
			? await fetch(BUY_URL, {
				method: "POST",
				body: new URLSearchParams({ collection: buyButton.dataset.collection }),
			})
			: await fetch(pageForm.action, {
				method: pageForm.method,
				body: new FormData(pageForm),
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
			coins: data.coins,
			coinsLabel: data.coinsLabel ?? "",
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

	// The won card lands in the middle of the window, under the peeking one
	function restOffset(element, winnerIndex) {
		return reelPeek - winnerIndex * cellPitch(element);
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

			timers.push(setTimeout(async () => {
				reels[index].landed = true;
				sounds.land(reels[index].rarity);
				// Good pulls light up the whole stage
				if (reels[index].rarity !== "common") {
					stageFlash = { key: stageFlash.key + 1, rarity: reels[index].rarity };
				}
			}, duration + BOUNCE_MS));

			return duration;
		});
	}

	// Disable both buttons while a pack is opening
	function setButtonsDisabled(disabled) {
		for (const button of [openButton, buyButton]) {
			if (button) {
				button.disabled = disabled;
			}
		}
	}

	// Open the next booster of the stack, or buy one with coins when buy is set.
	// cue plays the start sound, for openings the user clicked.
	async function openPack(buy = false, cue = false) {
		if (busy) {
			return;
		}

		busy = true;
		if (cue) {
			sounds.start();
		}
		clearTimers();
		paylineTicks = [];
		fadeGeometry = null;
		// Clear the stage so the previous reels cannot flash back into view
		reels = [];
		failed = false;
		message = "";
		stageAction = null;
		hidePage();
		setButtonsDisabled(true);
		openButton?.classList.remove("is-ready");
		buyButton?.classList.remove("is-ready");

		try {
			const result = await requestPack(buy);

			if (result.reels.length === 0) {
				showFailure(errorLabel);
				return;
			}

			// The booster is consumed: chain into the next one of the stack, if
			// any, else the next ones are bought
			const input = pageForm.querySelector("input[name=booster]");
			const chained = Boolean(result.next && input);
			if (chained) {
				input.value = result.next;
			}
			if (typeof result.coins === "number") {
				coins = result.coins;
			}

			reels = buildReels(result.reels);
			colors = result.palette;

			// Wait for the stacks to mount, size them, then measure them
			await tick();
			fitReels();
			await tick();
			settled = false;
			spinning = true;
			const durations = spinReels(result.reels);
			startFading();

			await delay(Math.max(...durations) + BOUNCE_MS + 80);
			timers.push(setTimeout(stopFading, FADE_SETTLE_MS));
			spinning = false;
			settled = true;
			// Updated once the cards are revealed, not to spoil them
			updatePageInfo(result);
			revealPage();
			// Its light stays, only its label changes
			openButton.querySelector(".pack-open-label").textContent = retryLabel ?? openLabel;
			openButton.hidden = !chained;
			buyButton.hidden = chained;
			// Calls for the next opening, until it is clicked
			const next = chained ? openButton : buyButton;
			next.classList.add("is-ready");

			// The won cards stay on the stage, its button opens the next one
			priceHtml = chained ? "" : (buyButton.querySelector(".pack-buy-price")?.outerHTML ?? "");
			stageAction = chained ? "open" : "buy";
			await tick();
			againButton?.focus();
		} catch (error) {
			showFailure(error instanceof PackError ? error.message : errorLabel);
		} finally {
			busy = false;
			setButtonsDisabled(false);
		}
	}

	// Open right away when the browser lets the sounds play. Otherwise they
	// would miss the spin: the page shows, its button calling for the click.
	async function openOnArrival(open) {
		if (open && (muted || (await soundsAllowed()))) {
			openPack();
			return;
		}
		revealPage();
		if (open) {
			openButton?.classList.add("is-ready");
			openButton?.focus();
		}
	}

	// Buy the next booster when the balance covers it, else offer coins
	function buyOrTopUp() {
		const price = Number(buyButton.dataset.price);
		if (coins >= price) {
			openPack(true, true);
			return;
		}
		// Credited without leaving: the clicked pack goes on with the purchase
		openTopup(price - coins, (result) => {
			coins = result.coins;
			const balance = document.querySelector("#pack-coins");
			if (balance && result.coinsLabel) {
				balance.textContent = result.coinsLabel;
			}
			buyOrTopUp();
		});
	}

	// Roll a number up to its new value, then make it pop
	function countUp(element, to) {
		const from = Number(element.textContent) || 0;
		const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
		if (to === from || reduced) {
			element.textContent = String(to);
			return;
		}

		const start = performance.now();
		function step(now) {
			const progress = Math.min(1, (now - start) / COUNT_UP_MS);
			element.textContent = String(Math.round(from + (to - from) * progress));
			if (progress < 1) {
				requestAnimationFrame(step);
			}
		}
		requestAnimationFrame(step);

		// Restart the pop even when it just ran
		element.classList.remove("is-updated");
		void element.offsetWidth;
		element.classList.add("is-updated");
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
		const balance = document.querySelector("#pack-coins");
		if (balance && result.coinsLabel) {
			balance.textContent = result.coinsLabel;
		}

		if (owned && typeof result.owned === "number") {
			countUp(owned, result.owned);
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
		pageForm = form;
		openButton = form.querySelector("button[type=submit]");
		buyButton = form.querySelector(".pack-buy");

		function handleSubmit(event) {
			event.preventDefault();
			openPack(false, true);
		}

		form.addEventListener("submit", handleSubmit);
		unlockSounds();
		buyButton?.addEventListener("click", buyOrTopUp);

		// Coming from the inventory's open button: open the owned booster
		openOnArrival(autostart);

		return () => {
			form.removeEventListener("submit", handleSubmit);
			buyButton?.removeEventListener("click", buyOrTopUp);
			clearTimers();
		};
	});
</script>

<div class="pack-stage" class:is-active={busy || reels.length > 0} aria-label={againLabel} bind:this={stageElement}>
	{#key stageFlash.key}
		{#if stageFlash.rarity}
			<div class="pack-stage-flash" data-rarity={stageFlash.rarity} aria-hidden="true"></div>
		{/if}
	{/key}

	{#if message}
		<section class={failed ? "section-error" : "section-success"}>{message}</section>
	{/if}

	{#if reels.length > 0}
		<div class="pack-reels" class:is-spinning={spinning} class:is-settled={settled}
			style="--c1: {colors.primary}; --c2: {colors.secondary}; --pack-count: {reels.length}; --pack-peek: {reelPeek}px; {reelScale ? `--pack-scale: ${reelScale}` : ''}">
			{#each reels as reel}
				<div class="pack-reel" class:is-landed={reel.landed} data-rarity={reel.rarity} style={reel.sway}>
					<div class="pack-rays" aria-hidden="true"></div>
					<div class="pack-window">
						<div class="pack-strip">
							{#each reel.slots as cell}
								<div class="pack-cell" data-rarity={cell.rarity} data-win={cell.win}>
									<div class="pack-cell-inner">
										{@html cell.html}
									</div>
								</div>
							{/each}
						</div>
					</div>
					<div class="pack-flash" aria-hidden="true"></div>
					<div class="pack-shine" aria-hidden="true"></div>
				</div>
			{/each}
			<div class="pack-payline" aria-hidden="true"></div>
		</div>
	{/if}

	<button type="button" class="pack-sound" aria-pressed={!muted}
		onclick={() => { muted = !muted; saveMuted(muted); }}>
		{muted ? soundOffLabel : soundOnLabel}
	</button>

	<div class="pack-stage-actions">
		{#if stageAction}
			<button type="button" class="pack-again" bind:this={againButton} onclick={openAgain}>
				<span class="pack-again-shine" aria-hidden="true"></span>
				{againLabel}
				{#if priceHtml}{@html priceHtml}{/if}
			</button>
		{/if}
	</div>
</div>

<style>
	/* Sound switch, in a corner where it stays while the cards spin */
	.pack-sound {
		position: fixed;
		right: 1rem;
		bottom: 1rem;
		z-index: 30;
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.35rem 0.6rem;
		border-radius: 999px;
		background: color-mix(in srgb, var(--color-bg-2) 85%, transparent);
		color: var(--color-text);
		border: 1px solid color-mix(in srgb, var(--color-text) 20%, transparent);
		backdrop-filter: blur(6px);
		font-size: 0.85rem;
		cursor: pointer;
	}

	.pack-sound[aria-pressed="false"] {
		opacity: 0.6;
	}
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
		flex: none;
	}

	/* The reels and the button under them. While opening, the stage takes
		the screen under the header, the reels in its middle. */
	.pack-stage {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 2rem;
	}

	.pack-stage.is-active {
		min-height: calc(100dvh - 5rem);
	}

	/* Keeps its height while spinning, so the reels don't move when the button shows */
	.pack-stage-actions {
		display: flex;
		justify-content: center;
		min-height: 4.5rem;
	}

	/* The next opening, large and calling for a click. Its ring and light
		only move and fade, the browser doesn't repaint them. */
	.pack-again {
		position: relative;
		/* The ring and the light stay between the background and the label */
		isolation: isolate;
		display: inline-flex;
		align-items: center;
		gap: 0.9rem;

		min-height: 4.5rem;
		padding: 0.9rem 3rem;

		color: var(--color-text-alt);
		background: var(--color-accent-primary);
		border: 0;
		border-radius: 1rem;
		box-shadow: 0 10px 30px rgb(0 0 0 / 40%);

		font-size: 1.6rem;
		font-weight: 800;

		animation: pack-again-in 0.45s cubic-bezier(0.34, 1.5, 0.64, 1) both;

		/* A ring bursting out of the button */
		&::before {
			content: "";
			position: absolute;
			inset: 0;
			z-index: -1;
			pointer-events: none;
			opacity: 0;

			border-radius: inherit;
			box-shadow: 0 0 0 0.4rem color-mix(in srgb, var(--color-accent-primary) 75%, transparent);
			animation: pack-again-ring 1.6s ease-out 0.45s infinite;
		}

		&:hover {
			background: var(--color-accent-primary-dark);

			.pack-again-shine {
				display: none;
			}
		}

		:global(.pack-buy-price) {
			font-size: 1.15rem;
		}
	}

	/* A light sweeping across the button, cut to its shape */
	.pack-again-shine {
		position: absolute;
		inset: 0;
		z-index: -1;
		overflow: hidden;
		pointer-events: none;
		border-radius: inherit;

		&::before {
			content: "";
			position: absolute;
			inset: 0 auto 0 0;
			width: 250%;

			background: linear-gradient(110deg,
					transparent 30%,
					var(--color-accent-primary-lighter) 50%,
					transparent 70%);
			animation: pack-again-shine 2.4s linear infinite;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.pack-again {
			animation: none;

			&::before {
				display: none;
			}
		}

		.pack-again-shine {
			display: none;
		}
	}

	/* Rare and legendary landings wash the stage in their color */
	.pack-stage-flash {
		position: fixed;
		inset: 0;
		z-index: 0;
		pointer-events: none;
		opacity: 0;
		animation: pack-stage-flash 1.1s ease-out both;
	}

	.pack-stage-flash[data-rarity="rare"] {
		background: radial-gradient(circle, rgb(255 20 64 / 45%), transparent 70%);
	}

	.pack-stage-flash[data-rarity="legendary"] {
		background: radial-gradient(circle, rgb(255 236 150 / 85%), rgb(255 220 80 / 35%) 45%, transparent 80%);
		animation-duration: 1.6s;
	}

	@media (prefers-reduced-motion: reduce) {
		.pack-stage-flash {
			display: none;
		}
	}

	/* Each column is a reel: the won card in the middle, the cards before and
		after it peeking above and below. --rarity is the color of the card it
		lands on. */
	.pack-reel {
		--rarity: #407fff;

		position: relative;
		flex: 0 0 auto;
		width: var(--pack-cell);
		height: calc(var(--pack-cell-height) + 2 * var(--pack-peek, 0px));
	}

	/* The stack turns behind the window. Cut at its top and bottom, but not
		on its sides where the won card tilts. The peeking cards fade out. */
	.pack-window {
		position: relative;
		/* Above the rays of every reel, not only its own */
		z-index: 1;
		height: 100%;
		overflow-x: visible;
		overflow-y: clip;

		/* Soft edges, the cards' own opacity follows their distance to the
			middle (fadeCells) */
		mask-image: linear-gradient(to bottom,
				transparent,
				#000 calc(var(--pack-peek) * 0.6),
				#000 calc(100% - var(--pack-peek) * 0.6),
				transparent);
		mask-size: calc(100% + 2 * var(--pack-gap)) 100%;
		mask-position: center;
		mask-repeat: no-repeat;
		mask-clip: no-clip;
	}

	.pack-strip {
		display: flex;
		flex-direction: column;
		gap: var(--pack-gap);
		transform: translateY(0);
		will-change: transform;
		/* Only the won card may take the pointer */
		pointer-events: none;
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

	/* Once locked, the cards around the won one gray out */
	.pack-cell[data-win="false"] {
		transition: filter 400ms ease;
	}

	.pack-reel.is-landed .pack-cell[data-win="false"] {
		filter: grayscale(0.85);
	}

	.pack-reel:not(.is-landed) .pack-cell :global(.card-effect),
	.pack-reel .pack-cell[data-win="false"] :global(.card-effect) {
		display: none;
	}

	/* The cards sway with their window, on their reel's clock. A sway on each
		card would be repainted every frame in Firefox, one per window runs on
		the compositor. */
	.pack-window {
		animation: pack-sway var(--sway-duration, 3.4s) ease-in-out var(--sway-delay, 0s) infinite alternate;
	}

	@media (prefers-reduced-motion: reduce) {
		.pack-window {
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
		animation: pack-payline-blink 1.6s ease-in-out infinite;
	}

	/* Scanning while the reels turn */
	.is-spinning .pack-payline {
		animation-duration: 0.45s;
	}

	/* Every reel locked: two bright flashes, then it fades out until the next spin */
	.is-settled .pack-payline {
		animation:
			pack-payline-flash 0.3s ease-out 2,
			pack-payline-out 0.8s ease-in 0.6s forwards;
	}

	@media (prefers-reduced-motion: reduce) {
		.pack-payline,
		.is-spinning .pack-payline,
		.is-settled .pack-payline {
			animation: none;
		}

		.is-settled .pack-payline {
			opacity: 0;
		}
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

	.pack-reel[data-rarity="rare"] {
		--rarity: #ff1440;
	}

	.pack-reel[data-rarity="legendary"] {
		--rarity: #ffdc50;
	}

	/* Landing: a ring in the rarity's color bursts out of the card, twice
		for good pulls */
	.pack-flash,
	.pack-flash::after {
		position: absolute;
		inset: var(--pack-peek, 0px) 0;
		z-index: 3;
		pointer-events: none;
		opacity: 0;

		border: 3px solid var(--rarity);
		border-radius: 0.75rem;
		box-shadow:
			0 0 18px var(--rarity),
			inset 0 0 18px var(--rarity);
	}

	.pack-flash::after {
		content: "";
		inset: -3px;
	}

	.is-landed .pack-flash {
		animation: pack-ring 0.8s ease-out both;
	}

	.is-landed:is([data-rarity="rare"], [data-rarity="legendary"]) .pack-flash::after {
		animation: pack-ring 0.9s ease-out 0.25s both;
	}

	/* Landing: a light sweeps across the card */
	.pack-shine {
		position: absolute;
		inset: var(--pack-peek, 0px) 0;
		z-index: 3;
		overflow: hidden;
		pointer-events: none;
		border-radius: 0.5rem;
	}

	.pack-shine::before {
		content: "";
		position: absolute;
		top: -50%;
		bottom: -50%;
		left: -60%;
		width: 40%;
		opacity: 0;
		transform: rotate(20deg);
		background: linear-gradient(90deg, transparent, rgb(255 255 255 / 80%), transparent);
	}

	.is-landed .pack-shine::before {
		animation: pack-shine 0.7s ease-out 0.05s both;
	}

	/* Good pulls: light rays turning behind the cards, all of them: they are
		under every window (z-index 1), whichever reel they belong to */
	.pack-rays {
		--rays-opacity: 0.55;

		position: absolute;
		left: 50%;
		top: 50%;
		z-index: 0;
		width: 260%;
		aspect-ratio: 1;
		translate: -50% -50%;
		pointer-events: none;
		opacity: 0;

		background: repeating-conic-gradient(
			color-mix(in srgb, var(--rarity) 75%, transparent) 0deg 7deg,
			transparent 7deg 18deg);
		mask-image: radial-gradient(circle, #000 0%, transparent 62%);
		border-radius: 50%;
	}

	.pack-reel[data-rarity="legendary"] .pack-rays {
		--rays-opacity: 0.95;
	}

	.is-landed:is([data-rarity="rare"], [data-rarity="legendary"]) .pack-rays {
		animation:
			pack-rays-in 0.6s ease-out both,
			pack-rays-spin 14s linear infinite;
	}

	@media (prefers-reduced-motion: reduce) {
		.is-landed .pack-flash,
		.is-landed .pack-flash::after,
		.is-landed .pack-shine::before {
			animation: none;
		}

		.is-landed:is([data-rarity="rare"], [data-rarity="legendary"]) .pack-rays {
			animation: none;
			opacity: var(--rays-opacity);
		}
	}

	/* Only the landing card pops, and only once its reel locked */
	.is-landed .pack-cell[data-win="true"] {
		z-index: 1;
		animation: pack-pop 0.75s ease-out both;
	}

	/* Rare and legendary cards keep glowing in their rarity's color: a glow
		behind the card that only fades, the browser doesn't repaint it */
	.is-landed:is([data-rarity="rare"], [data-rarity="legendary"]) .pack-cell[data-win="true"]::before {
		content: "";
		position: absolute;
		inset: 0;
		z-index: -1;
		pointer-events: none;
		opacity: 0.25;

		border-radius: calc(0.5rem * var(--pack-scale));
		box-shadow: 0 0 16px var(--glow);
		animation: pack-glow 1.4s ease-in-out infinite alternate;
	}

	.is-landed[data-rarity="rare"] .pack-cell[data-win="true"] {
		--glow: #ff1440;
	}

	.is-landed[data-rarity="legendary"] .pack-cell[data-win="true"] {
		--glow: #ffdc50;
	}

	@media (prefers-reduced-motion: reduce) {
		.is-landed .pack-cell[data-win="true"] {
			animation: pack-pop 0.75s ease-out both;
		}

		.is-landed .pack-cell[data-win="true"]::before {
			display: none;
		}
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

	@keyframes pack-ring {
		0% {
			opacity: 1;
			transform: scale(0.96);
		}

		100% {
			opacity: 0;
			transform: scale(1.35);
		}
	}

	/* Across the card, in widths of the light: from -60% to 130% of the card */
	@keyframes pack-shine {
		0% {
			translate: 0 0;
			opacity: 1;
		}

		100% {
			translate: 475% 0;
			opacity: 1;
		}
	}

	@keyframes pack-rays-in {
		from {
			opacity: 0;
			scale: 0.4;
		}

		to {
			opacity: var(--rays-opacity);
			scale: 1;
		}
	}

	@keyframes pack-rays-spin {
		to {
			rotate: 360deg;
		}
	}

	@keyframes pack-stage-flash {
		0% {
			opacity: 0;
		}

		15% {
			opacity: 1;
		}

		100% {
			opacity: 0;
		}
	}

	@keyframes pack-again-in {
		from {
			opacity: 0;
			transform: translateY(1rem) scale(0.85);
		}
	}

	@keyframes pack-again-ring {
		from {
			opacity: 1;
			scale: 1;
		}

		to {
			opacity: 0;
			scale: 1.1 1.5;
		}
	}

	/* The band crosses from before the left edge to past the right one */
	@keyframes pack-again-shine {
		from {
			translate: -60% 0;
		}

		to {
			translate: 0 0;
		}
	}

	@keyframes pack-payline-out {
		to {
			opacity: 0;
		}
	}

	@keyframes pack-payline-blink {
		50% {
			opacity: 0.35;
		}
	}

	@keyframes pack-payline-flash {
		50% {
			opacity: 1;
			box-shadow:
				0 0 22px rgb(255 235 160),
				0 0 60px rgb(255 215 106 / 80%);
		}
	}

	@keyframes pack-glow {
		to {
			opacity: 1;
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

</style>
