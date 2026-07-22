<script lang="ts">
	import UiCard from '$lib/components/UiCard.svelte';
	import { onMount } from 'svelte';

	let data = $state<any>(null);

	onMount(async () => {
		try {
			const res = await fetch('/api/systray');
			data = await res.json();
		} catch {
			data = null;
		}
	});
</script>

<div class="space-y-6 max-w-3xl">
	<div>
		<h2 class="h2">Scheduler</h2>
		<p class="opacity-60">RGB scheduler and related automation</p>
	</div>
	<UiCard title="Status">
		<pre class="text-xs overflow-auto opacity-80">{JSON.stringify(data, null, 2)}</pre>
	</UiCard>
</div>
