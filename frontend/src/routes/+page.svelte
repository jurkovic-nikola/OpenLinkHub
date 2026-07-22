<script lang="ts">
	import UiCard from '$lib/components/UiCard.svelte';
	import { getDevices, type DeviceSummary } from '$lib/api';
	import { ws } from '$lib/ws';
	import { onMount } from 'svelte';

	let devices = $state<Record<string, DeviceSummary>>({});

	onMount(() => {
		getDevices()
			.then((res) => {
				if (res.device && typeof res.device === 'object') {
					devices = res.device as Record<string, DeviceSummary>;
				}
			})
			.catch(() => {});

		const off = ws.onMessage((msg) => {
			if (msg.type === 'devices' && msg.data && typeof msg.data === 'object') {
				devices = msg.data as Record<string, DeviceSummary>;
			}
		});
		return () => off();
	});

	const list = $derived(
		Object.values(devices).filter((d) => d.Serial && !d.Hidden)
	);
</script>

<div class="space-y-6">
	<div>
		<h2 class="h2">Dashboard</h2>
		<p class="opacity-60 mt-1">Connected devices and system overview</p>
	</div>

	<div class="grid gap-4 grid-cols-[repeat(auto-fill,minmax(16rem,1fr))]">
		{#each list as device (device.Serial)}
			<a href={`/device/${device.Serial}`} class="block min-w-0 no-underline text-inherit">
				<UiCard title={device.Product ?? 'Device'}>
					<p class="text-sm opacity-70 break-all">{device.Serial}</p>
					{#if device.Firmware}
						<p class="text-xs mt-2 opacity-60">Firmware {device.Firmware}</p>
					{/if}
				</UiCard>
			</a>
		{:else}
			<p class="opacity-60">No devices connected.</p>
		{/each}
	</div>
</div>
