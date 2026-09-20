import { mount } from 'svelte';
import FormEnhancement from './FormEnhancement.svelte';

const target = document.querySelector('#login-enhancement');

if (target) {
	mount(FormEnhancement, {
		target,
		props: {
			formSelector: '#login-form',
			onSuccess: () => {},
		},
	});
}
