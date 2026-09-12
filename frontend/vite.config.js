import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

export default defineConfig({
	plugins: [svelte()],

	build: {
		lib: {
			entry: "src/register.js",
			formats: ["es"],
			fileName: "register",
		},
		lib: {
			entry: "src/login.js",
			formats: ["es"],
			fileName: "login",
		},

		emptyOutDir: true,
	},
});
