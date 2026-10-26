import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

export default defineConfig({
	plugins: [svelte({
		emitCss: false
	})],

	build: {
		rollupOptions: {
			input: {
<<<<<<< HEAD
				register: "src/register.js",
				login: "src/login.js",
				friendbutton: "src/friendbutton.js",
=======
				main: "src/main.js",
>>>>>>> master
			},
			output: {
				entryFileNames: "[name].js",
				format: "es",
			},
		},

		emptyOutDir: true,
	},
});
