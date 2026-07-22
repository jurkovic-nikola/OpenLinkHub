<script lang="ts">
	import UiCard from '$lib/components/UiCard.svelte';
	import { getDevices, postJson, type DeviceSummary } from '$lib/api';
	import { onMount } from 'svelte';

	let devices = $state<DeviceSummary[]>([]);
	let message = $state('');

	onMount(async () => {
		const res = await getDevices();
		devices = Object.values((res.device as Record<string, DeviceSummary>) ?? {}).filter(
			(d) => d.Serial && !d.Hidden
		);
	});

	async function setCluster(serial: string, enabled: boolean) {
		await postJson('/api/color/setCluster', { deviceId: serial, enabled });
		message = `Cluster ${enabled ? 'enabled' : 'disabled'} for ${serial}`;
	}
</script>

<div class="space-y-6">
	<div>
		<h2 class="h2">RGB Cluster</h2>
		<p class="opacity-60">Toggle cluster mode per device</p>
	</div>
	{#if message}
		<p class="text-sm opacity-70">{message}</p>
	{/if}
	<div class="grid gap-4 grid-cols-[repeat(auto-fill,minmax(16rem,1fr))]">
		{#each devices as device (device.Serial)}
			<UiCard title={device.Product ?? device.Serial}>
				<div class="flex gap-2">
					<button
						type="button"
						class="btn preset-filled-primary-500"
						onclick={() => setCluster(device.Serial!, true)}>Enable</button
					>
					<button type="button" class="btn preset-tonal" onclick={() => setCluster(device.Serial!, false)}
						>Disable</button
					>
				</div>
			</UiCard>
		{/each}
	</div>
</div>
