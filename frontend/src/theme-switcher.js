import { mount } from 'svelte';
import ThemeSwitcher from './ThemeSwitcher.svelte';

const target = document.querySelector('#theme-switcher');

if (target) {
	mount(ThemeSwitcher, {
		target,
	});
}
