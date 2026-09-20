import { mount } from 'svelte';
import Popover from './FormPopover.svelte';

const target = document.querySelector('#login-popover');

if (target) {
	mount(Popover, {
		target,
		props: {
			url: '/login',
			target,
			formSelector: '#login-form',
			enhancementSelector: '#login-enhancement',
		},
	});
}
