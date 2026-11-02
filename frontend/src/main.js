import { mount } from 'svelte';
import Popup from './FormPopup.svelte';
import FormEnhancement from './FormEnhancement.svelte';
import FriendButton from './FriendButton.svelte';
import CardRender from './CardRender.svelte';
import InventoryCardPopup from './InventoryCardPopup.svelte';
import PackOpening from './PackOpening.svelte';
import PackBackground from './PackBackground.svelte';
import ShopBuyPopup from './ShopBuyPopup.svelte';
import TradeMenu from './TradeMenu.svelte';
import TradeUpPicker from './TradeUpPicker.svelte';
import TradeUpResult from './TradeUpResult.svelte';

import NavbarEnhancer from './Navbar.svelte';
import ThemeSwitcher from './ThemeSwitcher.svelte';
import LanguageSwitcher from './LanguageSwitcher.svelte';
import Profile from './ProfileEnhancement.svelte';
import Notification from './Notification.svelte';
import Shoutbox from './Shoutbox.svelte';
import { enhanceNotificationList, enhanceRelativeTimes } from './relative-time.js';
import { enhanceCountdowns } from './countdown.js';
import { openTopup } from './topup.js';
import { apiFetch } from './api.js';

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
			props: {
				label: target.dataset.label,
				labelCurrent: target.dataset.labelCurrent,
			},
		});
	}
}

// Theme switcher
{
	const target = document.querySelector('#theme-switcher');
	if (target) {
		mount(ThemeSwitcher, {
			target,
			props: {
				labelLight: target.dataset.themeLabelLight,
				labelDark: target.dataset.themeLabelDark,
			},
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

// Countdown pills, e.g the time left before the next booster
{
	enhanceCountdowns();
}

// Log out in the background, then land on the home page, which visitors can
// see. Without JavaScript the form posts and the server redirects there.
{
	for (const form of document.querySelectorAll('form[data-logout]')) {
		form.addEventListener('submit', async (event) => {
			event.preventDefault();
			form.querySelector('button')?.setAttribute('aria-disabled', 'true');
			try {
				const response = await apiFetch(form.dataset.logout, {
					method: 'POST',
					headers: { Accept: 'application/json' },
				});
				if (response.ok) {
					location.assign('/');
					return;
				}
			} catch {
				// Offline: posted as is below, the browser tells what went wrong
			}
			// Refused: the plain form shows the server's own answer
			form.submit();
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
				const response = await apiFetch(button.dataset.accountDelete, {
					method: 'POST',
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

// Login and register popups: the header's links to their pages open the
// form in a popup instead, except on the page that shows it already
{
	for (const kind of ['login', 'register']) {
		const target = document.querySelector(`#${kind}-popup`);
		if (!target || document.querySelector(`#${kind}-form`)) {
			continue;
		}

		target.replaceChildren();
		mount(Popup, {
			target,
			props: {
				url: `/${kind}`,
				kind,
				label: target.dataset.label,
				labelLoading: target.dataset.labelLoading,
				labelError: target.dataset.labelError,
				labelClose: target.dataset.labelClose,
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

// Friend buttons
{
	for (const target of document.querySelectorAll('.friend-button')) {
		mount(FriendButton, {
			target,
			props: {
				dataset: target.dataset,
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

// Filters of the search forms, applied by their button: choosing an option
// doesn't reload the page, the arrow keys go through the options
{
	const forms = new Set();
	for (const select of document.querySelectorAll('select[data-filter]')) {
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

// Inventory boosters are links to their opening: dragging one to tilt it
// isn't a click on it
{
	// Pointer travel (px) after which a press counts as a tilt drag
	const dragThreshold = 6;

	for (const link of document.querySelectorAll('a.inventory-booster[data-booster-open]')) {
		let press;
		link.addEventListener('pointerdown', (event) => {
			press = { x: event.clientX, y: event.clientY };
		}, { passive: true });
		link.addEventListener('click', (event) => {
			if (press && Math.hypot(event.clientX - press.x, event.clientY - press.y) > dragThreshold) {
				event.preventDefault();
			}
			press = undefined;
		});
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
				againLabel: target.dataset.againLabel,
				coins: Number(target.dataset.coins) || 0,
				autostart: target.dataset.autostart === 'true',
				soundLabel: target.dataset.soundLabel,
				motionLabel: target.dataset.motionLabel,
				skipLabel: target.dataset.skipLabel,
				spinningLabel: target.dataset.spinningLabel,
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

// Trade menu
{
	const target = document.querySelector('#trade-menu');

	if (target) {
		mount(TradeMenu, {
			target,
			props: {
				target,
			},
		});
	}
}

// Confirmation popups opened by a button, e.g selling the duplicate cards
{
	for (const button of document.querySelectorAll('[data-dialog-open]')) {
		const dialog = document.querySelector(`#${CSS.escape(button.dataset.dialogOpen)}`);
		if (!dialog) {
			continue;
		}
		button.setAttribute('aria-controls', dialog.id);
		button.addEventListener('click', () => dialog.showModal());
		for (const close of dialog.querySelectorAll('[data-dialog-close]')) {
			close.addEventListener('click', () => dialog.close());
		}
		// Clicking the backdrop targets the dialog itself
		dialog.addEventListener('click', (event) => {
			if (event.target === dialog) {
				dialog.close();
			}
		});
		// Focus comes back to the button, as it opened the popup
		dialog.addEventListener('close', () => button.focus());
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
