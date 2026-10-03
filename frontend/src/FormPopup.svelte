<script>
	import { onMount, tick, mount } from 'svelte';
	import FormEnhancement from './FormEnhancement.svelte';

	let { url, target, formSelector, enhancementSelector } = $props();

	let label = $state('');
	let open = $state(false);
	let loading = $state(false);
	let loaded = $state(false);

	let content;
	let popup;

	let closeTimer;

	onMount(() => {
		if (target) {
			label = target.dataset.label ?? '';
		}
	});

	async function openPopover() {
		clearTimeout(closeTimer);
		open = true;

		if (loaded || loading) {
			await tick();
			popup?.focus();
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
						formSelector,
						onSuccess: handleFormSuccess
					}
				});
			}

			loaded = true;

			await tick();
			popup?.focus();
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

	function handleFormSuccess() {
		clearTimeout(closeTimer);

		closeTimer = setTimeout(() => {
			closePopover();
		}, 800);
	}

	function handleBackdropClick(event) {
		if (event.target === event.currentTarget) {
			closePopover();
		}
	}

	function handleKeydown(event) {
		if (event.key === 'Escape' && open) {
			event.preventDefault();
			closePopover();
		}
	}
</script>

<button
	type="button"
	aria-haspopup="dialog"
	aria-expanded={open}
	onclick={open ? closePopover : openPopover}
>
	{label}
</button>

<div
	class="backdrop"
	class:open
	role="presentation"
	aria-hidden={!open}
	onclick={handleBackdropClick}
>
	<div
		class="popover"
		role="dialog"
		aria-modal="true"
		aria-label={label}
		tabindex="-1"
		bind:this={popup}
	>
		<section class="section-popover">
			{#if loading}
				<p>Loading...</p>
			{/if}

			<div bind:this={content}></div>
		</section>
	</div>
</div>

<svelte:window onkeydown={handleKeydown} />

<style>
	.backdrop {
		position: fixed;
		inset: 0;
		z-index: 1000;

		display: flex;
		align-items: center;
		justify-content: center;

		padding: 16px;

		background: rgb(0 0 0 / 50%);

		visibility: hidden;
		opacity: 0;
		pointer-events: none;

		transition:
			opacity 150ms ease,
			visibility 150ms ease;
	}

	.backdrop.open {
		visibility: visible;
		opacity: 1;
		pointer-events: auto;
	}

	.popover {
		width: min(20rem, calc(100vw - 32px));
		max-height: calc(100vh - 32px);

		overflow: auto;

		border-radius: 8px;
		box-shadow: 0px 0px 50px 10px rgb(0 0 0 / 25%);
	}

	.section-popover {
		max-height: calc(100vh - 32px);
		overflow: auto;
		padding: 1rem;
	}
</style>
