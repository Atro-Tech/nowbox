# nowbox

**Instant AI agent sandboxes.**

[![GitHub Release](https://img.shields.io/github/v/release/Atro-Tech/nowbox)](https://github.com/Atro-Tech/nowbox/releases)
[![License](https://img.shields.io/github/license/Atro-Tech/nowbox)](LICENSE)

nowbox is a thin Go CLI that provisions an ephemeral sandbox, installs an agent with manifest-defined commands, and drops you into either a native terminal or a browser terminal.

```sh
curl -fsSL nowbox.lol | sh -s -- sprites claude
```

The point is portability: one CLI, one manifest format, multiple sandbox providers and agents.

## Current Status

nowbox is early-stage. The repo already contains a broad host and agent catalog, but not every manifest is production-ready yet.

Today, the codebase is best described as:

- A working CLI/runtime for manifest-driven sandbox sessions
- A working landing site for `nowbox.lol`
- A support matrix with a mix of working adapters, experimental adapters, and placeholders

If you publish the repo as-is, set expectations accordingly: this is an ambitious prototype with a strong direction, not a finished cross-provider platform.

## Install

```sh
curl -fsSL nowbox.lol | sh
```

That downloads the matching release binary for your platform and caches it under `~/.cache/nowbox`.

You can also download binaries directly from [GitHub Releases](https://github.com/Atro-Tech/nowbox/releases).

## Usage

```sh
nowbox <host> <agent> [client]
```

Examples:

```sh
nowbox sprites claude
nowbox sprites codex
nowbox vercel codex
nowbox daytona aider web
```

If you omit the host or agent, nowbox will prompt you interactively.

### Client modes

| Mode | Status | Notes |
| --- | --- | --- |
| `cli` | supported | Native terminal proxy |
| `web` | supported | Browser terminal UI |
| `mcp` | not implemented | Listed in some UI/docs, but not available in the CLI yet |

### Flags

```text
--host,   -h    Host provider
--agent,  -a    Agent
--client, -c    Client mode (cli, web)
```

## What Actually Works Today

The codebase currently has two connection paths:

- `websocket_exec`: full interactive terminal streaming over WebSocket
- `http_exec`: request/response style execution over HTTP

That means host support is not all-or-nothing. Some manifests are materially usable, some are experimental, and some are clearly placeholders for future adapters.

### Host support

| Host | Status | Notes |
| --- | --- | --- |
| `sprites` | best-supported | Uses the WebSocket exec path and matches the current interactive terminal model best |
| `vercel` | experimental | Uses HTTP exec; manifest exists, but needs real-world validation as a terminal experience |
| `daytona` | experimental | Same as above |
| `runloop` | experimental | Same as above |
| `blaxel` | experimental | Same as above |
| `docker` | experimental | Manifest exists, but this is not a real Docker attach flow yet |
| `podman` | experimental | Same caveat as Docker |
| `e2b` | incomplete | Create/destroy manifest exists, connect path is empty |
| `fly.io` | incomplete | Needs SSH/WireGuard-style adapter |
| `aws` | placeholder | Needs provider-specific adapter/auth flow |
| `cloudflare` | placeholder | Needs custom adapter/runtime wrapper |
| `codesandbox` | placeholder | Needs SDK-backed adapter |
| `modal` | placeholder | Needs SDK-backed adapter |
| `apple` | placeholder | Needs native adapter |

### Agent support

The repo currently ships manifests for:

- `claude`
- `codex`
- `aider`
- `cline`
- `goose`
- `openclaw`
- `opencode`

Agent support here means nowbox knows how to run the setup commands. It does **not** mean every agent/host combination has been validated end-to-end.

## How It Works

1. nowbox loads a host manifest and an agent manifest from GitHub or local cache.
2. The selected adapter creates the sandbox through the provider API.
3. nowbox connects to the sandbox stream.
4. Agent setup commands are sent into the sandbox.
5. You interact through the CLI terminal or the browser terminal.
6. On disconnect, nowbox destroys the sandbox and stores recovery metadata if cleanup needs to be retried.

## Why This Is Interesting

The strongest part of the project is the product shape:

- one install command
- one CLI
- one manifest format
- many possible hosts
- many possible agents

That is a good abstraction boundary. Most competing products are either:

- a sandbox provider with their own SDK and runtime model, or
- a single agent with an opinionated runtime

nowbox sits one layer above that and aims to unify them.

## Comparison

These are the closest reference points for positioning nowbox today:

| Project | What it is | Where nowbox is stronger | Where nowbox is weaker |
| --- | --- | --- | --- |
| [Vercel Sandbox](https://vercel.com/docs/vercel-sandbox/reference/readme) | Hosted ephemeral sandbox product | More provider-agnostic vision | Far less mature runtime and docs |
| [E2B](https://e2b.dev/docs) | SDK/platform for isolated agent sandboxes | Simpler CLI-first UX concept | E2B is much more complete and battle-tested |
| [Daytona](https://www.daytona.io/) | Dev environment/sandbox platform | Lighter abstraction layer across hosts | Daytona has a stronger platform and operational story |
| [OpenHands](https://github.com/All-Hands-AI/OpenHands) | Full autonomous coding agent system | Cleaner "bring your own host + bring your own agent" framing | OpenHands is far more complete as an end-user agent product |
| [Goose](https://github.com/block/goose) | Agent/CLI product | nowbox focuses on runtime portability, not just the agent | Goose has much stronger agent polish, docs, and ecosystem traction |

The benchmark takeaway:

- The idea is strong.
- The current implementation is still much earlier than the best-known projects in this space.
- The clearest differentiator is "unified launcher for many hosts and many agents."

## README-Level Risks To Fix Before Pushing

These were the main documentation problems in the previous version of this README:

- It implied a broader level of host support than the code currently provides.
- It showed examples like `mcp` as if they were usable.
- It mixed implemented features with placeholder architecture.
- It undersold the fact that many manifests are speculative and adapter-dependent.

Those are now corrected here.

## Manifest model

Manifest resolution currently works like this:

1. local manifest path
2. cached manifest in `~/.nowbox`
3. fetch from the `Atro-Tech/nowbox` GitHub repo
4. fetch from an explicit URL

That gives nowbox a useful distribution model, but it also means manifests are part of the runtime trust boundary. The CLI is not purely offline unless the needed manifests are already cached locally.

## Developing

### Go CLI

```sh
cd nowbox
go build .
go test ./...
```

Use the Go version pinned in [`nowbox/go.mod`](./nowbox/go.mod).

### Landing site

```sh
cd landing
npm install
npm run dev
```

For Vercel deployment, this repo is configured to use `@sveltejs/adapter-vercel`.

## Project Structure

| Directory | Purpose |
| --- | --- |
| `nowbox/` | Go CLI/runtime |
| `landing/` | SvelteKit landing site |

## Contributing

The highest-leverage contributions right now are:

- turning experimental hosts into validated hosts
- replacing placeholder manifests with real adapters
- adding end-to-end tests around create/connect/destroy flows
- tightening provider-specific docs and setup instructions

## Links

- [Website](https://nowbox.lol)
- [GitHub repo](https://github.com/Atro-Tech/nowbox)
