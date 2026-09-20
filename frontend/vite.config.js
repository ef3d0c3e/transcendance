import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

export default defineConfig({
	plugins: [svelte({
		emitCss: false
	})],

	build: {
		rollupOptions: {
			input: {
				register: "src/register.js",
				login: "src/login.js",
				login_popover: "src/login_popover.js",
				register_popover: "src/register_popover.js",
			},
			output: {
				entryFileNames: "[name].js",
				format: "es",
			},
		},

		emptyOutDir: true,
	},
});
