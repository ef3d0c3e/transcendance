import { mount } from "svelte";
import Register from "./RegisterEnhancement.svelte";

const target = document.querySelector("#register-enhancement");

if (target) {
	mount(Register, {
		target,
	});
}
