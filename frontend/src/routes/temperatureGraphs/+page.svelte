<script lang="ts">
	import UiCard from '$lib/components/UiCard.svelte';
	import { getTemperatures } from '$lib/api';
	import { onMount } from 'svelte';

	let profiles = $state<Record<string, any>>({});

	onMount(async () => {
		const res = await getTemperatures();
		profiles = (res.data as Record<string, any>) ?? {};
	});
</script>

<div class="space-y-6">
	<div>
		<h2 class="h2">Temperature graphs</h2>
		<p class="opacity-60">Graph-capable profiles from the temperatures API</p>
	</div>

	<div class="grid gap-4 grid-cols-[repeat(auto-fill,minmax(16rem,1fr))]">
		{#each Object.entries(profiles) as [name, profile]}
			<UiCard title={name}>
				<p class="text-sm opacity-70 break-words">
					{profile?.Graph ? 'Graph profile' : 'Standard profile'}
				</p>
				<a class="anchor text-sm mt-2 inline-block" href="/temperature">Edit profiles</a>
			</UiCard>
		{/each}
	</div>
</div>
