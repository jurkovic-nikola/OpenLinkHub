<script lang="ts">
	import UiCard from '$lib/components/UiCard.svelte';
	import StatRow from '$lib/components/StatRow.svelte';
	import { postJson, putJson, deleteJson } from '$lib/api';

	let {
		serial,
		device
	}: {
		serial: string;
		device: Record<string, any>;
	} = $props();

	const profile = $derived(device?.DeviceProfile ?? {});
	const channels = $derived(
		Array.isArray(device?.Devices) ? (device.Devices as Record<string, any>[]) : []
	);
	const rgbDevices = $derived(
		Array.isArray(device?.RgbDevices) ? (device.RgbDevices as Record<string, any>[]) : []
	);
	const temps = $derived(
		device?.TemperatureProfiles && typeof device.TemperatureProfiles === 'object'
			? (device.TemperatureProfiles as Record<string, any>)
			: {}
	);
	const rgbModes = $derived(Array.isArray(device?.RGBModes) ? (device.RGBModes as string[]) : []);
	const userProfiles = $derived(
		device?.UserProfiles && typeof device.UserProfiles === 'object'
			? (device.UserProfiles as Record<string, any>)
			: {}
	);

	let brightness = $state(100);
	let busy = $state(false);
	let message = $state('');

	$effect(() => {
		brightness = Number(profile?.BrightnessSlider ?? 100);
	});

	async function run(label: string, fn: () => Promise<unknown>) {
		busy = true;
		message = '';
		try {
			await fn();
			message = label;
		} catch (e) {
			message = e instanceof Error ? e.message : 'Request failed';
		} finally {
			busy = false;
		}
	}

	function setSpeed(channelId: number, profileName: string) {
		return run('Speed updated', () =>
			postJson('/api/speed', { deviceId: serial, channelId, profile: profileName })
		);
	}

	function setBrightness(value: number) {
		brightness = value;
		return run('Brightness updated', () =>
			postJson('/api/brightness/gradual', { deviceId: serial, brightness: value })
		);
	}

	function setGlobalRgb(mode: string) {
		return run('RGB updated', () =>
			postJson('/api/color/global', { deviceId: serial, profile: mode })
		);
	}

	function setGlobalSpeed(profileName: string) {
		return run('Speed profile updated', () =>
			postJson('/api/speed', { deviceId: serial, channelId: -1, profile: profileName })
		);
	}

	function toggleCluster(checked: boolean) {
		return run('Cluster updated', () =>
			postJson('/api/color/setCluster', { deviceId: serial, enabled: checked })
		);
	}

	function toggleOpenRgb(checked: boolean) {
		return run('OpenRGB updated', () =>
			postJson('/api/color/setOpenRgbIntegration', { deviceId: serial, enabled: checked })
		);
	}

	function changeProfile(name: string) {
		return run('Profile changed', () =>
			postJson('/api/userProfile/change', { deviceId: serial, profileName: name })
		);
	}

	function saveProfile() {
		return run('Profile saved', () => putJson('/api/userProfile', { deviceId: serial }));
	}

	function deleteProfile(name: string) {
		if (!name || name === 'none') return;
		return run('Profile deleted', () =>
			deleteJson('/api/userProfile/delete', { deviceId: serial, profileName: name })
		);
	}

	function setLabel(channelId: number) {
		const label = prompt('Label');
		if (label == null) return;
		return run('Label saved', () =>
			postJson('/api/label', { deviceId: serial, channelId, label })
		);
	}

	function setChannelRgb(channelId: number, mode: string) {
		return run('Channel RGB updated', () =>
			postJson('/api/color', { deviceId: serial, channelId, profile: mode })
		);
	}

	const tempOptions = $derived(
		Object.entries(temps).filter(([, v]) => !(v as { Hidden?: boolean })?.Hidden)
	);
</script>

