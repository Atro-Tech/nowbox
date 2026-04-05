# What is nowbox?

nowbox is a single command that spins up an AI coding agent inside an ephemeral cloud sandbox.

```sh
curl -fsSL nowbox.lol | sh -s -- sprites claude
```

That one line:

1. Provisions a fresh sandbox on a cloud provider (Sprites, Vercel, Daytona, Docker, etc.)
2. Installs the agent you picked (Claude, Codex, Aider, Cline, Goose, etc.)
3. Drops you into a terminal — or opens a browser UI — and you're coding

When you disconnect, the sandbox is destroyed. Nothing lingers.

## The shape

```
you
 └── nowbox
      ├── host (where it runs)
      │    ├── sprites
      │    ├── docker
      │    ├── vercel
      │    ├── daytona
      │    ├── fly.io
      │    └── ...
      ├── agent (what runs)
      │    ├── claude
      │    ├── codex
      │    ├── aider
      │    ├── goose
      │    └── ...
      └── client (how you interact)
           ├── cli
           ├── browser
           └── app
```

Pick a host. Pick an agent. Go.

## What it replaces

Without nowbox, setting up an AI agent in a sandbox means:

- Signing up for a sandbox provider
- Reading their SDK docs
- Writing boilerplate to provision, connect, and tear down
- Installing and configuring the agent manually
- Wiring up terminal I/O or a web UI

With nowbox, you skip all of that. One CLI, one manifest format, and you're running.

## How it works under the hood

nowbox uses a manifest system. Each host and each agent has a small manifest file that describes how to create, connect, and destroy. The CLI loads the right manifests, calls the provider API, streams I/O over WebSocket or HTTP, and cleans up on exit.

Manifests are fetched from GitHub and cached locally. You can also write your own.

## Current status

nowbox is early-stage and open source. Some host adapters are production-ready (Sprites), some are experimental, and some are placeholders for future work. See the [README](https://github.com/Atro-Tech/nowbox#readme) for the full support matrix.
