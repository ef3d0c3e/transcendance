<script lang="ts">
	import FriendButton from "./FriendButton.svelte";

	async function loadFriends() {
		const res = await fetch('/api/friends');
		const data = await res.json();
		console.log(data.relations);
		return new Promise((fulfil) => {
		fulfil(data.relations ?? []);
		})
	}
	let promise = $state(loadFriends());

</script>

{#await promise}
	<p>Loading relations...</p>
{:then relations}
	{#if relations.length === 0}
		<p>No Friends</p>
	{:else}
		{#each relations as relation}
			<div>
				<p>{relation.Target.Username}</p>
				<FriendButton data-id={relation.TargetID}></FriendButton>
			</div>
		{/each}
	{/if}
{/await}
