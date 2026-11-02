<!--
    Header navigation: when the title and the links don't fit side by side,
    the header turns compact and the navigation becomes a side menu, opened
    by the header's menu button (aria-expanded) and closed by its own close
    button, Escape, or a click or the focus going outside of it. Opening it
    moves the focus in, closing it with the keyboard brings it back.

    Other components close the menu with a `nav-menu-close` window event,
    e.g. the chat, whose panel can't be shown inside it.
-->

<script>
	import { onMount } from 'svelte';

	let { selector = 'header' } = $props();

	onMount(() => {
		const header = document.querySelector(selector);
		const title = header?.querySelector('.header-title');
		const nav = header?.querySelector('#header-nav');
		const toggle = header?.querySelector('.nav-toggle-btn');
		const closeButton = nav?.querySelector('.nav-close');
		if (!header || !title || !nav || !toggle) return;

		header.classList.add('js');

		let compact = false;

		function isOpen() {
			return header.classList.contains('nav-open');
		}

		function setOpen(open, { animate = true, focus = false } = {}) {
			header.classList.toggle('nav-animate', animate);
			header.classList.toggle('nav-open', open);
			toggle.setAttribute('aria-expanded', String(open));
			if (!focus) return;
			if (open) {
				nav.querySelector('.nav-links a, a, button:not(.nav-close)')?.focus();
			} else {
				toggle.focus();
			}
		}

		function update() {
			header.classList.remove('is-compact', 'nav-animate');
			// The nav grows to fill the header, measure its content instead
			header.classList.add('is-measuring');

			const cs = getComputedStyle(header);
			const available =
				header.clientWidth - parseFloat(cs.paddingLeft) - parseFloat(cs.paddingRight);
			const needed =
				title.getBoundingClientRect().width +
				nav.getBoundingClientRect().width +
				parseFloat(cs.columnGap);
			header.classList.remove('is-measuring');

			const next = needed > available;
			header.classList.toggle('is-compact', next);

			if (next !== compact) {
				compact = next;
				setOpen(false, { animate: false });
			}
		}

		const onToggle = () => setOpen(!isOpen(), { focus: true });
		const onClose = () => setOpen(false, { focus: true });

		// Escape works from anywhere in the menu, not only from its button
		const onKeydown = (e) => {
			if (e.key === 'Escape' && isOpen()) {
				setOpen(false, { focus: true });
			}
		};

		// A click outside of the menu closes it, as does the focus leaving it.
		// A popup opened from the menu (log in, register) gives the focus back
		// to its button in the menu: the menu stays open beneath it.
		const onPointerdown = (e) => {
			if (!isOpen() || e.target.closest?.('dialog[open]')) return;
			if (!nav.contains(e.target) && !toggle.contains(e.target)) {
				setOpen(false);
			}
		};
		const onFocusout = (e) => {
			const next = e.relatedTarget;
			if (!isOpen() || !(next instanceof Element) || next.closest('dialog[open]')) return;
			if (!nav.contains(next) && !toggle.contains(next)) {
				setOpen(false);
			}
		};
		const onCloseRequest = () => {
			if (isOpen()) setOpen(false);
		};

		toggle.addEventListener('click', onToggle);
		closeButton?.addEventListener('click', onClose);
		document.addEventListener('keydown', onKeydown);
		document.addEventListener('pointerdown', onPointerdown);
		nav.addEventListener('focusout', onFocusout);
		globalThis.addEventListener('nav-menu-close', onCloseRequest);

		update();

		const resize = new ResizeObserver(update);
		resize.observe(header);

		// Nav content can change (e.g. "Log in" becomes a username)
		const mutation = new MutationObserver(update);
		mutation.observe(nav, { childList: true, subtree: true, characterData: true });

		document.fonts?.ready.then(update);

		return () => {
			toggle.removeEventListener('click', onToggle);
			closeButton?.removeEventListener('click', onClose);
			document.removeEventListener('keydown', onKeydown);
			document.removeEventListener('pointerdown', onPointerdown);
			nav.removeEventListener('focusout', onFocusout);
			globalThis.removeEventListener('nav-menu-close', onCloseRequest);
			resize.disconnect();
			mutation.disconnect();
			header.classList.remove('js', 'is-compact', 'nav-animate', 'nav-open');
		};
	});
</script>
