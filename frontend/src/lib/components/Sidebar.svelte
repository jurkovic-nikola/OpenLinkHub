<script lang="ts">
	import { page } from '$app/stores';
	import type { DeviceSummary } from '$lib/api';

	let {
		devices = {},
		collapsed = false
	}: {
		devices?: Record<string, DeviceSummary>;
		collapsed?: boolean;
	} = $props();

	const nav = [
		{ href: '/', label: 'Dashboard' },
		{ href: '/temperature', label: 'Temperatures' },
		{ href: '/rgb', label: 'RGB' },
		{ href: '/rgbCluster', label: 'RGB Cluster' },
		{ href: '/macros', label: 'Macros' },
		{ href: '/lcd', label: 'LCD' },
		{ href: '/settings', label: 'Settings' }
	];

	function active(href: string, pathname: string) {
		if (href === '/') return pathname === '/';
		return pathname === href || pathname.startsWith(href + '/');
	}
</script>

<aside
	class="bg-surface-100-900 border-r border-surface-200-800 flex flex-col h-full overflow-y-auto transition-all {collapsed
		? 'w-16'
		: 'w-64'}"
>
	<div class="p-4 border-b border-surface-200-800">
		{#if !collapsed}
			<p class="text-xs uppercase tracking-widest opacity-60">OpenLinkHub</p>
			<h1 class="h4 mt-1">WebUI</h1>
		{:else}
			<p class="text-center font-bold">OLH</p>
		{/if}
	</div>

	<nav class="p-2 space-y-1">
		{#each nav as item}
			<a
				href={item.href}
				class="block rounded-base px-3 py-2 text-sm transition-colors {active(item.href, $page.url.pathname)
					? 'preset-tonal-primary'
					: 'hover:preset-tonal'}"
				title={item.label}
			>
				{#if collapsed}
					<span class="block text-center text-xs">{item.label.slice(0, 1)}</span>
				{:else}
					{item.label}
				{/if}
			</a>
		{/each}
	</nav>

	{#if !collapsed}
		<div class="mt-4 px-2 pb-4">
			<p class="px-3 text-xs uppercase tracking-wide opacity-50 mb-2">Devices</p>
			<div class="space-y-1">
				{#each Object.values(devices) as device}
					{#if device.Serial && !device.Hidden}
						<a
							href={`/device/${device.Serial}`}
							class="block rounded-base px-3 py-2 text-sm transition-colors { $page.url.pathname.includes(device.Serial)
								? 'preset-tonal-primary'
								: 'hover:preset-tonal'}"
						>
							<span class="break-words">{device.Product ?? device.Serial}</span>
						</a>
					{/if}
				{/each}
			</div>
		</div>
	{/if}
</aside>
