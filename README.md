# nowbox

**nowbox is a tool to simplify agent sandboxes.**

It gives you one consistent way to:

- boot an agent inside a sandbox
- connect to it immediately
- save that setup as a reusable `.now` launcher
- reopen it later
- share that launcher when you want someone else to use the same setup
- switch between interfaces: CLI, browser, native app, or MCP

The core idea is simple:

1. pick a sandbox host
2. pick an agent
3. boot fast
4. connect directly from your device to the sandbox
5. throw the session away, or save the launcher for later

[![GitHub Release](https://img.shields.io/github/v/release/Atro-Tech/nowbox)](https://github.com/Atro-Tech/nowbox/releases)
[![License](https://img.shields.io/github/license/Atro-Tech/nowbox)](LICENSE)

## What nowbox is

nowbox is not trying to be a big hosted control plane.

It is a thin bootstrap and connection layer for agent sandboxes.

The goal is:

- make sandboxes easy to start
- make agent setup easy to repeat
- keep the path from your machine to the sandbox as direct as possible

That is the p2p-ish model behind the project:

- your device talks to the sandbox
- nowbox helps create and connect the session
- the website only helps you get started

## The Simplest Way To Use It

### Demo script

```sh
curl -fsSL nowbox.lol/demo.now | sh
```

This is the fastest way to understand what nowbox feels like.

What `demo.now` is:

- a saved nowbox launcher
- a small shell script
- a preconfigured setup for **Sprites + Claude Code**
- backed by a token already embedded into the script

So instead of setting up your own host account first, you just run it and land in a working demo flow.

Important:

- this uses **our demo Sprites instance**
- the script includes access for demo purposes
- do not treat it as private or production-safe
- do not put secrets or sensitive work in it

Use it to see the flow. Use your own host setup for real work.

## The Next Step: Your Own Custom Launch

If you want your own sandbox instead of the demo, use the normal bootstrap command:

```sh
curl -fsSL nowbox.lol | sh -s -- <host> <agent> [interface]
```

Examples:

```sh
curl -fsSL nowbox.lol | sh -s -- sprites claude
curl -fsSL nowbox.lol | sh -s -- sprites codex
curl -fsSL nowbox.lol | sh -s -- daytona codex
curl -fsSL nowbox.lol | sh -s -- daytona codex browser
curl -fsSL nowbox.lol | sh -s -- vercel aider
```

What this does:

- downloads `nowbox` if needed
- prompts for any required host credentials
- creates the sandbox
- runs the agent setup
- connects you to it

If you omit the interface, nowbox uses the CLI by default.

## Installed Syntax

Once you have the binary, the main command is:

```sh
nowbox <host> <agent> [interface]
```

Examples:

```sh
nowbox sprites claude
nowbox sprites codex
nowbox daytona codex browser
nowbox sprites claude mcp
```

There is also a create mode:

```sh
nowbox create <host> <agent>
```

That writes a reusable `.now` launcher up front without opening a session first.

## Ephemeral Sessions

The default nowbox mental model is ephemeral sessions.

That means:

- you boot a sandbox
- you connect to it
- you do the work
- nowbox tears it down when the session ends

This is the default because the product is optimized for fast bootstrap, not long-lived environment management.

## `.now` Files

`.now` files are saved nowbox launchers.

They are the bridge between:

- a one-off ephemeral session
- and a reusable template you can run again later

### Create a `.now` file up front

```sh
nowbox create sprites claude
```

That creates a file with a generated name like:

```text
brave-owl-4821.now
```

Then you can run it with:

```sh
sh brave-owl-4821.now
```

### Save a `.now` file from a running session

You can also save from inside the session UI:

- CLI: use the save shortcut shown in the toolbar
- browser: click `save`
- native app: click `save`

### What is inside a `.now` file

A `.now` file is a shell launcher that:

- stores an encrypted `NOWBOX_TOKEN`
- tries the cached `nowbox` binary first
- falls back to downloading `nowbox` if needed
- relaunches the saved setup

### Reusing `.now` files

This is the easiest way to keep preferred setups around.

Examples:

- one `.now` for Sprites + Claude Code
- one `.now` for Daytona + Codex
- one `.now` for a browser-first workflow

### Sharing `.now` files

You can share a `.now` file if you want someone else to boot the same saved setup.

That makes `.now` files useful for:

- demos
- onboarding
- repeatable templates
- handing someone a working launch path instead of a setup guide

Treat them carefully:

- a `.now` file can carry access through its embedded token
- it is not just a harmless config file
- only share it intentionally

This is not yet a full hosted multiplayer session system. Richer hosted sharing is still future work.

## Cached Bootstrap vs Install

If you run:

```sh
curl -fsSL nowbox.lol | sh
```

nowbox downloads the binary into its cache and runs it from there.

The cache path is:

```text
~/.cache/nowbox/nowbox
```

That is enough for repeated bootstrap use, but it does not necessarily put `nowbox` on your normal shell `PATH`.

## Install For Persistent Use

If you want a more permanent install:

```sh
curl -fsSL nowbox.lol | sh -s -- install
```

What that does depends on the platform:

- macOS: installs a local `nowbox.app` and also tries to place the CLI on your `PATH`
- Linux: installs the binary into `/usr/local/bin` or `~/.local/bin`

After that, you can call it directly like this:

```sh
nowbox sprites claude
```

You can also download binaries from [GitHub Releases](https://github.com/Atro-Tech/nowbox/releases).

## Interfaces

nowbox supports multiple ways to interact with the same session.

Syntax:

```sh
nowbox <host> <agent> [interface]
```

### CLI

```sh
nowbox sprites claude
```

This is the default interface.

Use it when you want:

- the fastest path
- a terminal-native workflow
- no extra mode flag

### Browser

```sh
nowbox sprites claude browser
```

This opens a local browser-based terminal UI.

Use it when you want:

- a local web view
- a save button for `.now` files
- a lightweight UI without the native app build path

### Native app

```sh
nowbox sprites claude app
```

This opens a local native window backed by a webview.

Use it when you want:

- a desktop-style interface
- a local app window instead of a browser tab
- the app view for the running session

Note:

- app mode requires CGO / webview support
- if your build does not support it, use `cli` or `browser`

### MCP

```sh
nowbox sprites claude mcp
```

This starts a local MCP server for the running sandboxed agent.

That means another agent or tool can talk to the session through MCP instead of you manually driving the terminal.

This is the bridge for p2p-ish agent-to-agent workflows:

- you start the sandbox once
- nowbox exposes it locally over MCP
- another tool connects to that MCP endpoint

## Two Concrete Usage Patterns

### 1. The simplest possible path

Use the demo launcher:

```sh
curl -fsSL nowbox.lol/demo.now | sh
```

This is a saved template using our **Sprites** instance and **Claude Code**.

Its job is to make the first-run experience dead simple.

### 2. A custom path on your own host

Example:

```sh
curl -fsSL nowbox.lol | sh -s -- daytona codex
```

That means:

- use Daytona as the sandbox host
- use Codex as the agent
- boot the environment
- connect immediately

That is the normal nowbox workflow.

## What `nowbox.lol` Actually Does

`nowbox.lol` is intentionally small.

It mainly hosts:

- the bootstrap installer script
- the `demo.now` launcher
- the landing page

That is by design.

The long-term point of nowbox is not to keep routing everything through a big middle layer. It is to bootstrap quickly, then let your machine talk to the sandbox as directly as possible.

If you want to inspect what is being run, you can audit:

- [`landing/static/bootstrap.sh`](/Users/k/Projects/nowbox/landing/static/bootstrap.sh)
- [`landing/static/demo.now`](/Users/k/Projects/nowbox/landing/static/demo.now)

## Security And Trust Notes

There are two very different trust models here.

### Demo mode

```sh
curl -fsSL nowbox.lol/demo.now | sh
```

This is convenience-first.

- it uses the nowbox demo instance
- it includes demo access in the launcher
- it is for understanding the flow, not for sensitive work

### Your own setup

```sh
curl -fsSL nowbox.lol | sh -s -- <host> <agent>
```

This is the real path for normal usage.

You bring your own host account and credentials, and nowbox helps you launch and connect.

Also note:

- `.now` files contain reusable launch material, so handle them carefully
- the website only hosts the startup assets
- the product is currently optimized for speed and simplicity over heavy managed infrastructure

## Current Status

nowbox is still early.

The strongest path today is the fast bootstrap flow around a small set of host and agent combinations. The bigger vision is wider than what the current repo has fully proven.

The repo currently includes host support work for providers like:

- `sprites`
- `vercel`
- `daytona`
- `runloop`
- `blaxel`
- `docker`
- `podman`

And agent launch manifests for:

- `claude`
- `codex`
- `aider`
- `cline`
- `goose`
- `openclaw`
- `opencode`

Support in the repo means nowbox knows how to launch those combinations. It does not mean every host/agent pairing has been validated equally.

## What Is Coming

The direction from here is broader than the current repo.

Coming soon:

- fully managed and hosted keys
- hosted nowbox sessions
- richer session reuse
- better sharing flows
- more cloud-managed workflows

The idea is:

- start with direct, fast, local-to-sandbox usage
- keep the bootstrap extremely light
- add richer hosted workflows when they are actually useful

## Build From Source

### Go CLI

```sh
cd nowbox
go build .
go test ./...
```

### Landing site

```sh
cd landing
npm install
npm run dev
```

## Project Structure

| Directory | Purpose |
| --- | --- |
| `nowbox/` | Go CLI and runtime |
| `landing/` | landing site and bootstrap assets |

## Links

- [Website](https://nowbox.lol)
- [GitHub Releases](https://github.com/Atro-Tech/nowbox/releases)
