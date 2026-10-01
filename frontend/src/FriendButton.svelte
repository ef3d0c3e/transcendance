<script lang="ts">
	import { onMount } from "svelte";

	let { id } = $props();
	let relation = $state([])
	let action = $state("")

	async function loadRelation() {
		let target = "/api/friends/" + id
		const response = await fetch(target, {
			method: "GET",
		});
		const data = await response.json();
		relation = data.relation;
		switch (relation.toString()) {
			case "pending":
				action  = "cancel"
				break
			case "friends":
				action  = "unfriend"
				break
			case "blocked":
				action =  "unblock"
				break
			case "respond":
				action =  "accept"
				break
			default:
				action = "add"
		}
	}

	onMount(() => {
		loadRelation()
	});

	async function relationAction(act) {
		await fetch(`/friendaction?target=${id}&action=${act}`, {
			method: "GET",
		})
		loadRelation()
	}
</script>

{#if relation == "respond"}
	<p>This user has sent you a friend request</p>
	<button class="deny" onclick={()=>relationAction("deny")}>deny</button>
{/if}
<button class={relation} onclick={()=>relationAction(action)}>{ action }</button>


<style>
.blocked {
	background-color: darkred;
}
.friends {
	background-color: darkgreen;
}
.respond {
	background-color: lightgreen;
}
.deny {
	background-color: red;
}
</style>
