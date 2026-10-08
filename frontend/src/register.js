import { mount } from 'svelte';
import FormEnhancement from './FormEnhancement.svelte';

const target = document.querySelector('#register-enhancement');

if (target) {
	mount(FormEnhancement, {
		target,
		props: {
			formSelector: '#register-form',
			onSuccess: () => {},
		},
	});
}
