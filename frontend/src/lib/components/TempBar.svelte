<script lang="ts">
	let {
		cpu = '—',
		storage = [] as { Model?: string; TemperatureString?: string; Temperature?: number }[]
	}: {
		cpu?: string;
		storage?: { Model?: string; TemperatureString?: string; Temperature?: number }[];
	} = $props();
</script>

<div class="grid gap-3 grid-cols-[repeat(auto-fill,minmax(14rem,1fr))] mb-4">
	<div class="card preset-outlined-surface-200-800 p-4 min-w-0">
		<p class="text-xs uppercase tracking-wide opacity-60">CPU Temperature</p>
		<p class="text-2xl font-semibold text-warning-500 mt-1 break-words">{cpu}</p>
	</div>
	{#each storage as disk}
		<div class="card preset-outlined-surface-200-800 p-4 min-w-0">
			<p class="text-xs uppercase tracking-wide opacity-60">Storage Temperature</p>
			<p class="text-2xl font-semibold text-warning-500 mt-1">
				{disk.TemperatureString ?? (disk.Temperature != null ? `${disk.Temperature} °C` : '—')}
			</p>
			<p class="text-xs opacity-60 mt-1 break-words">{disk.Model ?? ''}</p>
		</div>
	{/each}
</div>
