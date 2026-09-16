<script>
	import Bell from '../icons/Bell.svelte'

	let notifications = [];
	let loaded = false;
	let loading = false;
	let error = null;

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
	<button aria-label="Notifications" style="width: 40px; height: 40px;">
		<Bell style="position: absolute;"/>
		{#if loaded && notifications.length > 0}
			<div id="notif-count" style="position: relative; background-color: lightblue; border-radius: 50%; width: 20px; height: 20px; top: -10px; left: 15px; z-index: 1;">
				<p style="text-align: center;">
					{notifications.length}
				</p>
			</div>
		{/if}
	</button>

	<div style="max-width: 50%;">
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
		<div id="notif-list">
			{#each notifications as notification}
			<div class:unread={notification.status==="unread" }>
				<div>
					{@html notification.icon}
				</div>

				<div id="notif" style="border: solid; border-radius: 15px;">
					<div id="title" style="padding-left: 20px;">
						<p>{notification.title}</p>
						{#if notification.status === "unread"}
						<div></div>
						{/if}
					</div>
						<p id="desc">{notification.description}</p>
						<p id="time" style="padding-left: 20px;">{time_ago(notification.date)}</p>
				</div>
			</div>
			{/each}
		</div>
		{/if}
	</div>
</div>

<!--FIXME style tag does not apply for some reason, resorting to inline for now-->
<style>
	#notif{
		border: solid;
		border-radius: 35px;
	}
	#notif > *{
		padding-left: 30px;
	}
</style>
