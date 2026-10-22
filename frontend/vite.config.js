import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

export default defineConfig({
	plugins: [svelte({
		emitCss: false
	})],

	build: {
		rollupOptions: {
			input: {
				main: "src/main.js",
				register: "src/register.js",
				login: "src/login.js",
				profile: "src/profile.js",
			},
			output: {
				entryFileNames: "[name].js",
				format: "es",
			},
		},

		emptyOutDir: true,
	},
});
