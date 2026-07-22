<script lang="ts">
	import UiCard from '$lib/components/UiCard.svelte';
	import { getDashboard, postJson } from '$lib/api';
	import { onMount } from 'svelte';

	let settings = $state<Record<string, any>>({});
	let message = $state('');
	let lang = $state<Record<string, string>>({});

	onMount(async () => {
		const [dash, language] = await Promise.all([
			getDashboard(),
			fetch('/api/language/').then((r) => r.json())
		]);
		settings = (dash.dashboard as Record<string, any>) ?? (dash.data as Record<string, any>) ?? {};
		lang = (language.data as Record<string, string>) ?? {};
	});

	async function save() {
		await postJson('/api/dashboard/update', settings);
		message = 'Settings saved';
	}
</script>

<div class="space-y-6 max-w-3xl">
	<div>
		<h2 class="h2">Settings</h2>
		<p class="opacity-60">Dashboard and UI preferences</p>
	</div>
	{#if message}
		<p class="text-sm opacity-70">{message}</p>
	{/if}

	<UiCard title="Dashboard">
		<label class="flex items-center justify-between gap-3 py-2">
			<span>Show CPU</span>
			<input type="checkbox" class="checkbox" bind:checked={settings.showCpu} />
		</label>
		<label class="flex items-center justify-between gap-3 py-2">
			<span>Show GPU</span>
			<input type="checkbox" class="checkbox" bind:checked={settings.showGpu} />
		</label>
		<label class="flex items-center justify-between gap-3 py-2">
			<span>Show Disk</span>
			<input type="checkbox" class="checkbox" bind:checked={settings.showDisk} />
		</label>
		<label class="flex items-center justify-between gap-3 py-2">
			<span>Celsius</span>
			<input type="checkbox" class="checkbox" bind:checked={settings.celsius} />
		</label>
		<label class="flex items-center justify-between gap-3 py-2">
			<span>Temperature bar</span>
			<input type="checkbox" class="checkbox" bind:checked={settings.temperatureBar} />
		</label>
		<div class="py-2 space-y-1">
			<label class="text-xs uppercase tracking-wide opacity-60" for="pageTitle">Page title</label>
			<input id="pageTitle" class="input w-full" bind:value={settings.pageTitle} />
		</div>
		<div class="py-2 space-y-1">
			<label class="text-xs uppercase tracking-wide opacity-60" for="theme">Theme</label>
			<input id="theme" class="input w-full" bind:value={settings.theme} />
		</div>
		<button type="button" class="btn preset-filled-primary-500 mt-3" onclick={() => save()}>Save</button>
	</UiCard>

	{#if Object.keys(lang).length}
		<UiCard title="Language strings">
			<p class="text-sm opacity-70 mb-2">{Object.keys(lang).length} keys loaded from /api/language/</p>
			<details>
				<summary class="cursor-pointer text-sm">Preview</summary>
				<pre class="text-xs mt-2 overflow-auto max-h-64 opacity-70">{JSON.stringify(lang, null, 2)}</pre>
			</details>
		</UiCard>
	{/if}
</div>
