<script lang="ts">
	let email = $state('');
	let submitted = $state(false);
	let submitting = $state(false);

	async function submit(e: Event) {
		e.preventDefault();
		if (!email || submitting) return;
		submitting = true;

		// TODO: wire up to actual backend/form service
		await new Promise(r => setTimeout(r, 600));

		submitted = true;
		submitting = false;
	}
</script>

<div class="page">
	<div class="content">
		<a href="/" class="back">&larr; back</a>

		<div class="logo-icon">
			<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z"/><path d="m3.3 7 8.7 5 8.7-5"/><path d="M12 22V12"/></svg>
		</div>
		<h1>early access</h1>
		<p class="sub">nowbox cloud is coming soon. get notified when it's ready.</p>

		{#if submitted}
			<div class="success">
				<p>you're on the list.</p>
				<p class="muted">we'll reach out when it's time.</p>
			</div>
		{:else}
			<form onsubmit={submit}>
				<input
					type="email"
					bind:value={email}
					placeholder="you@email.com"
					required
					disabled={submitting}
				/>
				<button type="submit" disabled={submitting || !email}>
					{submitting ? '...' : 'join waitlist'}
				</button>
			</form>
		{/if}

		<div class="details">
			<p>what you'll get:</p>
			<div class="list">
				<div class="item"><span class="tree">├──</span> managed sandbox hosting</div>
				<div class="item"><span class="tree">├──</span> one-click agent launches from the browser</div>
				<div class="item"><span class="tree">├──</span> team sharing and session history</div>
				<div class="item"><span class="tree">└──</span> same open source CLI underneath</div>
			</div>
		</div>
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
		max-width: 540px;
		width: 100%;
		padding: 32px;
	}

	.back {
		display: inline-block;
		color: #555;
		text-decoration: none;
		font-size: 13px;
		margin-bottom: 32px;
	}

	.back:hover {
		color: #aaa;
	}

	.logo-icon {
		color: #666;
		margin-bottom: 12px;
	}

	h1 {
		font-size: 24px;
		font-weight: bold;
		margin: 0 0 4px 0;
	}

	.sub {
		color: #666;
		margin: 0 0 32px 0;
		font-size: 13px;
	}

	form {
		display: flex;
		gap: 8px;
		margin-bottom: 40px;
	}

	input {
		flex: 1;
		padding: 12px 14px;
		background: #0a0a0a;
		border: 1px solid #333;
		color: #eee;
		font-family: inherit;
		font-size: 14px;
		outline: none;
	}

	input:focus {
		border-color: #555;
	}

	input::placeholder {
		color: #444;
	}

	input:disabled {
		opacity: 0.5;
	}

	button {
		padding: 12px 20px;
		background: transparent;
		border: 1px solid #e87a1e;
		color: #e87a1e;
		font-family: inherit;
		font-size: 13px;
		font-weight: 600;
		cursor: pointer;
		white-space: nowrap;
		transition: all 0.15s;
	}

	button:hover {
		background: #e87a1e;
		color: #fff;
	}

	button:disabled {
		opacity: 0.4;
		cursor: default;
	}

	button:disabled:hover {
		background: transparent;
		color: #e87a1e;
	}

	.success {
		margin-bottom: 40px;
		padding: 16px;
		border: 1px solid #333;
	}

	.success p {
		margin: 0;
		font-size: 14px;
	}

	.success .muted {
		color: #555;
		font-size: 12px;
		margin-top: 4px;
	}

	.details {
		color: #555;
		font-size: 13px;
	}

	.details p {
		margin: 0 0 8px 0;
	}

	.list {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.item {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 13px;
		color: #666;
	}

	.tree {
		color: #333;
		font-size: 12px;
	}

	@media (max-width: 600px) {
		.content {
			padding: 24px 16px;
		}

		form {
			flex-direction: column;
		}
	}
</style>
