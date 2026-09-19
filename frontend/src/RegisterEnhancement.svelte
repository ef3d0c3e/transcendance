<script>
	import { onMount } from "svelte";

	onMount(() => {
		const form = document.querySelector("#register-form");
		const result = document.querySelector("#register-result");

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

				result.innerHTML = '';
				const section = document.createElement("section")
				result.appendChild(section)
				if (!response.ok) {
					section.className = "section-error";
					section.textContent = data.message ?? "Registration failed.";
					return;
				}
				else {
					section.className = "section-success";
					section.textContent = data.message;
				}


				form.reset();
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
