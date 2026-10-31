import { mount } from 'svelte';
import Popup from './FormPopup.svelte';
import FormEnhancement from './FormEnhancement.svelte';
import CardRender from './CardRender.svelte';
import InventoryCardPopup from './InventoryCardPopup.svelte';
import PackOpening from './PackOpening.svelte';
import PackBackground from './PackBackground.svelte';
import ShopBuyPopup from './ShopBuyPopup.svelte';

import NavbarEnhancer from './Navbar.svelte';
import ThemeSwitcher from './ThemeSwitcher.svelte';
import LanguageSwitcher from './LanguageSwitcher.svelte';
import Profile from './ProfileEnhancement.svelte';
import Notification from './Notification.svelte';
import Shoutbox from './Shoutbox.svelte';
import { enhanceNotificationList, enhanceRelativeTimes } from './relative-time.js';
import { openTopup } from './topup.js';

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

// Notifications
{
	const target = document.querySelector('#notification');

	if (target) {
		mount(Notification, {
			target,
			props: {
				dataset: target.dataset,
			},
		});
	}
}

// Shoutbox
{
	const target = document.querySelector('#shoutbox');

	if (target) {
		mount(Shoutbox, {
			target,
			props: {
				dataset: target.dataset,
			},
		});
	}
}

// Notifications page
{
	const target = document.querySelector('#notification-list');

	if (target) {
		enhanceNotificationList(target);
	}
}

// Relative dates of server rendered lists
{
	for (const list of document.querySelectorAll('[data-relative-times]')) {
		enhanceRelativeTimes(list);
	}
}

// Log out, then land on a page visitors can see
{
	for (const button of document.querySelectorAll('[data-logout]')) {
		button.addEventListener('click', async () => {
			button.disabled = true;
			try {
				await fetch(button.dataset.logout, { credentials: 'same-origin' });
			} finally {
				location.assign('/cards/search/');
			}
		});
	}
}

// Account deletion, after confirmation, then land on a page visitors can see
{
	// Error message of a failed deletion response
	async function deletionError(response, fallback) {
		try {
			const body = await response.json();
			return body.message || fallback;
		} catch {
			return fallback;
		}
	}

	for (const target of document.querySelectorAll('[data-account-delete]')) {
		target.addEventListener('click', async (event) => {
			const button = event.currentTarget;
			const error = button.parentElement.querySelector('.section-error');
			if (!confirm(button.dataset.confirm)) {
				return;
			}

			button.disabled = true;
			error.hidden = true;
			try {
				const response = await fetch(button.dataset.accountDelete, {
					method: 'POST',
					credentials: 'same-origin',
				});
				if (response.ok) {
					location.assign('/cards/search/');
					return;
				}
				error.textContent = await deletionError(response, button.dataset.labelError);
				error.hidden = false;
			} catch {
				error.textContent = button.dataset.labelError;
				error.hidden = false;
			} finally {
				button.disabled = false;
			}
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

// Profile form
{
	const target = document.querySelector('#profile-enhancement');

	if (target) {
		mount(Profile, {
			target,
			props: {
				dataset: target.dataset,
			},
		});
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

// Inventory boosters open on click, dragging to tilt them doesn't count
{
	const tiles = document.querySelectorAll('.inventory-booster[data-booster-open] .card-container');
	// Pointer travel (px) after which a press counts as a tilt drag
	const dragThreshold = 6;
	let press;

	for (const tile of tiles) {
		const booster = tile.closest('.inventory-booster');
		const open = () => location.assign(booster.dataset.boosterOpen);

		tile.tabIndex = 0;
		tile.setAttribute('role', 'link');
		tile.setAttribute('aria-label', booster.dataset.boosterLabel ?? '');
		tile.addEventListener('pointerdown', (event) => {
			press = { x: event.clientX, y: event.clientY };
		}, { passive: true });
		tile.addEventListener('click', (event) => {
			if (press && Math.hypot(event.clientX - press.x, event.clientY - press.y) > dragThreshold) {
				return;
			}
			open();
		});
		tile.addEventListener('keydown', (event) => {
			if (event.key !== 'Enter') {
				return;
			}
			event.preventDefault();
			open();
		});
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
				againLabel: target.dataset.againLabel,
				coins: Number(target.dataset.coins) || 0,
				autostart: target.dataset.autostart === 'true',
				soundOnLabel: target.dataset.soundOn,
				soundOffLabel: target.dataset.soundOff,
			},
		});
	}
}

// Pack opening animated background
{
	const target = document.querySelector('#pack-background');

	if (target) {
		mount(PackBackground, { target, props: { target } });
	}
}

// Shop booster buy popup
{
	const target = document.querySelector('#shop-buy-popup');

	if (target) {
		mount(ShopBuyPopup, {
			target,
			props: {
				target,
			},
		});
	}
}

// Shop coin top-up popup
{
	const dialog = document.querySelector('#shop-topup-popup');

	if (dialog) {
		let returnFocus;

		for (const button of document.querySelectorAll('.shop-topup-open')) {
			button.addEventListener('click', () => {
				returnFocus = button;
				openTopup();
			});
		}
		for (const button of dialog.querySelectorAll('[data-shop-topup-close]')) {
			button.addEventListener('click', () => dialog.close());
		}
		// Clicking the backdrop targets the dialog itself
		dialog.addEventListener('click', (event) => {
			if (event.target === dialog) {
				dialog.close();
			}
		});
		dialog.addEventListener('close', () => {
			returnFocus?.focus();
			returnFocus = undefined;
		});
	}
}
