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
			case  "blocked":
				action =  "unblock"
				break
			default:
				action = "add"
		}
	}

	onMount(() => {
		loadRelation()
	});

	async function relationAction() {
		await fetch(`/friendaction?target=${id}&action=${action}`, {
			method: "GET",
		})
		loadRelation()
	}
</script>

<button class={relation} onclick={()=>relationAction()}>{ action }</button>

<style>
.blocked {
	background-color: red;
}
</style>
