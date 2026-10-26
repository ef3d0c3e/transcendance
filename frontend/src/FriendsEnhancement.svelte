<script lang="ts">
	import { onMount } from "svelte";
	import FriendButton from "./FriendButton.svelte";

	let relations = $state([]);

	onMount(() => {
		async function loadFriends() {
			const res = await fetch('/api/friends');
			const data = await res.json();
			relations = (data.relations ?? []);
		}
		loadFriends();
	})
</script>

{#if relations.length === 0}
	<p>No Friends</p>
{:else}
	{#each relations as relation}
		<div>
			<p>1</p>
			<p>{relation.Target.Username}</p>
			<FriendButton data-id={relation.TargetID}></FriendButton>
		</div>
	{/each}
	<p>FooBar test</p>
{/if}
