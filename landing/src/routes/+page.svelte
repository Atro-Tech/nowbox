<script lang="ts">
	let copied = $state(false);

	const command = 'curl nowbox.lol | sh';

	function copy() {
		navigator.clipboard.writeText(command);
		copied = true;
		setTimeout(() => copied = false, 2000);
	}
</script>

<div class="page">
	<div class="content">
		<h1>nowbox</h1>
		<p class="sub">instant ai agent sandboxes</p>

		<button class="box" onclick={copy}>
			<span class="prompt">$</span>
			<span class="cmd">{command}</span>
			<span class="copy">{copied ? 'copied' : 'copy'}</span>
		</button>
	</div>
</div>

<style>
	.page {
		min-height: 100vh;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.content {
		max-width: 480px;
		width: 100%;
		padding: 32px;
	}

	h1 {
		font-size: 24px;
		font-weight: bold;
		margin: 0 0 4px 0;
	}

	.sub {
		color: #888;
		margin: 0 0 32px 0;
		font-size: 13px;
	}

	.box {
		display: flex;
		align-items: center;
		gap: 12px;
		width: 100%;
		padding: 14px 16px;
		background: #0a0a0a;
		border: 1px solid #333;
		cursor: pointer;
		font-family: inherit;
		font-size: 14px;
		text-align: left;
	}

	.box:hover {
		border-color: #555;
	}

	.prompt {
		color: #555;
		user-select: none;
	}

	.cmd {
		color: #eee;
		flex: 1;
	}

	.copy {
		color: #555;
		font-size: 12px;
		user-select: none;
	}

	.box:hover .copy {
		color: #888;
	}
</style>
