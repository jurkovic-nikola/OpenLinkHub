<script lang="ts">
	import { page } from '$app/stores';
	import { getDevice } from '$lib/api';
	import { ws } from '$lib/ws';
	import { deviceFamily } from '$lib/deviceTypes';
	import HubDeviceView from '$lib/components/HubDeviceView.svelte';
	import GenericDeviceView from '$lib/components/GenericDeviceView.svelte';
	import { onMount } from 'svelte';

	let device = $state<Record<string, any> | null>(null);
	let error = $state('');
	let serial = $derived($page.params.serial ?? '');

	function unwrap(payload: unknown): Record<string, any> | null {
		if (!payload || typeof payload !== 'object') return null;
		const obj = payload as Record<string, any>;
		// API may return device object directly or nested
		if (obj.Serial || obj.Product || obj.Devices || obj.DeviceProfile) return obj;
		if (obj.GetDevice && typeof obj.GetDevice === 'object') return obj.GetDevice;
		return obj;
	}

	onMount(() => {
		if (!serial) return;

		getDevice(serial)
			.then((res) => {
				device = unwrap(res.device) ?? unwrap(res.data);
				if (!device) error = 'Device not found';
			})
			.catch((e) => {
				error = e instanceof Error ? e.message : 'Failed to load device';
			});

		ws.subscribeDevice(serial);
		const off = ws.onMessage((msg) => {
			if (msg.type === 'device' && msg.serial === serial && msg.data) {
				device = unwrap(msg.data) ?? device;
			}
		});

		return () => {
			off();
			ws.unsubscribeDevice(serial);
		};
	});

	const family = $derived(deviceFamily(device?.ProductType));
</script>

{#if error}
	<p class="text-error-500">{error}</p>
{:else if !device}
	<p class="opacity-60">Loading device…</p>
{:else if family === 'hub'}
	<HubDeviceView {serial} {device} />
{:else}
	<GenericDeviceView {serial} {device} {family} />
{/if}
