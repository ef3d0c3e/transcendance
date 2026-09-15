import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import { e_layout } from "e-layout";

export default defineConfig({
	plugins: [svelte({
		emitCss: false
	})],

	build: {
		rollupOptions: {
			input: {
				register: "src/register.js",
				login: "src/login.js",
				login_popover: "src/login-popover.js",
				register_popover: "src/register-popover.js",
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
