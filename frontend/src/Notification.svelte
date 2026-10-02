<script>
	import Bell from '../icons/Bell.svelte'
	export let dataset;

	let notifications = [];
	let lastLoaded = null;
	let loading = false;
	let error = null;
	let visible = false;

	async function loadNotifications() {
		if (loading || (lastLoaded != null && (new Date() - lastLoaded) < 5000)) {
			return;
		}
		// Don't load while the menu is open
		if (lastLoaded != null && visible) {
			return;
		}

		loading = true;
		error = null;

		try {
			const response = await fetch("/api/notifications", {
				method: "GET",
				credentials: "include",
			});

			if (!response.ok) {
				throw new Error(`HTTP ${response.status}`);
			}

			const data = await response.json();

			notifications = (data.notifications ?? []);
			lastLoaded = new Date();
		} catch (err) {
			console.error("Failed to load notifications:", err);
			error = "Could not load notifications";
		} finally {
			loading = false;
		}
	}

	function menu_open() {
		visible = true;
		document.body.addEventListener('click', event => {
			if (event.target.closest('#button')) {
				return;
			}
			menu_close();
		});
	}

	function menu_close() {
		visible = false;
		document.body.removeEventListener('click', event => {
			if (event.target.closest('#button')) {
				return;
			}
			menu_close();
		});
	}

loadNotifications()
</script>

<div onmouseenter={loadNotifications} role="alert">
<div id="button">
	<button aria-label="Notifications" onclick={menu_open}>
		<Bell/>
	</button>
	{#if lastLoaded != null && notifications.length > 0}
		<div id="notif-count">
				{notifications.length}
		</div>
	{/if}
	{#if visible}
	<div id="notif-wrapper">
		{#if loading}
		<div>
			<div></div>
			<span>{dataset.labelLoading}</span>
		</div>
		{:else if error}
		<div id="error">
			<div>!</div>
			<div>{error}</div>
		</div>
		{:else if notifications.length === 0}
		<div>
			<div>{dataset.labelNone}</div>
		</div>
		{:else}
		<a href="/notifications">{dataset.labelViewAll}</a>
		<e-stack id="notif-list">
			{#each notifications.slice(0, 5) as notification}
			<div id="notif" class:read={notification.Status==="read"}>
				<a href="/notification/{notification.ID}">
					<div>
						{@html notification.icon}
					</div>
					<div>
						{#if notification.Type === "FriendRequest"}
							<h1 id="title">{notification.Locale["Title"]}</h1>
							<p id="desc">{notification.Locale["Desc"]}</p>
						{:else if notification.Type === "Info"}
							<h1 id="title">{notification.Data["title"]}</h1>
							<p id="desc">{notification.Data["description"]}</p>
						{:else}
							<h1 id="title">Undefined Notification</h1>
							<p id="desc">You should not see this</p>
						{/if}
					</div>
				</a>
				<span>
					{#if notification.Type === "FriendRequest"}
						<a href="/notification/{notification.ID}/accept">{notification.Locale["Accept"]}</a>
						<a href="/notification/{notification.ID}/deny">{notification.Locale["Deny"]}</a>
					{/if}
					<p id="time">{notification.Locale["Time"]}</p>
				</span>
			</div>
			{/each}
		</e-stack>
		{/if}
	</div>
	{/if}
</div>
</div>

<style>
	#notif-count{
	 position: relative;
	 background-color: lightblue;
	 border-radius: 50%;
	 width: 20px;
	 height: 20px;
	 top: -20px;
	 left: 28px;
	 z-index: 1;
	 text-align: center;
	 }
	#notif-wrapper{
		position: fixed;
		max-height: 50vh;
		max-width: 50%;
		padding: 10px;
		background-color: lightcyan;
		overflow-y: scroll;
	}
	#notif{
		border-radius: 15px;
	}
	.read{
		background-color: white;
	}
	a{
		text-decoration: none;
		color: black;
	}
	#notif h1{
		font-size: medium;
	}
	#notif p{
		font-size: small;
	}
	#notif > *{
		padding-left: 30px;
	}
	#time{
		text-align: right;
	}
</style>
