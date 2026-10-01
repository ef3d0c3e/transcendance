import { mount } from 'svelte';
import FriendButton from './FriendButton.svelte';

if (document.querySelector("#friend-button") != null) {
	const friendbutton = mount(FriendButton, {
		target: document.querySelector('#friend-button'),
		props: {
			id: document.querySelector('#friend-button').getAttribute('data-id'),
		}
	});
}
