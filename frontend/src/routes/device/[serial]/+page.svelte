<script lang="ts">
	import { page } from '$app/stores';
	import { getDevice, getDevices } from '$lib/api';
	import { ws } from '$lib/ws';
	import { deviceFamily } from '$lib/deviceTypes';
	import HubDeviceView from '$lib/components/HubDeviceView.svelte';
	import GenericDeviceView from '$lib/components/GenericDeviceView.svelte';
	import { onMount } from 'svelte';

	let device = $state<Record<string, any> | null>(null);
	let productType = $state<number | undefined>(undefined);
	let error = $state('');
	let serial = $derived($page.params.serial ?? '');

	function unwrap(payload: unknown): Record<string, any> | null {
		if (!payload || typeof payload !== 'object') return null;
		const obj = payload as Record<string, any>;
		if (obj.GetDevice && typeof obj.GetDevice === 'object') {
			return obj.GetDevice as Record<string, any>;
		}
		return obj;
	}

	function isHubPayload(d: Record<string, any> | null): boolean {
		if (!d) return false;
		if (d.Template === 'cc.html' || d.Template === 'ccxt.html' || d.Template === 'lsh.html') return true;
		if (d.devices || d.Devices) return true;
		return deviceFamily(productType) === 'hub';
	}

	onMount(() => {
		if (!serial) return;

		getDevices()
			.then((res) => {
				const map = (res.device as Record<string, any>) ?? {};
				const summary = map[serial];
				if (summary?.ProductType != null) productType = summary.ProductType;
			})
			.catch(() => {});

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

	const family = $derived(deviceFamily(productType));
	const useHub = $derived(isHubPayload(device) || family === 'hub');
</script>

{#if error}
	<p class="text-error-500">{error}</p>
{:else if !device}
	<p class="opacity-60">Loading device…</p>
{:else if useHub}
	<HubDeviceView {serial} {device} />
{:else}
	<GenericDeviceView {serial} {device} {family} />
{/if}
