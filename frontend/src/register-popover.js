import { mount } from 'svelte';
import Popover from './FormPopover.svelte';

const target = document.querySelector('#register-popover');

if (target) {
	mount(Popover, {
		target,
		props: {
			url: '/register',
			target,
			formSelector: '#register-form',
			enhancementSelector: '#register-enhancement',
		},
	});
}
