<script lang="ts">
	let {
		label,
		value,
		hint
	}: {
		label: string;
		value?: string | number | null;
		hint?: string;
	} = $props();
</script>

<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-1 sm:gap-3 py-2 border-b border-surface-200-800/60 last:border-0">
	<span class="text-xs uppercase tracking-wide text-surface-600-400 shrink-0">{label}</span>
	<div class="min-w-0 text-sm font-medium break-words sm:text-right">
		{#if value != null && value !== ''}
			<span>{value}</span>
		{/if}
		{#if hint}
			<span class="opacity-60 ml-1">{hint}</span>
		{/if}
	</div>
</div>
