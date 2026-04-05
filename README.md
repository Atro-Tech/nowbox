# nowbox

**Instant AI agent sandboxes.**

[![GitHub Release](https://img.shields.io/github/v/release/Atro-Tech/nowbox)](https://github.com/Atro-Tech/nowbox/releases)
[![License](https://img.shields.io/github/license/Atro-Tech/nowbox)](LICENSE)

AI coding agents need sandboxes. Setting them up is slow, fragile, and different for every provider. nowbox makes it one command: pick a host, pick an agent, and you're in.

```
curl -fsSL nowbox.lol | sh -s -- sprites claude
```

A cloud sandbox spins up on [Sprites](https://sprites.dev), Claude Code gets installed, and you're connected via a native terminal. When you close, the sandbox is destroyed. Nothing lingers.

---

## Install

```sh
curl -fsSL nowbox.lol | sh
```

Downloads the right binary for your platform (macOS, Linux, Windows) and caches it at `~/.cache/nowbox`. That's it.

Or grab a binary directly from [GitHub Releases](https://github.com/Atro-Tech/nowbox/releases).

## Usage

```sh
nowbox <host> <agent> [client]
```

### Examples

```sh
nowbox sprites claude          # Claude Code on Sprites
nowbox sprites codex           # OpenAI Codex on Sprites
nowbox sprites aider web       # Aider in browser UI
nowbox docker claude           # Claude Code locally via Docker
```

If you omit the host or agent, nowbox will prompt you to pick one interactively.

### Client modes

| Mode  | Description                  |
|-------|------------------------------|
| `cli` | Native terminal (default)    |
| `web` | Browser-based terminal UI    |
| `mcp` | MCP server (coming soon)     |

### Flags

```
--host,   -h    Host provider
--agent,  -a    Agent
--client, -c    Client mode (cli, web, mcp)
```

## How it works

1. A single Go binary handles everything
2. It provisions a sandbox on your chosen host via the host's API
3. Installs the agent via manifest-defined setup commands
4. Connects you directly (WebSocket or WebRTC) -- nowbox gets out of the way after setup
5. When you disconnect, the sandbox is destroyed (ephemeral by default)

If nowbox crashes, orphaned sandboxes are detected and cleaned up on next run.

## Architecture

```
nowbox/
  main.go                          CLI entrypoint
  internal/
    manifest/                      TOML manifest loader
      packages/
        hosts/                     Host provider configs
          sprites/host.toml
          docker/host.toml
          fly/host.toml
          e2b/host.toml
          modal/host.toml
          ...
        agents/                    Agent configs
          claude/agent.toml
          codex/agent.toml
          aider/agent.toml
          goose/agent.toml
          cline/agent.toml
          ...
    session/                       Session lifecycle (create, connect, destroy)
    adapter/                       Connection adapters (WebSocket, WebRTC)
    terminal/                      TUI terminal proxy
    webui/                         Browser terminal UI
    names/                         Random session name generator

landing/                           SvelteKit landing page (nowbox.lol)
```

### Manifests

Host and agent configurations are TOML files embedded in the binary at compile time. Adding a new host or agent is just adding a `.toml` file.

**Host manifest** (`host.toml`) defines how to create, connect to, and destroy a sandbox:

```toml
name = "sprites"
description = "Fly.io sandbox"
adapter = "websocket_exec"

[keys]
required = ["SPRITES_API_KEY"]
prompt = "sprites api key"

[create]
method = "POST"
url = "https://api.sprites.dev/v1/sprites"
headers = { Authorization = "Bearer ${SPRITES_API_KEY}" }
body = '{"name":"${SESSION_NAME}"}'
parse_id = ".name"

[connect]
url = "wss://api.sprites.dev/v1/sprites/${INSTANCE_ID}/exec"
headers = { Authorization = "Bearer ${SPRITES_API_KEY}" }

[destroy]
method = "DELETE"
url = "https://api.sprites.dev/v1/sprites/${INSTANCE_ID}"
headers = { Authorization = "Bearer ${SPRITES_API_KEY}" }
```

**Agent manifest** (`agent.toml`) defines how to install and launch an agent:

```toml
name = "claude"
description = "Claude Code (Anthropic)"

[setup]
commands = [
  "which claude || npm install -g @anthropic-ai/claude-code",
  "claude"
]

[keys]
optional = ["ANTHROPIC_API_KEY"]
self_auth = true
```

## Adding a new host

1. Create `nowbox/internal/manifest/packages/hosts/<name>/host.toml`
2. Define the `[create]`, `[connect]`, and `[destroy]` sections
3. Build. The manifest is embedded automatically.

## Adding a new agent

1. Create `nowbox/internal/manifest/packages/agents/<name>/agent.toml`
2. Define `[setup].commands` -- the shell commands to install and launch the agent
3. Build.

## Building from source

```sh
cd nowbox
go build -o nowbox .
```

Requires Go 1.21+.

## Project structure

| Directory  | What it is                          |
|------------|-------------------------------------|
| `nowbox/`  | Go binary -- the product            |
| `landing/` | SvelteKit landing page (nowbox.lol) |

## Contributing

Contributions welcome. The highest-impact contributions right now:

- **New host manifests** -- add support for more sandbox providers
- **New agent manifests** -- add support for more AI coding agents
- **Bug fixes and testing** -- especially around session lifecycle and cleanup

Open an issue first for anything large. For new hosts/agents, just open a PR with the TOML file.

## Links

- **Website:** [nowbox.lol](https://nowbox.lol)
- **Repository:** [github.com/Atro-Tech/nowbox](https://github.com/Atro-Tech/nowbox)