<div class="space-y-6">
	<div class="flex flex-wrap items-end justify-between gap-3">
		<div class="min-w-0">
			<h2 class="h2 break-words">{device?.Product ?? 'Hub'}</h2>
			<p class="opacity-60 text-sm break-all">{serial}</p>
		</div>
		{#if message}
			<p class="text-sm opacity-70">{message}</p>
		{/if}
	</div>

	<div class="grid gap-4 lg:grid-cols-[minmax(18rem,22rem)_1fr]">
		<div class="space-y-4 min-w-0">
			<UiCard title={device?.Product ?? 'Settings'}>
				<StatRow label="Firmware" value={device?.Firmware} />
				<div class="py-2 border-b border-surface-200-800/60 space-y-1">
					<label class="text-xs uppercase tracking-wide text-surface-600-400" for="profiles">Profile</label>
					<select
						id="profiles"
						class="select w-full"
						disabled={busy}
						onchange={(e) => changeProfile((e.currentTarget as HTMLSelectElement).value)}
					>
						{#each Object.keys(userProfiles) as name}
							<option value={name} selected={userProfiles[name]?.Active}>{name}</option>
						{/each}
					</select>
				</div>
				<div class="py-2 border-b border-surface-200-800/60 space-y-2">
					<label class="text-xs uppercase tracking-wide text-surface-600-400" for="brightness"
						>Brightness</label
					>
					<input
						id="brightness"
						class="w-full"
						type="range"
						min="0"
						max="100"
						value={brightness}
						disabled={busy}
						oninput={(e) => setBrightness(Number((e.currentTarget as HTMLInputElement).value))}
					/>
					<p class="text-sm">{brightness} %</p>
				</div>
				<div class="flex items-center justify-between gap-3 py-2 border-b border-surface-200-800/60">
					<span class="text-xs uppercase tracking-wide text-surface-600-400">Cluster</span>
					<input
						type="checkbox"
						class="checkbox"
						checked={!!profile?.RGBCluster}
						disabled={busy}
						onchange={(e) => toggleCluster((e.currentTarget as HTMLInputElement).checked)}
					/>
				</div>
				<div class="flex items-center justify-between gap-3 py-2 border-b border-surface-200-800/60">
					<span class="text-xs uppercase tracking-wide text-surface-600-400">OpenRGB</span>
					<input
						type="checkbox"
						class="checkbox"
						checked={!!profile?.OpenRGBIntegration}
						disabled={busy}
						onchange={(e) => toggleOpenRgb((e.currentTarget as HTMLInputElement).checked)}
					/>
				</div>
				<div class="py-2 border-b border-surface-200-800/60 space-y-1">
					<label class="text-xs uppercase tracking-wide text-surface-600-400" for="globalRgb">RGB</label>
					<select
						id="globalRgb"
						class="select w-full"
						disabled={busy}
						onchange={(e) => setGlobalRgb((e.currentTarget as HTMLSelectElement).value)}
					>
						<option value="">None</option>
						{#each rgbModes as mode}
							<option value={mode}>{mode}</option>
						{/each}
					</select>
				</div>
				<div class="py-2 border-b border-surface-200-800/60 space-y-1">
					<label class="text-xs uppercase tracking-wide text-surface-600-400" for="globalSpeed"
						>Speed</label
					>
					<select
						id="globalSpeed"
						class="select w-full"
						disabled={busy}
						onchange={(e) => setGlobalSpeed((e.currentTarget as HTMLSelectElement).value)}
					>
						{#each tempOptions as [name]}
							<option value={name} selected={profile?.MultiProfile === name}>{name}</option>
						{/each}
					</select>
				</div>
				<div class="flex flex-col gap-2 pt-3">
					<button type="button" class="btn preset-filled-primary-500" disabled={busy} onclick={() => saveProfile()}
						>Save user profile</button
					>
					<select
						id="deleteProfile"
						class="select w-full"
						disabled={busy}
						onchange={(e) => deleteProfile((e.currentTarget as HTMLSelectElement).value)}
					>
						<option value="none">Delete profile…</option>
						{#each Object.entries(userProfiles) as [name, p]}
							{#if !p?.Active}
								<option value={name}>{name}</option>
							{/if}
						{/each}
					</select>
				</div>
			</UiCard>
		</div>

		<div class="space-y-6 min-w-0">
			<section class="space-y-3">
				<h3 class="h4">Channels</h3>
				<div class="grid gap-4 grid-cols-[repeat(auto-fill,minmax(16rem,1fr))]">
					{#each channels as ch (ch.ChannelId)}
						<UiCard title={ch.Label || ch.Name || `Channel ${ch.ChannelId}`}>
							{#snippet actions()}
								<button
									type="button"
									class="btn btn-sm preset-tonal"
									onclick={() => setLabel(Number(ch.ChannelId))}>Set label</button
								>
							{/snippet}
							{#if ch.HasTemps}
								<StatRow
									label={ch.ContainsPump ? 'Liquid' : 'Temperature'}
									value={ch.TemperatureString ?? (ch.Temperature != null ? `${ch.Temperature} °C` : '—')}
								/>
							{/if}
							<StatRow
								label="Speed"
								value={ch.Rpm != null ? `${ch.Rpm} RPM` : ch.SpeedString ?? '—'}
							/>
							<div class="pt-2 space-y-1">
								<label class="text-xs uppercase tracking-wide text-surface-600-400" for={`spd-${ch.ChannelId}`}
									>Profile</label
								>
								<select
									id={`spd-${ch.ChannelId}`}
									class="select w-full"
									disabled={busy}
									onchange={(e) =>
										setSpeed(Number(ch.ChannelId), (e.currentTarget as HTMLSelectElement).value)}
								>
									{#each tempOptions as [name]}
										<option value={name} selected={ch.Profile === name}>{name}</option>
									{/each}
								</select>
							</div>
						</UiCard>
					{/each}
				</div>
			</section>

			{#if rgbDevices.length}
				<section class="space-y-3">
					<h3 class="h4">Lighting</h3>
					<div class="grid gap-4 grid-cols-[repeat(auto-fill,minmax(16rem,1fr))]">
						{#each rgbDevices as rgb (rgb.ChannelId ?? rgb.Name)}
							<UiCard title={rgb.Name ?? `RGB ${rgb.ChannelId}`}>
								<div class="space-y-1">
									<label class="text-xs uppercase tracking-wide text-surface-600-400" for={`rgb-${rgb.ChannelId}`}
										>Mode</label
									>
									<select
										id={`rgb-${rgb.ChannelId}`}
										class="select w-full"
										disabled={busy}
										onchange={(e) =>
											setChannelRgb(
												Number(rgb.ChannelId),
												(e.currentTarget as HTMLSelectElement).value
											)}
									>
										{#each rgbModes as mode}
											<option value={mode} selected={rgb.Mode === mode || rgb.Profile === mode}
												>{mode}</option
											>
										{/each}
									</select>
								</div>
								<a class="anchor text-sm mt-3 inline-block" href={`/rgb?device=${serial}&channel=${rgb.ChannelId}`}
									>Configure</a
								>
							</UiCard>
						{/each}
					</div>
				</section>
			{/if}
		</div>
	</div>
</div>
