<!--
    Submits the login or register form in the background and writes the
    server's answer in the form's `.form-result`: its `[role=alert]` for a
    failure, its `[role=status]` for a success. Both live regions are in the
    page from the start, so screen readers announce what is written in.

    The form gives the labels of its own failures: `data-label-error` (the
    answer isn't JSON), `data-label-network` (no answer) and, with a password
    confirmation, `data-label-mismatch`.
-->

<script>
	import { onMount, tick } from "svelte";
	import { apiFetch } from "./api.js";

	// `onSuccess` is called once the success message is shown
	let { formSelector, onSuccess } = $props();

	// Shows the field of the login type picked, the other one is left out of
	// the request. Without JavaScript both are shown.
	function enhanceLoginType(form) {
		const choices = form.querySelectorAll('input[name="login_type"]');
		const fields = form.querySelectorAll("[data-login-field]");

		function updateFields() {
			const selected = form.querySelector('input[name="login_type"]:checked')?.value;
			for (const field of fields) {
				const active = field.dataset.loginField === selected;
				const input = field.querySelector("input");
				field.hidden = !active;
				input.disabled = !active;
				input.required = active;
			}
		}

		updateFields();
		for (const choice of choices) {
			choice.addEventListener("change", updateFields);
		}
		return () => {
			for (const choice of choices) {
				choice.removeEventListener("change", updateFields);
			}
		};
	}

	// The confirmation must match the password, checked by the browser
	// along the other constraints before the form is sent
	function enhanceConfirmation(form) {
		const password = form.elements.password;
		const confirmation = form.elements.password_confirm;
		if (!password || !confirmation) {
			return null;
		}

		function check() {
			const mismatch = confirmation.value !== "" && confirmation.value !== password.value;
			confirmation.setCustomValidity(mismatch ? (form.dataset.labelMismatch ?? "") : "");
		}

		password.addEventListener("input", check);
		confirmation.addEventListener("input", check);
		return () => {
			password.removeEventListener("input", check);
			confirmation.removeEventListener("input", check);
		};
	}

	// Writes a message in the result, clearing the other kind. Cleared a frame
	// before, so the same message twice is announced again.
	function report(result, kind, text) {
		const failure = result.querySelector('[role="alert"]');
		const success = result.querySelector('[role="status"]');
		failure.textContent = "";
		success.textContent = "";
		return new Promise((resolve) => {
			requestAnimationFrame(() => {
				(kind === "error" ? failure : success).textContent = text;
				resolve();
			});
		});
	}

	onMount(() => {
		const form = document.querySelector(formSelector);
		const result = form?.querySelector(".form-result");

		if (!form || !result) {
			return;
		}

		const cleanupLoginType = form.id === "login-form" ? enhanceLoginType(form) : null;
		const cleanupConfirmation = enhanceConfirmation(form);

		async function handleSubmit(event) {
			event.preventDefault();

			const button = form.querySelector("button[type=submit]");
			// Not disabled: the focus would fall off the button while waiting
			if (button.getAttribute("aria-disabled") === "true") {
				return;
			}
			button.setAttribute("aria-disabled", "true");

			try {
				const response = await apiFetch(form.action, {
					method: form.method,
					body: new FormData(form),
				});

				// A proxy error page or a body too large isn't JSON
				const data = await response.json().catch(() => null);
				if (!response.ok || !data) {
					await report(result, "error", data?.message ?? form.dataset.labelError ?? "");
					return;
				}

				await report(result, "success", data.message ?? "");

				// Only the message stays, where the focus goes as the
				// submitted button is gone
				form.reset();
				form.replaceChildren(result);
				const message = result.querySelector('[role="status"]');
				message.tabIndex = -1;
				await tick();
				message.focus();
				onSuccess?.();
			} catch {
				await report(result, "error", form.dataset.labelNetwork ?? "");
			} finally {
				button.removeAttribute("aria-disabled");
			}
		}

		form.addEventListener("submit", handleSubmit);

		return () => {
			cleanupLoginType?.();
			cleanupConfirmation?.();
			form.removeEventListener("submit", handleSubmit);
		};
	});
</script>
