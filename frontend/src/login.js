import { mount } from 'svelte';
import Login from './LoginEnhancement.svelte';

const target = document.querySelector('#login-enhancement');

if (target) {
	mount(Login, {
		target,
	});
}
