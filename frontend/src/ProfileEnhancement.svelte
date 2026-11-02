<!--
    Profile editor: only the changed fields are sent, in the background. The
    server answers a message per field, written next to each field (tied to
    it by its aria-describedby) and flagged with aria-invalid; a failure of
    the whole request, or the success, goes to the `#profile-result` live
    regions. The save button is only shown while something changed.
-->

<svelte:options css="injected" />
<style>
	/* Checkmark drawn next to the save button once saved */
	:global(#profile-form button.saved) {
		--checkmark-color: var(--color-success-3);
		--checkmark-height: 2rem;
		--checkmark-width: 1rem;
		position: relative;
	}

	:global(#profile-form button.saved)::after {
		content: "";
		display: inline-block;
		position: absolute;
		left: calc(100% + 0.5rem);
		top: 50%;
		height: var(--checkmark-height);
		width: var(--checkmark-width);
		border-right: 3px solid var(--checkmark-color);
		border-top: 3px solid var(--checkmark-color);
		transform: scaleX(-1) rotate(135deg);
		transform-origin: left top;
		animation: checkmark 800ms ease forwards;
	}

	@keyframes checkmark {
		0% {
			height: 0;
			width: 0;
			opacity: 1;
		}
		10% {
			height: 0;
			width: var(--checkmark-width);
			opacity: 1;
		}
		20% {
			height: var(--checkmark-height);
			width: var(--checkmark-width);
			opacity: 1;
		}
		75% {
			height: var(--checkmark-height);
			width: var(--checkmark-width);
			opacity: 1;
		}
		100% {
			height: var(--checkmark-height);
			width: var(--checkmark-width);
			opacity: 0;
		}
	}
</style>

