<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	let { children } = $props();
</script>

<svelte:head>
	<title>nowbox — instant ai agent sandboxes</title>
	<meta name="description" content="One command to spin up a cloud sandbox with any AI coding agent. Pick a host, pick an agent, go." />
	<meta name="theme-color" content="#111111" />

	<meta property="og:type" content="website" />
	<meta property="og:title" content="nowbox" />
	<meta property="og:description" content="Instant AI agent sandboxes. One command." />
	<meta property="og:url" content="https://nowbox.lol" />
	<meta property="og:site_name" content="nowbox" />

	<meta name="twitter:card" content="summary" />
	<meta name="twitter:title" content="nowbox" />
	<meta name="twitter:description" content="Instant AI agent sandboxes. One command." />

	<link rel="icon" href={favicon} />
	<link rel="canonical" href="https://nowbox.lol" />
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
