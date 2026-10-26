import { mount } from 'svelte';
import Friends from './FriendsEnhancement.svelte';

const target = document.querySelector('#friends-list');

if (target) {
	mount(Friends, {
		target,
	});
}
