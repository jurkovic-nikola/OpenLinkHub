<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	onMount(() => goto('/rgb'));
</script>

<p class="opacity-60">Redirecting to RGB…</p>
