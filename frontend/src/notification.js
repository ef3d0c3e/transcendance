import { mount } from 'svelte';
import notification from './Notification.svelte';

const target = document.querySelector('#notification');

if (target) {
	mount(notification, {
		target,
	});
}
