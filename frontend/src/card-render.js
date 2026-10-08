import { mount } from 'svelte';
import Render from './CardRender.svelte';

// Add this to any page that renders cards, this will let svelte apply effects on each card
const target = document.querySelector('#card-rendering');

if (target) {
	mount(Render, {target});
}
