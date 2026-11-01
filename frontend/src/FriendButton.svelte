<script lang="ts">
	import { onMount } from "svelte";

	let { dataset } = $props()
	let relation = $state([])
	let popup = $state([])
	let action = $state("")
	let text = $state("")
	let error = $state("")

	async function loadRelation() {
		let target = "/api/friends/" + dataset.id
		try {
			const response = await fetch(target, {
				method: "GET",
			});
			const data = await response.json();
			relation = data.relation;
			switch (relation.toString()) {
				case "pending":
					action  = "cancel"
					text	= dataset.labelcancel
					break
				case "friends":
					action  = "unfriend"
					text	= dataset.labelunfriend
					break
				case "blocked":
					action =  "unblock"
					text	= dataset.labelunblock
					break
				case "respond":
					action =  "accept"
					text	= dataset.labelaccept
					break
				default:
					action = "add"
					text	= dataset.labeladd
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
		const r = await fetch(`/friendaction?target=${dataset.id}&action=${act}`, {
			method: "POST",
		})
		const d = await r.json()
		popup = d.response
		loadRelation()
	}
	
	$effect(() => {
		setInterval(() => {
			popup = "";
		}, 3000);
	});

</script>

<div>
{#if error != ""}
	<span>error: {{error}}</span>
{/if}
{#if relation.toString() != "you"}
	{#if relation.toString() == "respond"}
		<button class="deny" onclick={()=>relationAction("deny")}>{ dataset.labeldeny }</button>
	{/if}
	<button class={action} onclick={()=>relationAction(action)}>{ text }</button>
	{#if dataset.block && action == "add"}
		<button class="deny" onclick={()=>relationAction("block")}>{ dataset.labelblock }</button>
	{/if}
{/if}

{#key popup}
	{#if popup && popup.toString() != "ok"}
		<div id="friend-response-popup">{popup}</div>
	{/if}
{/key}
</div>

<style>
div {
	display: flex;
	justify-content: center;
}
button {
	margin: 2%;
	width: 100px;
	padding-left: 0;
	padding-right: 0;
}
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
	border-radius: 2px;
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
