<script lang="ts">
	import UiCard from '$lib/components/UiCard.svelte';
	import { onMount } from 'svelte';

	let macros = $state<Record<string, any> | any[]>([]);
	let message = $state('');

	onMount(async () => {
		const res = await fetch('/api/macro/');
		const json = await res.json();
		macros = json.data ?? [];
	});

	const entries = $derived(
		Array.isArray(macros)
			? macros.map((m, i) => [String(i), m] as const)
			: Object.entries(macros as Record<string, any>)
	);
</script>

<div class="space-y-6">
	<div>
		<h2 class="h2">Macros</h2>
		<p class="opacity-60">Macro profiles from the API</p>
	</div>
	{#if message}
		<p class="text-sm">{message}</p>
	{/if}
	<div class="grid gap-4 grid-cols-[repeat(auto-fill,minmax(16rem,1fr))]">
		{#each entries as [id, macro]}
			<UiCard title={(macro as any)?.Name ?? `Macro ${id}`}>
				<pre class="text-xs opacity-70 overflow-auto max-h-48">{JSON.stringify(macro, null, 2)}</pre>
			</UiCard>
		{:else}
			<p class="opacity-60">No macros found.</p>
		{/each}
	</div>
</div>
