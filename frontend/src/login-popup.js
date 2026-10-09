import { mount } from 'svelte';
import Popup from './FormPopup.svelte';

const target = document.querySelector('#login-popup');

if (target) {
	mount(Popup, {
		target,
		props: {
			url: '/login',
			target,
			formSelector: '#login-form',
			enhancementSelector: '#login-enhancement',
		},
	});
}
