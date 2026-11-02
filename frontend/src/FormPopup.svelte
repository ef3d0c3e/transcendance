<!--
    Login or register form in a modal <dialog>, opened from the header. The
    form is the server's page fragment (`url` with `?fragment=1`), loaded on
    the first opening and submitted by FormEnhancement. `kind` ("login" or
    "register") prefixes the ids of the fragment: `#<kind>-form`,
    `#<kind>-enhancement`.

    A successful login reloads the page, whose header then shows the user.
    A successful registration keeps its message on screen until the popup
    is closed; the next opening loads a fresh form.
-->

<script>
	import { mount, tick, unmount } from "svelte";
	import FormEnhancement from "./FormEnhancement.svelte";
	import { apiFetch } from "./api.js";

	// `labelLoading`, `labelError` and `labelClose` are the popup's own texts
	let { url, kind, label, labelLoading, labelError, labelClose } = $props();

	let dialog;
	let trigger;
	let content;
	// "idle", "loading", "loaded" or "failed"
	let status = $state("idle");
	// The fragment, rendered by the server
	let html = $state("");
	let enhancement = null;
	// The form went through: the next opening starts over
	let done = false;

	// First field of the form, else the close button
	function focusStart() {
		const field = content?.querySelector(
			"form input:not([type=hidden]):not([type=radio]):not(:disabled)",
		);
		(field ?? dialog.querySelector(".form-popup-close"))?.focus();
	}

	function handleSuccess() {
		done = true;
		if (kind === "login") {
			// The header shows the logged in user
			globalThis.location.reload();
		}
	}

	async function load() {
		status = "loading";
		try {
			const response = await apiFetch(`${url}?fragment=1`);
			if (!response.ok) {
				status = "failed";
				return;
			}
			html = await response.text();
			status = "loaded";
			await tick();

			const target = content.querySelector(`#${kind}-enhancement`);
			if (target) {
				enhancement = mount(FormEnhancement, {
					target,
					props: {
						formSelector: `#${kind}-form`,
						onSuccess: handleSuccess,
					},
				});
			}
		} catch {
			status = "failed";
		}
	}

	async function open() {
		dialog.showModal();
		if (status === "idle" || status === "failed") {
			await load();
		}
		focusStart();
	}

	function close() {
		dialog.close();
	}

	// Escape, the close button or the backdrop: back to the header's button
	async function handleClose() {
		trigger?.focus();
		if (!done) {
			return;
		}
		done = false;
		if (enhancement) {
			await unmount(enhancement);
			enhancement = null;
		}
		html = "";
		status = "idle";
	}

	// Clicking the backdrop targets the dialog itself. Pressed in the form
	// and released over the backdrop (selecting a field's text) isn't one.
	let pressedBackdrop = false;

	function handlePointerdown(event) {
		pressedBackdrop = event.target === dialog;
	}

	function handleClick(event) {
		if (event.target === dialog && pressedBackdrop) {
			close();
		}
		pressedBackdrop = false;
	}

	// The dialog lives in <body>, not in the header's navigation: the
	// compact layout hides that one, and an open dialog would vanish with it
	function portal(node) {
		document.body.append(node);
		return () => node.remove();
	}
</script>

<button
	type="button"
	bind:this={trigger}
	aria-haspopup="dialog"
	onclick={open}
>
	{label}
</button>

<dialog
	class="popup form-popup"
	aria-label={label}
	bind:this={dialog}
	{@attach portal}
	onpointerdown={handlePointerdown}
	onclick={handleClick}
	onclose={handleClose}
>
	<button type="button" class="form-popup-close" aria-label={labelClose} title={labelClose} onclick={close}>
		<svg width="18" height="18" viewBox="0 0 24 24" aria-hidden="true">
			<path d="M6 6l12 12M18 6L6 18" stroke="currentColor" stroke-width="2" stroke-linecap="round" fill="none" />
		</svg>
	</button>

	<div class="form-popup-content" bind:this={content} aria-busy={status === "loading"}>
		{#if status === "loading"}
			<p class="form-popup-status" role="status">{labelLoading}</p>
		{:else if status === "failed"}
			<p class="section-error" role="alert">{labelError}</p>
		{:else if status === "loaded"}
			<!-- Same-origin fragment rendered by the server's html/template,
			     so everything user-provided in it is escaped -->
			<!-- eslint-disable-next-line svelte/no-at-html-tags -->
			{@html html}
		{/if}
	</div>
</dialog>

<style>
	.form-popup {
		--popup-width: 30rem;
	}

	/* The close button sits in the top corner, over the form's own padding */
	.form-popup-close {
		position: absolute;
		top: 0.5rem;
		right: 0.5rem;
		z-index: 1;

		display: inline-flex;
		align-items: center;
		justify-content: center;

		min-width: 2.5rem;
		min-height: 2.5rem;
		padding: 0.25rem;

		color: var(--color-text-muted);
		background: transparent;
		border: 0;
		border-radius: 0.5rem;

		&:hover {
			color: var(--color-text);
			background: var(--color-bg-4);
		}
	}

	.form-popup-content {
		padding: 1.25rem;
	}

	.form-popup-status {
		margin: 0;
		color: var(--color-text-muted);
	}

	/* The fragment's section is the page's card: the popup is the card here */
	.form-popup-content :global(section) {
		margin: 0;
		padding: 0;

		background: none;
		border: 0;
		box-shadow: none;
	}

	/* Room for the close button beside the title */
	.form-popup-content :global(h1) {
		padding-inline: 2.5rem;
	}
</style>
