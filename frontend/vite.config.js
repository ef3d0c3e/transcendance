import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

export default defineConfig({
	plugins: [svelte()],

	build: {
		rollupOptions: {
			input: {
				register: "src/register.js",
				login: "src/login.js",
				notifications: "src/notifications.js",
			},
			output: {
				entryFileNames: "[name].js",
				format: "es",
			},
		},

		emptyOutDir: true,
	},
});
