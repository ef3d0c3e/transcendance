<script>
	import { onMount, tick } from "svelte";

	let {
		formSelector,
		openLabel,
		retryLabel,
		errorLabel,
		obtainedLabel,
		returnLabel,
		returnUrl,
		openAgainLabel,
		autostart = false,
	} = $props();

	let reels = $state([]);
	let showcase = $state([]);
	let message = $state("");
	let failed = $state(false);
	let colors = $state({ primary: "#407fff", secondary: "#7f2fff" });
	let busy = $state(false);
	// Card scale in the reels, fitted to the viewport once they are mounted
	let reelScale = $state(null);

	let stageElement = $state();
	let resultDialog;
	// Boosters left in the opened stack, shown on the popup's open again button
	let remaining = $state(0);
	let remainingLabel = $state("");
	let remainingBadge = $derived(remaining > 9 ? "9+" : String(remaining));
	// Form and button of the page, reused to open again from the popup
	let pageForm;
	let pageButton;
	let timers = [];

	// Slot machine: five vertical reels side by side. Each drops past its
	// window on its own clock, overshoots the payline, then springs back up
	// onto the card it landed on.
	const SPIN_MIN_MS = 5000;
	const SPIN_MAX_MS = 9000;
	const SPIN_EASE = "cubic-bezier(0.08, 0.72, 0.16, 1)";
	const BOUNCE_MS = 280;
	const BOUNCE_EASE = "cubic-bezier(0.34, 1.3, 0.64, 1)";
	// How far the reel drops past the won card before springing back, in cards.
	// Must stay under half a card, so the won card never leaves the payline.
	const OVERSHOOT_CELLS = 0.3;
	// How far the stack is pulled up before it starts its drop, in cards.
	// Matches reelBelow on the server, the cards stacked beneath the winner
	const RUN_CELLS = 24;
	// Share of the viewport the reels may use
	const REELS_MAX_WIDTH = 0.92;
	const REELS_MAX_HEIGHT = 0.7;
	const REELS_MAX_SCALE = 1;

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

	function scrollToReels() {
		const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
		stageElement.scrollIntoView({ behavior: reduced ? "auto" : "smooth", block: "center" });
	}

	// Failure reported by the server, its message is shown as is
	class PackError extends Error {}

	function showFailure(text) {
		failed = true;
		message = text;
		reels = [];
		showcase = [];
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

			return {
				slots,
				rarity: winner?.rarity ?? "common",
				landed: false,
			};
		});
	}

	// The cards actually won, held back until every reel has stopped
	function buildShowcase(stacks) {
		return stacks.map((slots) => slots.find((cell) => cell.win)).filter(Boolean);
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
		resultDialog?.close();
		// Clear the stage so the previous reels cannot flash back into view
		reels = [];
		showcase = [];
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
			remaining = result.remaining;
			remainingLabel = result.remainingLabel;
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
			showcase = buildShowcase(result.reels);
			// Updated once the cards are revealed, not to spoil them
			updatePageInfo(result);
			button.textContent = retryLabel ?? openLabel;

			// Obtained cards are shown in a popup
			await tick();
			resultDialog.showModal();
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

	// Open the next booster of the stack, the page form already points to it
	function openAgain() {
		if (remaining === 0 || !pageForm) {
			return;
		}
		resultDialog.close();
		openPack(pageForm, pageButton);
	}

	onMount(() => {
		const form = document.querySelector(formSelector);
		pageForm = form;
		pageButton = form?.querySelector("button[type=submit]");

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
			<div class="pack-reel" class:is-landed={reel.landed} data-rarity={reel.rarity}>
				<div class="pack-strip">
					{#each reel.slots as cell}
						<div class="pack-cell" data-rarity={cell.rarity} data-win={cell.win}>
							<div class="pack-cell-inner">
								{@html cell.html}
							</div>
						</div>
					{/each}
				</div>
				<div class="pack-shade" aria-hidden="true"></div>
				<div class="pack-flash" aria-hidden="true"></div>
			</div>
		{/each}
		<div class="pack-payline" aria-hidden="true"></div>
	</div>
{/if}

<dialog class="pack-result" aria-labelledby="pack-result-title" bind:this={resultDialog}>
	{#if showcase.length > 0}
	<section class="pack-showcase">
		<h3 class="pack-showcase-title" id="pack-result-title">{obtainedLabel}</h3>
		<div class="pack-showcase-row" style="--pack-count: {showcase.length}">
			{#each showcase as card, index}
				<div class="pack-showcase-card" data-rarity={card.rarity}
					style="animation-delay: {index * 90}ms">
					<div class="pack-showcase-inner">
						{@html card.html}
					</div>
				</div>
			{/each}
		</div>
		<div class="pack-result-actions">
			<button type="button" class="pack-result-again" disabled={remaining === 0}
				title={remainingLabel} onclick={openAgain}>
				{openAgainLabel}
				<span class="pack-result-count" aria-label={remainingLabel}>{remainingBadge}</span>
			</button>
			<!-- svelte-ignore a11y_autofocus -->
			<a class="pack-result-return" href={returnUrl} autofocus>{returnLabel}</a>
		</div>
	</section>
	{/if}
</dialog>

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
		overflow: hidden;
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
		pointer input and never wear their effects. Hovering and effects
		belong to the obtained cards in the showcase. */
	.pack-reel .pack-cell {
		pointer-events: none;
	}

	.pack-reel .pack-cell :global(.card-effect) {
		display: none;
	}

	/* Cylinder shading so the card curves away at the edges of the window */
	.pack-shade {
		position: absolute;
		inset: 0;
		z-index: 2;
		pointer-events: none;
		border-radius: inherit;
		background:
			linear-gradient(180deg,
				rgb(0 0 0 / 45%) 0%,
				transparent 20%,
				transparent 80%,
				rgb(0 0 0 / 45%) 100%),
			linear-gradient(90deg,
				rgb(0 0 0 / 35%) 0%,
				transparent 16%,
				transparent 84%,
				rgb(0 0 0 / 35%) 100%);
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
		animation: pack-pop 0.75s ease-out both;
	}

	.is-landed .pack-flash {
		animation: pack-flash 1s ease-out both;
	}

	.is-landed[data-rarity="legendary"] .pack-flash {
		animation-duration: 1.3s;
	}

	/* The prize: the five obtained cards, bigger and fully interactive */
	/* Popup with the obtained cards */
	/* The dialog is the panel, it scrolls when the cards don't fit */
	.pack-result {
		width: min(100% - 2rem, 90rem);
		max-width: none;
		max-height: min(100dvh - 2rem, 60rem);
		padding: 0;
		overflow: auto;
		overscroll-behavior: contain;

		color: var(--color-text);
		background: var(--color-bg-2);
		border: 1px solid var(--color-border-1);
		border-radius: 1rem;
		box-shadow: 0 10px 30px var(--color-shadow);
	}

	/* Without the global section panel, the dialog already draws it */
	.pack-result .pack-showcase {
		padding: 1.5rem 1rem 0;

		background: none;
		border: 0;
		border-radius: 0;
		box-shadow: none;
	}

	/* Small or zoomed in screens: the popup takes the whole screen */
	@media (max-width: 640px), (max-height: 600px) {
		.pack-result {
			inset: 0;
			width: auto;
			height: auto;
			max-height: none;
			margin: 0;

			border: 0;
			border-radius: 0;
		}
	}

	.pack-result::backdrop {
		background: rgb(0 0 0 / 55%);
		backdrop-filter: blur(3px);
	}

	/* The page behind the popup doesn't scroll */
	:global(html:has(.pack-result[open])) {
		overflow: hidden;
	}

	/* Stays visible at the bottom while the cards scroll */
	.pack-result-actions {
		position: sticky;
		bottom: 0;

		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 0.75rem;

		margin-top: 0.5rem;
		padding: 1rem 0 max(1rem, env(safe-area-inset-bottom));

		background: var(--color-bg-2);
	}

	/* Looks like the page's open button */
	.pack-result-again {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.6rem;

		min-height: 2.5rem;
		padding: 0.45rem 0.75rem 0.45rem 1.25rem;

		color: var(--color-text-alt);
		background: var(--color-accent-primary);

		&:hover:not(:disabled) {
			background: var(--color-accent-primary-dark);
		}

		&:disabled {
			color: var(--color-text-muted);
			background: var(--color-bg-4);
			opacity: 1;
		}
	}

	.pack-result-count {
		min-width: 1.6rem;
		padding: 0.1rem 0.45rem;

		background: rgb(255 255 255 / 22%);
		border-radius: 999px;

		font-size: 0.8rem;
		font-weight: 700;
		font-variant-numeric: tabular-nums;
		text-align: center;
	}

	.pack-result-again:disabled .pack-result-count {
		background: var(--color-bg-2);
	}

	/* Looks like the generic buttons */
	.pack-result-return {
		display: inline-flex;
		align-items: center;
		justify-content: center;

		min-height: 2.5rem;
		min-width: 8rem;
		padding: 0.45rem 1.25rem;

		color: var(--color-text-alt);
		background: var(--color-accent-secondary);
		border-radius: 0.55rem;

		font-weight: 600;
		line-height: 1.3;
		text-decoration: none;

		transition: background-color 120ms ease;
	}

	.pack-result-return:hover {
		color: var(--color-text-alt);
		background: var(--color-accent-secondary-dark);
	}

	.pack-result-return:focus-visible {
		outline: 3px solid color-mix(in srgb, var(--color-accent-secondary) 55%, transparent);
		outline-offset: 2px;
	}

	.pack-showcase {
		--showcase-scale: 0.8;
		--showcase-cell: calc(30ch * var(--showcase-scale));
		--showcase-cell-height: calc(30em * var(--showcase-scale));
		--showcase-gap: 1.15rem;

		margin: 0 auto;
	}

	.pack-showcase-title {
		margin: 0 0 1rem;
		text-align: center;
	}

	.pack-showcase-row {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: var(--showcase-gap);
		width: calc(var(--pack-count) * var(--showcase-cell) + (var(--pack-count) - 1) * var(--showcase-gap));
		max-width: 100%;
		margin: 0 auto;
	}

	.pack-showcase-card {
		width: var(--showcase-cell);
		height: var(--showcase-cell-height);
		animation: showcase-in 0.55s cubic-bezier(0.34, 1.4, 0.64, 1) both;
	}

	.pack-showcase-inner {
		width: 30ch;
		height: 30em;
		transform: scale(var(--showcase-scale));
		transform-origin: top left;
	}

	@media (width <= 1100px) {
		.pack-reels {
			--pack-scale: 0.42;
		}

		.pack-showcase {
			--showcase-scale: 0.68;
		}
	}

	@media (width <= 820px) {
		.pack-reels {
			--pack-scale: 0.32;
		}

		.pack-showcase {
			--showcase-scale: 0.54;
		}
	}

	@media (width <= 560px) {
		.pack-reels {
			--pack-scale: 0.24;
		}

		.pack-showcase {
			--showcase-scale: 0.42;
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

	@keyframes showcase-in {
		0% {
			opacity: 0;
			transform: scale(0.7);
		}

		100% {
			opacity: 1;
			transform: scale(1);
		}
	}
</style>
