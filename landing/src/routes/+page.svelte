<script lang="ts">
	let copied = $state(false);
	let mode = $state<'simple' | 'demo' | 'customize'>('simple');
	let host = $state('');
	let agent = $state('');
	let client = $state('');
	let activeTab = $state('host');
	let origin = $state('nowbox.lol');

	$effect(() => {
		origin = window.location.host;
	});

	const hosts = [
		{ id: 'sprites', icon: `<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M7 1v12M1 7h12M3 3l8 8M11 3l-8 8"/></svg>` },
		{ id: 'modal', muted: true, icon: `<svg viewBox="0 0 24 24" fill="currentColor"><path d="M4.89 5.57 0 14l2.52 4.4h5.05l4.4-7.72 4.51 7.71 5 .04L24 14.06l-4.86-8.45-5.07-.02-2.08 3.6L9.94 5.57Zm.84.73h3.79l1.84 3.25H7.57Zm9.19.02 3.8.01 4.23 7.36-3.74-.03zm-9.82.35L6.94 9.91l-4.21 7.39-1.89-3.3Zm9.19.01 4.3 7.34-1.9 3.28-4.3-7.34zm-6.71 3.6h3.79l-4.21 7.39H3.36Zm11.64 4.11 3.74.03-1.89 3.28-3.74-.03z"/></svg>` },
		{ id: 'e2b', icon: `<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="2" y="2" width="10" height="10" rx="1"/><path d="M5 5h4M5 7h3M5 9h4"/></svg>` },
		{ id: 'daytona', icon: `<svg viewBox="0 0 275 287" fill="currentColor"><path d="M14.56 193.74H114.28V227.93H14.56z"/><path d="M148.46 74.08H262.43V108.27H148.46z"/><path d="M88.63 84.61l84.61-84.61 24.18 24.18-84.61 84.61z"/><path d="M89.16 170.08l-64.98-64.98-24.18 24.18 64.98 64.98z"/><path d="M174.63 217.91l-68.5 68.5-24.17-24.18 68.5-68.5z"/><path d="M174.11 132.44l76.55 76.55-24.17 24.18-76.55-76.55z"/><path d="M88.63 48.43v82.62H54.45V48.43z"/><path d="M208.29 168.09v102.57h-34.19V168.09z"/></svg>` },
		{ id: 'fly.io', icon: `<svg viewBox="0 0 24 24" fill="currentColor"><path d="M11.99 0c-2.45 0-5 .93-6.54 2.9-1.17 1.5-1.66 3.47-1.49 5.35.29 2.11 1.48 3.96 2.68 5.67a41.5 41.5 0 004.22 4.83c-1.06.83-1.94 2.29-1.36 3.64.82 2.32 4.67 2.05 5.12-.37.39-1.29-.69-2.53-1.43-3.31 2.39-2.43 4.71-5.04 6.17-8.15.6-1.32.9-2.8.61-4.24-.28-2.34-1.82-4.47-3.97-5.46A8.5 8.5 0 0011.99 0m-.24 1.58v15.53c-1.87-3.63-4.03-7.84-3.02-12.04.35-1.54 1.25-3.31 3.02-3.49m2 .04c1.53.36 3.03 1.1 3.9 2.48 1.3 1.93 1.32 4.55.1 6.52-1.27 2.4-3.06 4.46-4.92 6.42 1.47-2.97 3.07-6.11 3.18-9.5-.04-2.08-.44-4.61-2.27-5.92M11.97 20.1c.85.34 1.6 1.98.15 2.17-.66.15-1.37-.6-1-1.22.21-.36.49-.73.84-.95"/></svg>` },
		{ id: 'cloudflare', muted: true, icon: `<svg viewBox="0 0 24 24" fill="currentColor"><path d="M16.51 16.84c.15-.5.09-.97-.16-1.32-.22-.31-.6-.5-1.06-.52l-8.66-.11a.16.16 0 01-.13-.07c-.03-.04-.04-.1-.02-.16.03-.08.11-.15.2-.15l8.74-.11c1.03-.05 2.16-.89 2.55-1.91l.5-1.3c.02-.06.03-.11.01-.17-.56-2.55-2.84-4.45-5.55-4.45-2.5 0-4.63 1.62-5.39 3.86-.49-.37-1.12-.56-1.79-.5-1.2.12-2.17 1.08-2.29 2.29-.03.31 0 .61.06.89C1.57 13.17 0 14.78 0 16.75c0 .17.01.35.04.53.01.08.08.15.17.15h15.98c.09 0 .18-.06.2-.16l.12-.43zm2.76-5.56c-.08 0-.16 0-.24.01-.06 0-.1.04-.13.1l-.34 1.17c-.15.51-.09.97.15 1.32.23.32.61.5 1.06.52l1.84.11c.06 0 .11.03.13.07.03.04.04.11.02.16-.03.08-.11.15-.2.15l-1.92.11c-1.04.05-2.16.89-2.55 1.91l-.14.36c-.03.07.02.14.1.14h6.6c.08 0 .15-.05.17-.13.11-.41.18-.84.18-1.28 0-2.6-2.13-4.73-4.73-4.73"/></svg>` },
		{ id: 'docker', icon: `<svg viewBox="0 0 24 24" fill="currentColor"><path d="M13.98 11.08h2.12a.19.19 0 00.19-.19V9.01a.19.19 0 00-.19-.19h-2.12a.19.19 0 00-.18.19v1.89c0 .1.08.18.18.18m-2.95-5.43h2.12a.19.19 0 00.19-.19V3.57a.19.19 0 00-.19-.18h-2.12a.19.19 0 00-.18.18v1.89c0 .1.08.19.18.19m0 2.71h2.12a.19.19 0 00.19-.18V6.29a.19.19 0 00-.19-.19h-2.12a.18.18 0 00-.18.19v1.89c0 .1.08.18.18.18m-2.93 0h2.12a.19.19 0 00.18-.18V6.29a.19.19 0 00-.18-.19H8.1a.19.19 0 00-.19.19v1.89c0 .1.08.18.19.18m-2.96 0h2.12a.19.19 0 00.18-.18V6.29a.19.19 0 00-.18-.19H5.14a.19.19 0 00-.19.19v1.89c0 .1.08.18.19.18m5.89 2.72h2.12a.19.19 0 00.19-.19V9.01a.19.19 0 00-.19-.19h-2.12a.18.18 0 00-.18.19v1.89c0 .1.08.18.18.18m-2.93 0h2.12a.19.19 0 00.18-.19V9.01a.18.18 0 00-.18-.19H8.1a.19.19 0 00-.19.19v1.89c0 .1.08.18.19.18m-2.96 0h2.12a.19.19 0 00.18-.19V9.01a.19.19 0 00-.18-.19H5.14a.19.19 0 00-.19.19v1.89c0 .1.08.18.19.18m-2.92 0h2.12a.19.19 0 00.18-.19V9.01a.18.18 0 00-.18-.19H2.22a.19.19 0 00-.19.19v1.89c0 .1.08.18.19.18M23.76 9.89c-.07-.05-.67-.51-1.95-.51-.34 0-.68.03-1.01.09-.25-1.7-1.65-2.53-1.72-2.57l-.34-.2-.23.33c-.28.44-.49.92-.61 1.43-.23.97-.09 1.88.4 2.66-.6.33-1.55.41-1.74.42H.75a.75.75 0 00-.75.75c-.03 1.44.2 2.88.69 4.06.55 1.43 1.36 2.48 2.41 3.12 1.18.72 3.1 1.14 5.28 1.14.98 0 1.96-.09 2.93-.27a12.25 12.25 0 003.82-1.39c.98-.57 1.86-1.29 2.61-2.14 1.25-1.42 2-3 2.55-4.4h.22c1.37 0 2.22-.55 2.68-1.01.31-.29.55-.65.71-1.05l.1-.29z"/></svg>` },
		{ id: 'aws', muted: true, icon: `<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M2 8c2 2 6 2 8 0"/><path d="M7 8l3-1"/></svg>` },
		{ id: 'blaxel', icon: `<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.2"><circle cx="7" cy="7" r="5.5"/><path d="M2 5c3 1 7 1 10 0M2 9c3-1 7-1 10 0M5 2c-1 3-1 7 0 10M9 2c1 3 1 7 0 10"/></svg>` },
		{ id: 'runloop', icon: `<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M11 5a4 4 0 11-1-1"/><path d="M10 2v3h3"/></svg>` },
		{ id: 'vercel', icon: `<svg viewBox="0 0 24 24" fill="currentColor"><path d="m12 1.608 12 20.784H0Z"/></svg>` },
		{ id: 'codesandbox', muted: true, icon: `<svg viewBox="0 0 24 24" fill="currentColor"><path d="M0 24H24V0H0V2.45H21.55V21.55H2.45V0H0Z"/></svg>` },
		{ id: 'podman', icon: `<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M7 1l5 3v6l-5 3-5-3V4z"/><circle cx="7" cy="7" r="2"/></svg>` },
		{ id: 'apple', muted: true, icon: `<svg viewBox="0 0 24 24" fill="currentColor"><path d="M12.15 6.9c-.95 0-2.42-1.08-3.96-1.04-2.04.03-3.91 1.18-4.96 3.01-2.12 3.68-.55 9.1 1.52 12.09 1.01 1.45 2.21 3.09 3.79 3.04 1.52-.07 2.09-.99 3.94-.99 1.83 0 2.35.99 3.96.95 1.64-.03 2.68-1.48 3.68-2.95 1.15-1.69 1.64-3.32 1.66-3.41-.04-.01-3.18-1.22-3.22-4.86-.03-3.04 2.48-4.49 2.6-4.56-1.43-2.09-3.62-2.32-4.39-2.38-2-.15-3.68 1.09-4.61 1.09zM15.53 3.83c.84-1.01 1.4-2.43 1.25-3.83-1.21.05-2.66.81-3.53 1.82-.78.9-1.45 2.34-1.27 3.71 1.34.1 2.72-.69 3.56-1.7"/></svg>` },
	];

	const agents = [
		{ id: 'claude', icon: `<svg viewBox="0 0 24 24" fill="currentColor"><path d="M17.3 3.54h-3.67l6.7 16.92H24Zm-10.6 0L0 20.46h3.74l1.37-3.55h7l1.37 3.55h3.74L10.54 3.54Zm-.37 10.22 2.29-5.95 2.29 5.95Z"/></svg>` },
		{ id: 'codex', icon: `<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="7" cy="7" r="5"/><circle cx="7" cy="7" r="1.5"/></svg>` },
		{ id: 'aider', icon: `<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M5 2l-3 5 3 5M9 2l3 5-3 5"/></svg>` },
		{ id: 'openclaw', icon: `<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M4 3c-2 2-2 6 0 8M10 3c2 2 2 6 0 8M6 6l1 2 1-2"/></svg>` },
		{ id: 'hermes', icon: `<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M7 2v10M4 4c0-2 6-2 6 0M3 7h8"/></svg>` },
		{ id: 'opencode', icon: `<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M5 4L2 7l3 3M9 4l3 3-3 3"/></svg>` },
		{ id: 'goose', icon: `<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M4 11c0-3 1-5 3-7 1 0 2 1 2 2s-1 2-2 2c2 0 3 1 3 3"/></svg>` },
		{ id: 'cline', icon: `<svg viewBox="0 0 24 24" fill="currentColor"><path d="m23.37 13.56-1.44-2.9V8.99c0-2.76-2.22-5-4.95-5h-2.46c.18-.37.27-.78.27-1.21A2.77 2.77 0 0012.02 0a2.77 2.77 0 00-2.76 2.78c0 .43.1.85.27 1.21H7.07c-2.74 0-4.95 2.24-4.95 5v1.67L.64 13.55a.94.94 0 000 .93l1.47 2.85v1.67C2.11 21.76 4.33 24 7.07 24h9.9c2.74 0 4.95-2.24 4.95-5v-1.67l1.44-2.87a.89.89 0 000-.91m-12.85 2.36a2.27 2.27 0 01-2.26 2.27 2.27 2.27 0 01-2.26-2.27v-4.04A2.27 2.27 0 018.25 9.6a2.27 2.27 0 012.26 2.27zm7.29 0a2.27 2.27 0 01-2.26 2.27 2.27 2.27 0 01-2.26-2.27v-4.04a2.27 2.27 0 012.26-2.27 2.27 2.27 0 012.26 2.27z"/></svg>` },
	];

	const clients = [
		{ id: 'cli', icon: `<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M3 4l3 3-3 3M8 10h3"/></svg>` },
		{ id: 'web', icon: `<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="7" cy="7" r="5"/><path d="M2 7h10M7 2c-2 2-2 8 0 10M7 2c2 2 2 8 0 10"/></svg>` },
		{ id: 'mcp', muted: true, icon: `<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M3 3v8M11 3v8M3 7h8"/></svg>` },
	];

	function command() {
		if (mode === 'demo') return `curl -fsSL ${origin}/demo.now | sh`;
		if (mode === 'simple') return `curl -fsSL ${origin} | sh`;

		const parts: string[] = [];
		if (host) parts.push(host);
		if (agent) parts.push(agent);
		if (client && client !== 'cli') parts.push(client);
		if (parts.length === 0) return `curl -fsSL ${origin} | sh`;
		return `curl -fsSL ${origin} | sh -s -- ${parts.join(' ')}`;
	}

	function commandHTML() {
		if (mode === 'demo') {
			return `<span style="color:#555">curl -fsSL ${origin}/demo.now | sh</span>`;
		}

		let s = `<span style="color:#555">curl -fsSL ${origin} | sh</span>`;
		if (mode === 'simple') return s;

		if (host || agent || (client && client !== 'cli')) {
			s += `<span style="color:#555"> -s --</span>`;
		}
		if (host) s += ` <span style="color:#7dd3fc">${host}</span>`;
		if (agent) s += ` <span style="color:#a5f3a6">${agent}</span>`;
		if (client && client !== 'cli') s += ` <span style="color:#fbbf24">${client}</span>`;
		return s;
	}

	function copy() {
		navigator.clipboard.writeText(command());
		copied = true;
		setTimeout(() => copied = false, 2000);
	}

	function toggle(current: string, value: string): string {
		return current === value ? '' : value;
	}
