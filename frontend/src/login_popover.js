import { mount } from 'svelte';
import Popover from './FormPopover.svelte';

const target = document.querySelector('#login-popover');

if (target) {
	mount(Popover, {
		target,
		props: {
			url: '/login?fragment=1',
			formSelector: '#login-form',
			enhancementSelector: '#login-enhancement',
			label: 'Login',
		},
	});
}
