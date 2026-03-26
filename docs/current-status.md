# PopiArt CLI Current Status

Date: `2026-03-26`

This document summarizes the current repository-local status of `popiartcli` after the first MCP discoverability and runtime-baseline implementation pass.

It is intentionally different from the design docs:

- [docs/project-relationship.md](./project-relationship.md) defines ownership boundaries
- [docs/mcp-discoverability-v1.md](./mcp-discoverability-v1.md) defines the target V1 design
- this file records what is actually implemented now

## Repository Status

### Implemented In `popiartcli`

- `popiart mcp serve`
  - starts a real stdio MCP server
  - supports `initialize`, `ping`, `tools/list`, and `tools/call`
  - supports newline-delimited JSON-RPC over stdio
  - also supports `Content-Length` framed JSON-RPC for compatibility
- `popiart mcp serve --describe`
  - prints the current server metadata and tool surface
- `popiart mcp print-config`
  - prints a generic MCP server config snippet for an agent
- `popiart mcp doctor`
  - checks local discoverability state and remote runtime-baseline readiness
- `popiart bootstrap --install-mcp`
  - generates `~/.popiart/agents/<agent>/mcp.json`
- `popiart bootstrap --install-skill`
  - generates `~/.popiart/agents/<agent>/SKILL.md`
- `popiart bootstrap --with-runtime-baseline`
  - generates `~/.popiart/skillsets/runtime-baseline.json`
- `popiart bootstrap --discoverable`
  - convenience flag that combines discoverability assets

### Implemented MCP Tool Surface

The current server exposes these tools:

- `list_skills`
- `get_skill`
- `get_skill_schema`
- `run_skill`
- `get_job`
- `wait_job`
- `get_job_logs`
- `list_artifacts`
- `pull_artifact`
- `whoami`
- `current_project`

### Implemented Runtime-Baseline Definition

The repository now treats these three skill ids as the official runtime baseline:

1. `popiskill-image-text2image-basic-v1`
2. `popiskill-image-img2img-basic-v1`
3. `popiskill-video-image2video-basic-v1`

The `img2img` and `image2video` execution contracts have been written in [docs/mcp-discoverability-v1.md](./mcp-discoverability-v1.md).

## Verified

The current repo-local implementation has been verified with:

- `go test ./...`
- `go run ./cmd/popiart mcp serve --describe`

Tests currently cover:

- MCP `initialize`
- MCP `tools/list`
- MCP `tools/call` using `current_project`
- `Content-Length` response framing
- bootstrap generation for:
  - agent env files
  - agent MCP config snippets
  - agent skill wrappers

## What Is Not Done Yet

### Not Done In `popiartcli`

- agent-native installation into each product's real MCP config or skill directory
  - current bootstrap only generates assets under `~/.popiart/agents/<agent>/`
  - users or future installers still need to copy, link, or merge those assets into the real agent config
- MCP `resources`
- MCP `prompts`
- MCP `sampling`
- richer artifact-aware tool results such as `primary_artifact_id` or artifact-role metadata

### Not Done Outside This Repo

These items still belong to `popiartServer` or `PopiNewAPI` and are not solved by this repo alone:

- remote registration of the three official runtime-baseline skills
- default route mapping for `text2image`, `img2img`, and `image2video`
- provider-specific execution for masks, motion controls, duration limits, output fetching, and billing attribution
- guaranteed end-to-end availability of the three baseline skills

Because of that, the current state is:

- `popiartcli` can make `PopiArt` discoverable
- `popiartcli` can expose a usable MCP tool surface
- `popiartcli` can diagnose whether remote runtime pieces are present
- `popiartcli` cannot, by itself, guarantee that all three baseline runtime skills will execute successfully end to end

## Recommended Next Steps

1. Add agent-specific installers so `bootstrap` can write directly into the real config format for `codex`, `claude-code`, `openclaw`, and `opencode`.
2. Register the three baseline runtime skills by default in `popiartServer`.
3. Wire default routes for `text2image`, `img2img`, and `image2video`.
4. Validate that `popiart mcp doctor` passes against a real deployed environment.
