import { mount } from 'svelte';
import FriendButton from './FriendButton.svelte';

const friendbutton = mount(FriendButton, {
	target: document.querySelector('#friend-button'),
	props: {
		id: document.querySelector('#friend-button').getAttribute('data-id'),
	}
});
