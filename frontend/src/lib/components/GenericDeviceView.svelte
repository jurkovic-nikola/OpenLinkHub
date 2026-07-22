<script lang="ts">
	import UiCard from '$lib/components/UiCard.svelte';
	import StatRow from '$lib/components/StatRow.svelte';
	import type { DeviceFamily } from '$lib/deviceTypes';
	import { postJson } from '$lib/api';

	let {
		serial,
		device,
		family
	}: {
		serial: string;
		device: Record<string, any>;
		family: DeviceFamily;
	} = $props();

	let busy = $state(false);
	let message = $state('');

	async function setBrightness(value: number) {
		busy = true;
		try {
			await postJson('/api/brightness/gradual', { deviceId: serial, brightness: value });
			message = 'Brightness updated';
		} catch (e) {
			message = e instanceof Error ? e.message : 'Failed';
		} finally {
			busy = false;
		}
	}
</script>

<div class="space-y-6">
	<div>
		<p class="text-xs uppercase tracking-wide opacity-60">{family}</p>
		<h2 class="h2 break-words">{device?.Product ?? 'Device'}</h2>
		<p class="opacity-60 text-sm break-all">{serial}</p>
		{#if message}
			<p class="text-sm mt-2 opacity-70">{message}</p>
		{/if}
	</div>

	<div class="grid gap-4 grid-cols-[repeat(auto-fill,minmax(16rem,1fr))]">
		<UiCard title="Overview">
			<StatRow label="Firmware" value={device?.Firmware} />
			{#if device?.BatteryLevel != null}
				<StatRow label="Battery" value={`${device.BatteryLevel}%`} />
			{/if}
			{#if device?.DeviceProfile?.BrightnessSlider != null}
				<div class="pt-2 space-y-2">
					<label class="text-xs uppercase tracking-wide text-surface-600-400" for="brightness"
						>Brightness</label
					>
					<input
						id="brightness"
						type="range"
						class="w-full"
						min="0"
						max="100"
						value={device.DeviceProfile.BrightnessSlider}
						disabled={busy}
						oninput={(e) => setBrightness(Number((e.currentTarget as HTMLInputElement).value))}
					/>
				</div>
			{/if}
		</UiCard>

		{#if family === 'keyboard'}
			<UiCard title="Keyboard">
				<p class="text-sm opacity-70">
					Use RGB and Macros pages for lighting and key assignments. Live LED sync uses REST + WebSocket
					updates.
				</p>
				<a class="anchor text-sm mt-3 inline-block" href={`/rgb?device=${serial}`}>RGB editor</a>
				<a class="anchor text-sm mt-2 ml-3 inline-block" href="/macros">Macros</a>
			</UiCard>
		{/if}

		{#if family === 'mouse'}
			<UiCard title="Mouse">
				{#if device?.Dpi != null}
					<StatRow label="DPI" value={device.Dpi} />
				{/if}
				<p class="text-sm opacity-70 mt-2">DPI stages and zone colors are available via the API.</p>
			</UiCard>
		{/if}

		{#if family === 'headset'}
			<UiCard title="Headset">
				<p class="text-sm opacity-70">Equalizer and sidetone controls use the headset REST endpoints.</p>
			</UiCard>
		{/if}
	</div>
</div>
