import { mount } from 'svelte';
import Popup from './FormPopup.svelte';
import FormEnhancement from './FormEnhancement.svelte';
import CardRender from './CardRender.svelte';
import InventoryCardPopup from './InventoryCardPopup.svelte';
import InventoryBoosterPopup from './InventoryBoosterPopup.svelte';
import TradeUpPicker from './TradeUpPicker.svelte';
import TradeUpResult from './TradeUpResult.svelte';
import PackOpening from './PackOpening.svelte';

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

// Cards
{
	const target = document.querySelector('#card-rendering');

	if (target) {
		mount(CardRender, {target});
	}
}

// Inventory card popup
{
	const target = document.querySelector('#inventory-card-popup');

	if (target) {
		mount(InventoryCardPopup, { target, props: { target } });
	}
}

// Selects that apply as soon as they change (inventory filters)
{
	const forms = new Set();
	for (const select of document.querySelectorAll('select[data-autosubmit]')) {
		select.addEventListener('change', () => select.form?.requestSubmit());
		if (select.form) {
			forms.add(select.form);
		}
	}

	// Leave unset fields out of the URL, collected first since deleting while iterating skips entries
	for (const form of forms) {
		form.addEventListener('formdata', (event) => {
			const empty = [];
			for (const [name, value] of event.formData) {
				if (value === '') {
					empty.push(name);
				}
			}
			for (const name of empty) {
				event.formData.delete(name);
			}
		});
	}
}

// Inventory booster popup
{
	const target = document.querySelector('#inventory-booster-popup');

	if (target) {
		mount(InventoryBoosterPopup, { target, props: { target } });
	}
}

// Tradeup contract picker
{
	const target = document.querySelector('#tradeup-picker');

	if (target) {
		mount(TradeUpPicker, { target, props: { target } });
	}
}

// Tradeup result popup
{
	const target = document.querySelector('#tradeup-result');

	if (target) {
		mount(TradeUpResult, { target, props: { target } });
	}
}

// Pack opening
{
	const target = document.querySelector('#pack-opening');

	if (target) {
		mount(PackOpening, {
			target,
			props: {
				formSelector: '#pack-form',
				openLabel: target.dataset.openLabel,
				retryLabel: target.dataset.retryLabel,
				errorLabel: target.dataset.errorLabel,
				obtainedLabel: target.dataset.obtainedLabel,
				returnLabel: target.dataset.returnLabel,
				returnUrl: target.dataset.returnUrl,
				openAgainLabel: target.dataset.openAgainLabel,
				autostart: target.dataset.autostart === 'true',
			},
		});
	}
}
