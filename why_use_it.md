# Why use nowbox?

## AI agents need sandboxes

AI coding agents are powerful but dangerous to run on your local machine. They install packages, modify files, run arbitrary commands. You want them in a sandbox — isolated, disposable, and far away from your actual work.

The problem is that setting up that sandbox is annoying. Every provider has its own SDK, its own auth flow, its own way of doing things. And then you still have to install and configure the agent on top of it.

## The current landscape is fragmented

Today, you either:

- **Use a sandbox provider** (E2B, Daytona, Vercel Sandbox) and wire up the agent yourself
- **Use an agent product** (Goose, OpenHands) that bundles its own runtime and locks you in
- **Use Docker locally** and hope the agent doesn't trash your machine anyway

Every option couples you to one provider or one agent. Want to switch from Claude to Codex? Different setup. Want to move from E2B to Fly.io? Start over.

## nowbox sits above all of that

nowbox is the layer that connects hosts and agents. You pick both independently.

```sh
nowbox sprites claude    # Claude on Sprites
nowbox sprites codex     # Codex on Sprites
nowbox vercel aider      # Aider on Vercel
nowbox docker goose      # Goose on Docker
```

Same CLI. Same workflow. Different combinations.

## Why this matters

**For individual developers:** Stop wasting time on sandbox setup. Try different agents without committing to a platform. Get a clean environment every time.

**For teams:** Standardize how your team runs AI agents. One tool, consistent behavior, no per-developer setup drift.

**For agent builders:** Ship a nowbox manifest instead of building your own sandbox infrastructure. Your agent becomes instantly runnable on every host nowbox supports.

**For sandbox providers:** Get discovered by users who are already running agents. A nowbox host manifest is free distribution.

## Open source and agnostic

nowbox is MIT licensed. It doesn't favor any host or any agent. The manifest format is simple and open — anyone can add support for a new provider or a new agent.

No vendor lock-in. No platform tax. Just a thin, portable CLI that connects the pieces.
