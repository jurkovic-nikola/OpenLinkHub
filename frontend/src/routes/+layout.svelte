<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import TempBar from '$lib/components/TempBar.svelte';
	import { getDevices, type DeviceSummary } from '$lib/api';
	import { ws, type WsServerMessage } from '$lib/ws';
	import { onMount } from 'svelte';

	let { children } = $props();

	let devices = $state<Record<string, DeviceSummary>>({});
	let cpuTemp = $state('—');
	let storage = $state<{ Model?: string; TemperatureString?: string; Temperature?: number }[]>([]);
	let sidebarCollapsed = $state(false);

	onMount(() => {
		getDevices()
			.then((res) => {
				if (res.device && typeof res.device === 'object') {
					devices = res.device as Record<string, DeviceSummary>;
				}
			})
			.catch(() => {});

		ws.connect();
		ws.subscribe('dashboard');
		ws.subscribe('temps');

		const off = ws.onMessage((msg: WsServerMessage) => {
			if (msg.type === 'devices' && msg.data && typeof msg.data === 'object') {
				devices = msg.data as Record<string, DeviceSummary>;
			}
			if (msg.type === 'temps' && msg.data && typeof msg.data === 'object') {
				const d = msg.data as {
					cpu?: string;
					storage?: { Model?: string; TemperatureString?: string; Temperature?: number }[];
				};
				if (d.cpu) cpuTemp = d.cpu;
				if (Array.isArray(d.storage)) storage = d.storage;
			}
		});

		return () => {
			off();
			ws.unsubscribe('dashboard');
			ws.unsubscribe('temps');
		};
	});
</script>

<svelte:head>
	<title>OpenLinkHub</title>
	<link rel="icon" href={favicon} />
</svelte:head>

<div class="flex h-screen min-h-0 bg-surface-50-950 text-surface-950-50">
	<Sidebar {devices} collapsed={sidebarCollapsed} />
	<div class="flex-1 min-w-0 flex flex-col overflow-hidden">
		<header class="flex items-center gap-3 px-4 py-2 border-b border-surface-200-800 shrink-0">
			<button
				type="button"
				class="btn preset-tonal btn-sm"
				onclick={() => (sidebarCollapsed = !sidebarCollapsed)}
			>
				Menu
			</button>
			<span class="text-sm opacity-70 truncate">OpenLinkHub WebUI</span>
		</header>
		<main class="flex-1 overflow-y-auto p-4 md:p-6">
			<TempBar cpu={cpuTemp} {storage} />
			{@render children()}
		</main>
	</div>
</div>
