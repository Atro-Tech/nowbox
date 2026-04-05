<script lang="ts">
	let { children } = $props();
</script>

<svelte:head>
	<title>nowbox</title>
</svelte:head>

{@render children()}

<style>
	:global(body) {
		margin: 0;
		padding: 0;
		background: #111;
		color: #eee;
		font-family: 'SF Mono', 'Menlo', 'Monaco', 'Courier New', monospace;
		font-size: 14px;
	}
	:global(*) {
		box-sizing: border-box;
	}
</style>
