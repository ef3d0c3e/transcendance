<script lang="ts">
	import { onMount } from "svelte";

	let { id } = $props();
	let relation = $state([])

	async function loadRelation() {
		console.log(id)
		let target = "/api/friends/" + id
		const response = await fetch(target, {
			method: "GET",
		});
		const data = await response.json();
		relation = data.relation;
	}

	onMount(() => {
		loadRelation()
		console.log(relation)
	});
</script>

<p>{ relation }</p>
