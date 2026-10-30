<script>
	import { onMount } from "svelte";

	let { formSelector, onSuccess } = $props();

	function enhanceLoginType(form) {
		const choices = form.querySelectorAll('input[name="login_type"]');
		const fields = form.querySelectorAll('[data-login-field]');

		function updateFields() {
			const selected = form.querySelector('input[name="login_type"]:checked')?.value;
			for (const field of fields) {
				const active = field.dataset.loginField === selected;
				const input = field.querySelector('input');
				field.hidden = !active;
				input.disabled = !active;
				input.required = active;
			}
		}

		updateFields();
		for (const choice of choices) {
			choice.addEventListener('change', updateFields);
		}
		return () => {
			for (const choice of choices) {
				choice.removeEventListener('change', updateFields);
			}
		};
	}

	onMount(() => {
		const form = document.querySelector(formSelector);
		const result = form?.querySelector("#form-result");

		if (!form || !result) {
			return;
		}

		const cleanupLoginType = form.id === "login-form" ? enhanceLoginType(form) : null;

		async function handleSubmit(event) {
			event.preventDefault();

			const button = form.querySelector("button[type=submit]");

			button.disabled = true;

			try {
				const response = await fetch(form.action, {
					method: form.method,
					body: new FormData(form),
				});

				const data = await response.json();

				var section;
				if (!result.hasChildNodes()) {
					section = document.createElement("section");
					result.appendChild(section);
				} else {
					section = result.childNodes[0];
				}

				if (!response.ok) {
					section.className = "section-error";
					section.textContent = data.message ?? "Request failed.";
					return;
				}

				section.className = "section-success";
				section.textContent = data.message;

				form.reset();
				form.replaceChildren(result);
				onSuccess();
			} catch (error) {
				result.textContent = "Unable to contact the server.";
			} finally {
				button.disabled = false;
			}
		}

		form.addEventListener("submit", handleSubmit);

		return () => {
			cleanupLoginType?.();
			form.removeEventListener("submit", handleSubmit);
		};
	});
</script>
