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
				login_popup: "src/login-popup.js",
				register_popup: "src/register-popup.js",
				theme_switcher: "src/theme-switcher.js",
			},
			output: {
				entryFileNames: "[name].js",
				format: "es",
			},
		},

		emptyOutDir: true,
	},
});
