<script>
	import { onMount } from "svelte";

	let { formSelector, onSuccess } = $props();

	onMount(() => {
		const form = document.querySelector(formSelector);
		const result = form.querySelector("#form-result");

		if (!form || !result) {
			return;
		}

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
			form.removeEventListener("submit", handleSubmit);
		};
	});
</script>
