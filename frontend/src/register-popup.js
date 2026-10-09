import { mount } from 'svelte';
import Popup from './FormPopup.svelte';

const target = document.querySelector('#register-popup');

if (target) {
	mount(Popup, {
		target,
		props: {
			url: '/register',
			target,
			formSelector: '#register-form',
			enhancementSelector: '#register-enhancement',
		},
	});
}
