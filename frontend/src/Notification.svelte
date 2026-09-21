<script>
	import Bell from '../icons/Bell.svelte'

	let notifications = [];
	let loaded = false;
	let loading = false;
	let error = null;
	let visible = false;

	async function loadNotifications() {
		if (loaded || loading) {
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

			notifications = (data.notifications ?? []).slice(0, 5);
			loaded = true;
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

	function time_ago(dateString) {
		const date = new Date(dateString);

		if (Number.isNaN(date.getTime())) {
			return "";
		}

		const seconds = Math.floor((Date.now() - date.getTime()) / 1000);

		if (seconds < 10) {
			return "just now";
		}

		if (seconds < 60) {
			return `${seconds}s ago`;
		}

		const minutes = Math.floor(seconds / 60);

		if (minutes < 60) {
			return `${minutes}m ago`;
		}

		const hours = Math.floor(minutes / 60);

		if (hours < 24) {
			return `${hours}h ago`;
		}

		const days = Math.floor(hours / 24);

		if (days < 7) {
			return `${days}d ago`;
		}

		const weeks = Math.floor(days / 7);

		if (weeks < 5) {
			return `${weeks}w ago`;
		}

		const months = Math.floor(days / 30);

		if (months < 12) {
			return `${months}mo ago`;
		}

		const years = Math.floor(days / 365);

		return `${years}y ago`;
	}
</script>

<div onmouseenter={loadNotifications} role="alert">
<div id="button">
	<button aria-label="Notifications" onclick={menu_open}>
		<Bell/>
	</button>
	{#if loaded && notifications.length > 0}
		<div id="notif-count">
				{notifications.length}
		</div>
	{/if}
	{#if visible}
	<div id="notif-wrapper">
		{#if loading}
		<div>
			<div></div>
			<span>Loading notifications...</span>
		</div>
		{:else if error}
		<div id="error">
			<div>!</div>
			<div>{error}</div>
		</div>
		{:else if notifications.length === 0}
		<div>
			<div>No new notifications</div>
		</div>
		{:else}
		<a href="/notifications">View all notifications</a>
		<e-stack id="notif-list">
			{#each notifications as notification}
			<a href="/notification/{notification.ID}">
				<div id="notif" class:read={notification.Status==="read"}>
						<div>
							{@html notification.icon}
						</div>
						<div>
							{#if notification.Type === "FriendRequest"}
								<h1 id="title">Friend Request</h1>
								<p id="desc">{notification.Data["Username"]} sent you a friend request</p>
							{:else if notification.Type === "Info"}
								<h1 id="title">{notification.Data["title"]}</h1>
								<p id="desc">{notification.Data["description"]}</p>
							{:else}
								<h1 id="title">Undefined Notification</h1>
								<p id="desc">You should not see this</p>
							{/if}
							<p id="time">{time_ago(notification.CreatedAt)}</p>
						</div>
				</div>
			</a>
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
