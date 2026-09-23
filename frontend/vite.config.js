import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

export default defineConfig({
	plugins: [svelte()],

	build: {
		rollupOptions: {
			input: {
				register: "src/register.js",
				login: "src/login.js",
				friendbutton: "src/friendbutton.js",
			},
			output: {
				entryFileNames: "[name].js",
				format: "es",
			},
		},

		emptyOutDir: true,
	},
});
