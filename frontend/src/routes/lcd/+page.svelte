<script lang="ts">
	import UiCard from '$lib/components/UiCard.svelte';
	import { getDevices, type DeviceSummary } from '$lib/api';
	import { onMount } from 'svelte';

	let devices = $state<DeviceSummary[]>([]);

	onMount(async () => {
		const res = await getDevices();
		devices = Object.values((res.device as Record<string, DeviceSummary>) ?? {}).filter(
			(d) => d.Serial && !d.Hidden
		);
	});
</script>

<div class="space-y-6">
	<div>
		<h2 class="h2">LCD</h2>
		<p class="opacity-60">Open a device page to configure LCD modes, rotation, and images</p>
	</div>
	<div class="grid gap-4 grid-cols-[repeat(auto-fill,minmax(16rem,1fr))]">
		{#each devices as device (device.Serial)}
			<a href={`/device/${device.Serial}`} class="block no-underline text-inherit min-w-0">
				<UiCard title={device.Product ?? device.Serial}>
					<p class="text-sm opacity-70">Open device for LCD controls</p>
				</UiCard>
			</a>
		{/each}
	</div>
</div>