<script>
	import { onMount } from "svelte";
	import { apiFetch } from "./api.js";

	let { dataset } = $props();

	// Writes a message in the result, clearing the other kind. Cleared a frame
	// before, so the same message twice is announced again.
	function report(result, kind, text) {
		const failure = result.querySelector('[role="alert"]');
		const success = result.querySelector('[role="status"]');
		failure.textContent = "";
		success.textContent = "";
		if (text) {
			requestAnimationFrame(() => {
				(kind === "error" ? failure : success).textContent = text;
			});
		}
	}

	// The new password and its confirmation must match, checked by the
	// browser along the other constraints before the form is sent
	function enhanceConfirmation(form) {
		const password = form.elements.new_password;
		const confirmation = form.elements.confirm_password;
		if (!password || !confirmation) {
			return null;
		}

		function check() {
			const mismatch = confirmation.value !== "" && confirmation.value !== password.value;
			confirmation.setCustomValidity(mismatch ? (dataset.labelMismatch ?? "") : "");
		}

		password.addEventListener("input", check);
		confirmation.addEventListener("input", check);
		return () => {
			password.removeEventListener("input", check);
			confirmation.removeEventListener("input", check);
		};
	}

	function enhanceAvatar() {
		const avatarForm = document.querySelector("#avatar-upload-form");
		const avatarFile = document.querySelector("#avatar-file");
		const avatarButton = document.querySelector("#avatar-upload-button");
		const avatarDeleteButton = document.querySelector("#avatar-delete-button");
		const avatarDeleteDialog = document.querySelector("#avatar-delete-dialog");

		function confirmAvatarDeletion() {
			avatarDeleteDialog?.showModal();
		}

		// Cancelling comes back to the button that asked
		function restoreFocus() {
			avatarDeleteButton?.focus();
		}

		function chooseAvatar() {
			avatarFile?.click();
		}

		function uploadAvatar() {
			if (avatarFile?.files.length) {
				avatarForm?.requestSubmit();
			}
		}

		avatarDeleteButton?.addEventListener("click", confirmAvatarDeletion);
		avatarDeleteDialog?.addEventListener("close", restoreFocus);
		avatarButton?.addEventListener("click", chooseAvatar);
		avatarFile?.addEventListener("change", uploadAvatar);

		return () => {
			avatarDeleteButton?.removeEventListener("click", confirmAvatarDeletion);
			avatarDeleteDialog?.removeEventListener("close", restoreFocus);
			avatarButton?.removeEventListener("click", chooseAvatar);
			avatarFile?.removeEventListener("change", uploadAvatar);
		};
	}

	onMount(() => {
		const form = document.querySelector("#profile-form");
		const result = document.querySelector("#profile-result");

		if (!form || !result) {
			return;
		}

		const cleanupAvatar = enhanceAvatar();
		const cleanupConfirmation = enhanceConfirmation(form);

		const button = form.querySelector("button[type=submit]");
		const errorFields = form.querySelectorAll(".profile-error");
		const fields = form.querySelectorAll("input, textarea, select");

		// Shown again as soon as a field changes
		button.hidden = true;

		function safelyHide(element) {
			if (element.contains(document.activeElement)) {
				form.querySelector("input")?.focus();
			}
			element.hidden = true;
		}

		function hideButtonOnSave() {
			button.classList.remove("saved");
			safelyHide(button);
		}

		function hideButtonAfterSave() {
			if (button.classList.contains("saved")) {
				button.addEventListener("animationend", hideButtonOnSave, { once: true });
			} else {
				safelyHide(button);
			}
		}

		// The confirmation and current password are only asked for along a
		// new password
		function showPasswordFields(shown) {
			for (const name of ["confirm_password", "password"]) {
				const field = form.elements[name];
				field.disabled = !shown;
				if (shown) {
					field.parentElement.hidden = false;
				} else {
					safelyHide(field.parentElement);
				}
			}
		}

		function isModified(field) {
			const initial = field.tagName === "SELECT"
				? [...field.options].find((option) => option.defaultSelected)?.value
				: field.defaultValue;
			return initial !== field.value;
		}

		function handleChange(event) {
			const field = event.currentTarget;
			const modified = isModified(field);

			if (field.name === "new_password") {
				showPasswordFields(modified);
			}
			field.classList.toggle("modified", modified);

			if (form.querySelectorAll(".modified").length === 0) {
				safelyHide(button);
			} else {
				button.hidden = false;
			}
		}

		// Writes each field's message next to it, flags it, and moves the
		// focus to the first one in error
		function showFieldErrors(messages) {
			let first = null;
			for (const errorField of errorFields) {
				const name = errorField.id.replace(/^profile-error-/, "");
				if (!(name in messages)) {
					continue;
				}
				const message = messages[name];
				errorField.textContent = message;
				errorField.hidden = !message;

				const input = form.elements[name];
				if (!input) {
					continue;
				}
				if (message) {
					input.setAttribute("aria-invalid", "true");
					first ??= input;
				} else {
					input.removeAttribute("aria-invalid");
				}
			}
			first?.focus();
		}

		// The saved values become the fields' defaults: nothing is modified
		function markSaved() {
			for (const field of fields) {
				field.classList.remove("modified");
				if (field.tagName === "SELECT") {
					const { value } = field;
					for (const option of field.options) {
						option.defaultSelected = option.value === value;
					}
				} else if (!field.name.includes("password")) {
					field.defaultValue = field.value;
				}
			}
			form.reset();
			if (form.elements.new_password) {
				showPasswordFields(false);
			}
		}

		async function handleSubmit(event) {
			event.preventDefault();

			// Not disabled: the focus would fall off the button while waiting
			if (button.getAttribute("aria-disabled") === "true") {
				return;
			}
			button.setAttribute("aria-disabled", "true");

			const formData = new FormData();
			for (const field of form.querySelectorAll(".modified")) {
				formData.append(field.name, field.value);
			}
			report(result, "error", "");

			try {
				// The profile endpoint always answers JSON, it never redirects
				const response = await apiFetch(form.action, {
					method: form.method,
					body: formData,
					headers: {
						Accept: "application/json",
					},
				});

				// Server errors may not be JSON
				const data = await response.json().catch(() => null);
				if (!data) {
					report(result, "error", dataset.labelUnexpectedError);
					return;
				}

				if (!response.ok && typeof data.message === "string") {
					report(result, "error", data.message);
					return;
				}

				showFieldErrors(data.message ?? {});
				if (response.ok) {
					if (data.profileUrl) {
						window.history.replaceState(window.history.state, "", data.profileUrl);
					}
					report(result, "success", dataset.labelSaved);
					// Restart the checkmark even when it just ran
					button.classList.remove("saved");
					void button.offsetWidth;
					button.classList.add("saved");
					markSaved();
					hideButtonAfterSave();
				}
			} catch {
				report(result, "error", dataset.labelNetworkError);
			} finally {
				button.removeAttribute("aria-disabled");
			}
		}

		form.addEventListener("submit", handleSubmit);
		for (const field of fields) {
			field.addEventListener("input", handleChange);
		}

		return () => {
			cleanupAvatar();
			cleanupConfirmation?.();
			form.removeEventListener("submit", handleSubmit);
			button.removeEventListener("animationend", hideButtonOnSave);
			for (const field of fields) {
				field.removeEventListener("input", handleChange);
			}
		};
	});
</script>
