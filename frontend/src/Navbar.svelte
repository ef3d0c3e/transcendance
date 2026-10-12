<script lang>
	import { onMount } from 'svelte';

	let { selector = 'header' } = $props();

	onMount(() => {
		const header = document.querySelector(selector);
		const title = header?.querySelector('.header-title');
		const nav = header?.querySelector('#header-nav');
		const toggle = header?.querySelector('#nav-toggle');
		if (!header || !title || !nav || !toggle) return;

		header.classList.add('js');
		toggle.setAttribute('aria-controls', 'header-nav');

		let compact = false;

		function update() {
			header.classList.remove('is-compact', 'nav-animate');

			const cs = getComputedStyle(header);
			const available =
				header.clientWidth - parseFloat(cs.paddingLeft) - parseFloat(cs.paddingRight);
			const needed =
				title.getBoundingClientRect().width +
				nav.getBoundingClientRect().width +
				parseFloat(cs.columnGap);

			const next = needed > available;
			header.classList.toggle('is-compact', next);

			if (next !== compact) {
				compact = next;
				toggle.checked = false;
			}
		}

		const onChange = () => header.classList.add('nav-animate');
		const onKeydown = (e) => {
			if (e.key === 'Escape' && toggle.checked) {
				header.classList.add('nav-animate');
				toggle.checked = false;
			}
		};
		toggle.addEventListener('change', onChange);
		toggle.addEventListener('keydown', onKeydown);

		update();

		const resize = new ResizeObserver(update);
		resize.observe(header);

		// Nav content can change (e.g. "Log in" becomes a username)
		const mutation = new MutationObserver(update);
		mutation.observe(nav, { childList: true, subtree: true, characterData: true });

		document.fonts?.ready.then(update);

		return () => {
			toggle.removeEventListener('change', onChange);
			toggle.removeEventListener('keydown', onKeydown);
			resize.disconnect();
			mutation.disconnect();
			header.classList.remove('js', 'is-compact', 'nav-animate');
		};
	});
</script>