</script>

<div class="page">
	<div class="content">
		<div class="logo-icon">
			<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z"/><path d="m3.3 7 8.7 5 8.7-5"/><path d="M12 22V12"/></svg>
		</div>
		<h1>nowbox</h1>
		<p class="sub">instant ai agent sandboxes <a href="https://github.com/Atro-Tech/nowbox#readme" class="learn-more">learn more &rarr;</a></p>

		<div class="builder">
			<div class="mode-row">
				<button class="mode-btn" class:active={mode === 'simple'} onclick={() => mode = 'simple'}>
					install
				</button>
				<button class="mode-btn" class:active={mode === 'demo'} onclick={() => mode = 'demo'}>
					demo
					<span class="mode-hint">use our keys</span>
				</button>
				<button class="mode-btn" class:active={mode === 'customize'} onclick={() => mode = 'customize'}>
					customize
				</button>
			</div>

			<div class="cmd-row">
				<div class="cmd">
					<span>{@html commandHTML()}</span>
				</div>
				<button class="copy-btn" onclick={copy}>
					{#if copied}copied{:else}<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="14" height="14"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>{/if}
				</button>
			</div>

			{#if mode === 'demo'}
				<div class="demo-info">
					<p>sprites + claude code — no keys needed</p>
				</div>
			{/if}

			{#if mode === 'customize'}
				<div class="tabs">
					<button class="tab-box" class:active={activeTab === 'host'} onclick={() => activeTab = 'host'}>
						<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5" width="14" height="14"><rect x="2" y="2" width="10" height="10" rx="1.5"/><path d="M2 6h10"/></svg>
						<span>host</span>{#if host}<span class="host-c">{host}</span>{/if}
					</button>
					<button class="tab-box" class:active={activeTab === 'agent'} onclick={() => activeTab = 'agent'}>
						<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5" width="14" height="14"><path d="M3 4l3 3-3 3M8 10h3"/></svg>
						<span>agent</span>{#if agent}<span class="agent-c">{agent}</span>{/if}
					</button>
					<button class="tab-box" class:active={activeTab === 'client'} onclick={() => activeTab = 'client'}>
						<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5" width="14" height="14"><circle cx="7" cy="7" r="5"/><path d="M2 7h10M7 2c-2 2-2 8 0 10M7 2c2 2 2 8 0 10"/></svg>
						<span>client</span>{#if client}<span class="client-c">{client}</span>{/if}
					</button>
				</div>

				<div class="pills-area">
					{#if activeTab === 'host'}
						<div class="pills">
							{#each hosts as h}
								<button class="chip" class:on={host === h.id} class:muted={h.muted} disabled={h.muted} onclick={() => host = toggle(host, h.id)}>
									<span class="chip-icon">{@html h.icon}</span>
									{h.id}
								</button>
							{/each}
						</div>
					{:else if activeTab === 'agent'}
						<div class="pills">
							{#each agents as a}
								<button class="chip" class:on={agent === a.id} class:muted={a.muted} disabled={a.muted} onclick={() => agent = toggle(agent, a.id)}>
									<span class="chip-icon">{@html a.icon}</span>
									{a.id}
								</button>
							{/each}
						</div>
					{:else}
						<div class="pills">
							{#each clients as c}
								<button class="chip" class:on={client === c.id} class:muted={c.muted} disabled={c.muted} onclick={() => client = toggle(client, c.id)}>
									<span class="chip-icon">{@html c.icon}</span>
									{c.id}
								</button>
							{/each}
						</div>
					{/if}
				</div>
			{/if}
		</div>
	</div>
</div>

<a href="https://github.com/Atro-Tech/nowbox" class="github-link" target="_blank" rel="noopener noreferrer" aria-label="View nowbox on GitHub">
	<svg viewBox="0 0 16 16" width="20" height="20" fill="currentColor"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0016 8c0-4.42-3.58-8-8-8z"/></svg>
</a>

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
		margin: 0 0 40px 0;
		font-size: 13px;
	}

	.builder {
		display: flex;
		flex-direction: column;
	}

	.mode-row {
		display: flex;
		gap: 0;
		margin-bottom: 16px;
	}

	.mode-btn {
		flex: 1;
		padding: 10px 8px;
		font-family: inherit;
		font-size: 13px;
		color: #555;
		background: transparent;
		border: 1px solid #222;
		border-right: none;
		cursor: pointer;
		transition: all 0.1s;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
	}

	.mode-btn:last-child {
		border-right: 1px solid #222;
	}

	.mode-btn:hover {
		color: #888;
		border-color: #444;
	}

	.mode-btn:hover + .mode-btn {
		border-left-color: #444;
	}

	.mode-btn.active {
		color: #eee;
		border-color: #eee;
	}

	.mode-btn.active + .mode-btn {
		border-left-color: #eee;
	}

	.mode-hint {
		font-size: 10px;
		color: #555;
		opacity: 0.7;
	}

	.mode-btn.active .mode-hint {
		color: #888;
	}

	.cmd-row {
		display: flex;
		align-items: stretch;
		gap: 8px;
		margin-bottom: 24px;
		margin-left: -40px;
		margin-right: -40px;
	}

	.cmd {
		flex: 1;
		padding: 16px 20px;
		background: #0a0a0a;
		border: 1px solid #222;
		font-size: 15px;
		white-space: nowrap;
		overflow-x: auto;
	}

	.copy-btn {
		padding: 0 14px;
		background: #0a0a0a;
		border: 1px solid #222;
		color: #333;
		cursor: pointer;
		font-family: inherit;
		font-size: 11px;
		display: flex;
		align-items: center;
	}

	.copy-btn:hover {
		color: #888;
		border-color: #444;
	}

	.copy-btn :global(svg) {
		display: block;
	}

	.demo-info {
		color: #555;
		font-size: 12px;
		text-align: center;
		margin-top: -16px;
	}

	.demo-info p {
		margin: 0;
	}

	.host-c { color: #7dd3fc; }
	.agent-c { color: #a5f3a6; }
	.client-c { color: #fbbf24; }

	.tabs {
		display: flex;
		gap: 6px;
		margin-bottom: 16px;
	}

	.tab-box {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		padding: 12px 8px;
		font-family: inherit;
		font-size: 13px;
		color: #444;
		background: transparent;
		border: 1px solid #222;
		cursor: pointer;
		transition: all 0.1s;
	}

	.tab-box:hover {
		color: #888;
		border-color: #444;
	}

	.tab-box.active {
		color: #eee;
		border-color: #eee;
	}

	.tab-box :global(svg) {
		flex-shrink: 0;
		opacity: 0.5;
	}

	.tab-box.active :global(svg) {
		opacity: 1;
	}

	.pills-area {
		padding: 8px 0;
		min-height: 100px;
		height: 100px;
	}

	.pills {
		display: flex;
		flex-wrap: wrap;
		gap: 10px;
	}

	.chip {
		font-family: inherit;
		font-size: 13px;
		padding: 6px 16px;
		background: transparent;
		color: #444;
		border: 1px solid transparent;
		cursor: pointer;
		transition: all 0.1s;
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.chip:hover {
		color: #888;
	}

	.chip.muted {
		opacity: 0.25;
		cursor: default;
	}

	.chip.muted:hover {
		color: #444;
	}

	.chip.on {
		border-color: #333;
		color: #eee;
	}

	.chip-icon {
		width: 14px;
		height: 14px;
		display: flex;
		align-items: center;
		justify-content: center;
		opacity: 0.5;
	}

	.chip.on .chip-icon {
		opacity: 1;
	}

	.chip-icon :global(svg) {
		width: 14px;
		height: 14px;
	}

	.learn-more {
		color: #666;
		font-size: 12px;
		text-decoration: none;
		margin-left: 6px;
	}

	.learn-more:hover {
		color: #999;
	}

	.github-link {
		position: fixed;
		bottom: 20px;
		right: 20px;
		color: #444;
		text-decoration: none;
		transition: color 0.15s;
	}

	.github-link:hover {
		color: #888;
	}
</style>
