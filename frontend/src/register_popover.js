import { mount } from 'svelte';
import Popover from './FormPopover.svelte';

const target = document.querySelector('#register-popover');

if (target) {
	mount(Popover, {
		target,
		props: {
			url: '/register?fragment=1',
			formSelector: '#register-form',
			enhancementSelector: '#register-enhancement',
			label: 'Register',
		},
	});
}
