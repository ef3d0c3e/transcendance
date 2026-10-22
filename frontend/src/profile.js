import { mount } from 'svelte';
import Profile from './ProfileEnhancement.svelte';

const target = document.querySelector('#profile-enhancement');

if (target) {
	mount(Profile, {
		target,
	});
}
