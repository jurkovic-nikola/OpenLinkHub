<script lang="ts">
	import UiCard from '$lib/components/UiCard.svelte';
	import { getTemperatures, postJson, putJson, deleteJson } from '$lib/api';
	import { onMount } from 'svelte';

	let profiles = $state<Record<string, any>>({});
	let selected = $state('');
	let message = $state('');
	let newName = $state('');

	onMount(() => {
		refresh();
	});

	async function refresh() {
		const res = await getTemperatures();
		profiles = (res.data as Record<string, any>) ?? {};
		if (!selected && Object.keys(profiles).length) selected = Object.keys(profiles)[0];
	}

	async function createProfile() {
		if (!newName.trim()) return;
		await postJson('/api/temperatures/new', { profile: newName.trim() });
		newName = '';
		message = 'Created';
		await refresh();
	}

	async function removeProfile() {
		if (!selected) return;
		await deleteJson('/api/temperatures/delete', { profile: selected });
		message = 'Deleted';
		selected = '';
		await refresh();
	}

	async function saveProfile() {
		if (!selected || !profiles[selected]) return;
		await putJson('/api/temperatures/update', { profile: selected, ...profiles[selected] });
		message = 'Saved';
	}
</script>

<div class="space-y-6 max-w-3xl">
	<div>
		<h2 class="h2">Temperatures</h2>
		<p class="opacity-60">Fan and pump temperature profiles</p>
	</div>

	{#if message}
		<p class="text-sm opacity-70">{message}</p>
	{/if}

	<div class="grid gap-4 md:grid-cols-2">
		<UiCard title="Profiles">
			<select class="select w-full" bind:value={selected}>
				{#each Object.keys(profiles) as name}
					<option value={name}>{name}</option>
				{/each}
			</select>
			<div class="flex gap-2 mt-3">
				<button type="button" class="btn preset-tonal" onclick={() => removeProfile()}>Delete</button>
				<a class="btn preset-tonal" href="/temperatureGraphs">Graphs</a>
			</div>
		</UiCard>

		<UiCard title="New profile">
			<input class="input w-full" placeholder="Name" bind:value={newName} />
			<button type="button" class="btn preset-filled-primary-500 mt-3" onclick={() => createProfile()}
				>Create</button
			>
		</UiCard>
	</div>

	{#if selected && profiles[selected]}
		<UiCard title={`Edit: ${selected}`}>
			<pre class="text-xs overflow-auto max-h-80 opacity-80">{JSON.stringify(profiles[selected], null, 2)}</pre>
			<button type="button" class="btn preset-filled-primary-500 mt-3" onclick={() => saveProfile()}
				>Save</button
			>
		</UiCard>
	{/if}
</div>
