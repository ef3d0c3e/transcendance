<svelte:options css="injected" />
<style>
/* Style et animation pour checkmark */

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

	let { dataset } = $props();

	onMount(() => {
		const form = document.querySelector("#profile-form");
		const result = document.querySelector("#profile-result");

		if (!form || !result) {
			return;
		}
		const avatarForm = document.querySelector("#avatar-upload-form");
		const avatarFile = document.querySelector("#avatar-file");
		const avatarButton = document.querySelector("#avatar-upload-button");

		const avatarDeleteButton = document.querySelector("#avatar-delete-button");
		const avatarDeleteDialog = document.querySelector("#avatar-delete-dialog");

		function confirmAvatarDeletion() {
			avatarDeleteDialog?.showModal();
		}

		avatarDeleteButton?.addEventListener("click", confirmAvatarDeletion);

		function chooseAvatar() {
			avatarFile?.click();
		}

		function uploadAvatar() {
			if (avatarFile?.files.length) {
				avatarForm?.requestSubmit();
			}
		}

		avatarButton?.addEventListener("click", chooseAvatar);
		avatarFile?.addEventListener("change", uploadAvatar);

		const button = form.querySelector("button[type=submit]");
		const errorFields = form.querySelectorAll(".profile-error")

		function safelyHide(el) {
			if (el.contains(document.activeElement)) {
				form.querySelector("input")?.focus();
			}
			el.hidden = true;
		}

		function hideButtonOnSave() {
			button.classList.remove("saved")
			safelyHide(button);
		}

		function hideButtonAfterSave() {
			if (button.classList.contains("saved")) {
				button.addEventListener("animationend", hideButtonOnSave, { once: true });
			} else {
				safelyHide(button);
			}
		}

		function handleChange(event) {
			const field = event.currentTarget;

			if (field.defaultValue !== field.value) {
				if (field.name === "new_password") {
					form.elements["confirm_password"].disabled = false;
					form.elements["password"].disabled = false;
					form.elements["confirm_password"].parentElement.hidden = false;
					form.elements["password"].parentElement.hidden = false;
				}
				field.classList.add("modified");
			} else {
				if (field.name === "new_password") {
					form.elements["confirm_password"].disabled = true;
					form.elements["password"].disabled = true;
					safelyHide(form.elements["confirm_password"].parentElement);
					safelyHide(form.elements["password"].parentElement);
				}
				field.classList.remove("modified");
			}
			if (form.querySelectorAll(".modified").length == 0) {
				safelyHide(button);
			} else {
				button.hidden = false;
			}
		}

		async function handleSubmit(event) {
			event.preventDefault();

			const formData = new FormData();

			button.disabled = true;
			for (const field of form.querySelectorAll(".modified")) {
				formData.append(field.name, field.value);
			}
			result.textContent = "";

			try {
				const response = await fetch(form.action, {
					method: form.method,
					body: formData,
					headers: {
						'Accept': 'application/json'
					}
				});

				if (response.redirected) {
					window.location.assign(response.url);
					return;
				}

				// Server errors may not be JSON
				const data = await response.json().catch(() => null);
				if (!data) {
					result.textContent = dataset.labelUnexpectedError;
					return;
				}

				if (!response.ok && typeof data.message === "string") {
					result.textContent = data.message;
					return;
				}

				result.textContent = "";
				for (const [fieldName, errorMsg] of Object.entries(data.message)) {
					for (const field of errorFields) {
						if (field.id == `profile-error-${fieldName}`) {
							field.hidden = !errorMsg;
							field.textContent = errorMsg
							break;
						}
					}
				}
				if (response.ok) {
					if (data.profileUrl) {
						window.history.replaceState(window.history.state, "", data.profileUrl);
					}
					button.classList.remove("saved");
					void button.offsetWidth;
					button.classList.add("saved");
					for (const field of form.querySelectorAll("input, textarea")) {
						field.classList.remove("modified");
						if (!field.name.includes("password"))
							field.defaultValue = field.value;
					}
					form.reset();
					if (form.elements["new_password"]) {
						form.elements["confirm_password"].disabled = true;
						form.elements["password"].disabled = true;
						safelyHide(form.elements["confirm_password"].parentElement);
						safelyHide(form.elements["password"].parentElement);
					}
					hideButtonAfterSave();
				}
			} catch (error) {
				result.textContent = dataset.labelNetworkError;
			} finally {
				button.disabled = false;
			}
		}

		form.addEventListener("submit", handleSubmit);
		for (const field of form.querySelectorAll("input, textarea")) {
			field.addEventListener("input", handleChange);
		}

		return () => {
			avatarDeleteButton?.removeEventListener("click", confirmAvatarDeletion);
			avatarButton?.removeEventListener("click", chooseAvatar);
			avatarFile?.removeEventListener("change", uploadAvatar);
			form.removeEventListener("submit", handleSubmit);
			button.removeEventListener("animationend", hideButtonOnSave);
			for (const field of form.querySelectorAll("input, textarea")) {
				field.removeEventListener("input", handleChange);
			}
		};
	});
</script>
