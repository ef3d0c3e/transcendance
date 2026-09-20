<script>
	import { tick } from 'svelte';
	import { mount } from 'svelte';
	import { onMount } from "svelte";

	let { url, target, formSelector, enhancementSelector } = $props();

	import FormEnhancement from './FormEnhancement.svelte';


	let label = $state("");
	onMount(() => {
		if (!target) {
			return;
		}

		label = target.dataset.label ?? "";
	});

	let open = $state(false);
	let loading = $state(false);
	let loaded = $state(false);
	let positioned = $state(false);

	let trigger;
	let content;
	let popup;

	let left = $state(0);
	let top = $state(0);

	let closeTimer;

	async function openPopover() {
		clearTimeout(closeTimer);

		open = true;

		if (loaded || loading) {
			positionPopup();
			return;
		}

		loading = true;

		try {
			const response = await fetch(url + '?fragment=1');

			if (!response.ok) {
				throw new Error(`HTTP ${response.status}`);
			}

			content.innerHTML = await response.text();

			const enhancementTarget =
				content.querySelector(enhancementSelector);

			if (enhancementTarget) {
				mount(FormEnhancement, {
					target: enhancementTarget,
					props: {
						formSelector: formSelector,
						onSuccess: handleFormSuccess,
					},
				});
			}

			loaded = true;

			await tick();

			positionPopup();
			positioned = true;
		} catch (error) {
			console.error(error);
			content.textContent = 'Unable to load form.';
		} finally {
			loading = false;
		}
	}

	function closePopover() {
		clearTimeout(closeTimer);
		open = false;
	}

	function handlePointerLeave() {
		clearTimeout(closeTimer);

		closeTimer = setTimeout(() => {
			closePopover();
		}, 150);
	}

	function handlePointerEnter() {
		clearTimeout(closeTimer);
	}

	function handleFormSuccess() {
		clearTimeout(closeTimer);
		console.log('HERE');

		closeTimer = setTimeout(() => {
			console.log('Timeout');
			closePopover();
		}, 800);
	}

	function positionPopup() {
		if (!trigger || !popup) {
			return;
		}

		const triggerRect = trigger.getBoundingClientRect();
		const popupRect = popup.getBoundingClientRect();

		const margin = 8;
		const gap = 6;

		let newTop = triggerRect.bottom + gap;

		if (newTop + popupRect.height > window.innerHeight - margin) {
			newTop = triggerRect.top - popupRect.height - gap;

			newTop = Math.max(margin, newTop);
		}

		let newLeft = triggerRect.right - popupRect.width;

		newLeft = Math.max(
			margin,
			Math.min(newLeft, window.innerWidth - popupRect.width - margin),
		);

		left = Math.round(newLeft);
		top = Math.round(newTop);
	}

	function handleResize() {
		if (open) {
			positionPopup();
		}
	}

	function handleScroll() {
		if (open) {
			positionPopup();
		}
	}
</script>

<div
	style="display: inline-block"
	bind:this={trigger}
	onpointerenter={openPopover}
	onpointerleave={handlePointerLeave}
	role="presentation"
>
	<a href="{url}">
		<button type="button" aria-haspopup="dialog" aria-expanded={open}>
			{label}
		</button>
	</a>

	<div
		class="popover"
		role="dialog"
		tabindex="-1"
		class:open
		class:positioned
		bind:this={popup}
		style:left={`${left}px`}
		style:top={`${top}px`}
		onpointerenter={handlePointerEnter}
		onpointerleave={handlePointerLeave}
	>
		<section class="section-popover" style="max-height: calc(100vh - 16px)">
			{#if loading}
				<p>Loading...</p>
			{/if}

			<div bind:this={content}></div>
		</section>
	</div>
</div>

<svelte:window onresize={handleResize} onscroll={handleScroll} />

<style>
	.popover {
		position: fixed;
		z-index: 1000;

		width: min(20rem, calc(100vw - 16px));
		max-height: calc(100vh - 16px);

		visibility: hidden;
		opacity: 0;
		pointer-events: none;

		transition:
			opacity 150ms ease,
			visibility 150ms ease;
	}

	.popover.open {
		visibility: visible;
		opacity: 1;
		pointer-events: auto;
	}

	.popover:not(.positioned) {
		visibility: hidden;
	}
</style>
