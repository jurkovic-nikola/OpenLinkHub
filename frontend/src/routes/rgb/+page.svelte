<script lang="ts">
	import UiCard from '$lib/components/UiCard.svelte';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';

	let profiles = $state<Record<string, any>>({});
	let message = $state('');
	let deviceFilter = $derived($page.url.searchParams.get('device') ?? '');

	onMount(async () => {
		const res = await fetch('/api/color/');
		const json = await res.json();
		profiles = (json.data as Record<string, any>) ?? {};
	});
</script>

<div class="space-y-6">
	<div>
		<h2 class="h2">RGB</h2>
		<p class="opacity-60">
			Color profiles{#if deviceFilter}
				for device {deviceFilter}{/if}
		</p>
	</div>

	{#if message}
		<p class="text-sm">{message}</p>
	{/if}

	<div class="grid gap-4 grid-cols-[repeat(auto-fill,minmax(16rem,1fr))]">
		{#each Object.keys(profiles) as name}
			<UiCard title={name}>
				<pre class="text-xs opacity-70 overflow-auto max-h-40">{JSON.stringify(profiles[name], null, 2)}</pre>
			</UiCard>
		{:else}
			<p class="opacity-60">No RGB profiles loaded.</p>
		{/each}
	</div>
</div>
