<script lang="ts">
	import { onMount } from "svelte";

	let { id, block = false } = $props()
	let relation = $state([])
	let response = $state([])
	let action = $state("")
	let error = $state("")

	async function loadRelation() {
		let target = "/api/friends/" + id
		try {
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
		catch (err)
		{
			error = err;
		}
	}

	onMount(() => {
		loadRelation()
	});

	async function relationAction(act) {
		const r = await fetch(`/friendaction?target=${id}&action=${act}`, {
			method: "GET",
		})
		const d = await r.json()
		response = d.response
		loadRelation()
	}
</script>

{#if error != ""}
	<span>error: {{error}}</span>
{/if}
{#if relation.toString() != "you"}
	{#if relation.toString() == "respond"}
		<button class="deny" onclick={()=>relationAction("deny")}>deny</button>
	{/if}
	<button class={action} onclick={()=>relationAction(action)}>{ action }</button>
	{#if block && action.toString() != "unblock"}
		<button class="deny" onclick={()=>relationAction("block")}>block</button>
	{/if}
{/if}

{#if response && response.toString() != "ok"}
	<div id="friend-response-popup">{response}</div>
{/if}

<style>
.block {
	background-color: darkred;
}
.unblock {
	background-color: darkred;
}
#friend-response-popup {
	position: fixed;
	bottom: 5%;
	align-self: center;
	text-align: center;
	background-color: darkred;
	border-radius: 10px;
}
.add {
	background-color: darkgreen;
}
.accept {
	background-color: lightgreen;
}
.deny {
	background-color: red;
}
</style>
