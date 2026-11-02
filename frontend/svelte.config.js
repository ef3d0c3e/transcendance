// Every component uses runes ($props, $state, onclick attributes): the legacy
// syntax (export let, $:, on:click) fails the build instead of mixing in
const config = {
	compilerOptions: {
		runes: true,
	},
};

export default config;
