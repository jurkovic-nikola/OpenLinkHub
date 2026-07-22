<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		actions,
		children,
		class: className = ''
	}: {
		title?: string;
		actions?: Snippet;
		children: Snippet;
		class?: string;
	} = $props();
</script>

<div class="card preset-outlined-surface-200-800 bg-surface-100-900 flex flex-col min-w-0 {className}">
	{#if title || actions}
		<header class="flex items-start justify-between gap-3 px-4 py-3 border-b border-surface-200-800">
			{#if title}
				<h3 class="font-semibold text-sm tracking-wide uppercase text-surface-600-400 break-words min-w-0">
					{title}
				</h3>
			{/if}
			{#if actions}
				<div class="shrink-0">{@render actions()}</div>
			{/if}
		</header>
	{/if}
	<div class="p-4 flex-1 min-w-0">
		{@render children()}
	</div>
</div>
