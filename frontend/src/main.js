import { mount } from 'svelte';
import Popup from './FormPopup.svelte';
import FormEnhancement from './FormEnhancement.svelte';

import NavbarEnhancer from './Navbar.svelte';
import ThemeSwitcher from './ThemeSwitcher.svelte';
import LanguageSwitcher from './LanguageSwitcher.svelte';

// Navbar
{
	mount(NavbarEnhancer, { target: document.body });
}

// Language switcher
{
	const target = document.querySelector('#language-switcher');

	if (target) {
		mount(LanguageSwitcher, {
			target,
		});
	}
}

// Theme switcher
{
	const target = document.querySelector('#theme-switcher');
	if (target) {
		mount(ThemeSwitcher, {
			target,
		});
	}
}

// Login popup
{
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
}

// Register popup
{
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
}

// Login form
{
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
}

// Register form
{
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
}
