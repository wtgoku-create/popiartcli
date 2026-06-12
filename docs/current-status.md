# PopiArt CLI Current Status

Date: `2026-06-12`

This document summarizes the current repository-local status of `popiartcli` after the main-site task migration pass.

It is intentionally different from the design docs:

- [docs/project-relationship.md](./project-relationship.md) defines ownership boundaries
- [docs/mcp-discoverability-v1.md](./mcp-discoverability-v1.md) defines the target V1 design
- this file records what is actually implemented now

## Repository Status

### Repository Shape

- The authoritative product implementation is the Go CLI under `cmd/popiart` plus `internal/`.
- The root `package.json` is now repository-task metadata only and no longer publishes the legacy Node shell as the active CLI entrypoint.
- The old Node shell under `bin/` and `src/` is retained only as migration reference and should not be treated as the shipped product surface.

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
  - also reports the resolved native MCP config path and native skill directory
- `popiart mcp doctor`
  - checks local discoverability state and remote runtime-baseline readiness
- `popiart bootstrap --install-mcp`
  - generates `~/.popiart/agents/<agent>/mcp.json`
  - also writes the resolved native MCP config for `codex`, `claude-code`, `openclaw`, and `opencode`
- `popiart bootstrap --install-skill`
  - generates `~/.popiart/agents/<agent>/SKILL.md`
  - also writes a native `popiart` skill wrapper into the resolved agent skill directory
- `popiart bootstrap --with-runtime-baseline`
  - generates `~/.popiart/skillsets/runtime-baseline.json`
- `popiart bootstrap --discoverable`
  - convenience flag that combines discoverability assets
  - now makes `PopiArt` immediately visible from the supported agents' native MCP and skill directories
- `popiart artifacts upload`
  - keeps the artifact-shaped command surface for compatibility
  - now uploads through `/api_client/media/upload` underneath
  - returns `artifact_id` as the compatibility-facing form of `media.id`
  - supports the common `agent chat attachment -> artifact -> img2img` path
- `popiart media upload`
  - uploads a local file and requests a stable media URL from the server
  - is the native media-facing upload surface in the current migration mode
- `popiart media get`
  - resolves media metadata through `/api_client/media/detail?id=...`
- `popiart skills pull/install/use-local`
  - supports installed local skills without changing bundled seed skills
  - merges installed local skills into `skills list/get/schema`
  - allows `popiart run` to resolve `execution.mode=remote-runtime` from an installed local skill when it maps to an already bridged official skill

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
- `upload_artifact`
- `get_media`
- `upload_media`
- `whoami`
- `current_project`

### Implemented Runtime-Baseline Definition

The repository now treats these seven skill ids as the official runtime baseline:

1. `popiskill-image-text2image-basic-v1`
2. `popiskill-image-img2img-basic-v1`
3. `popiskill-image-img2img-popistudio-alice-showcase-v1`
4. `popiskill-video-image2video-basic-v1`
5. `popiskill-video-image2video-popistudio-alice-showcase-v1`
6. `popiskill-audio-tts-multimodel-v1`
7. `popiskill-audio-stt-local-v1`

The `img2img` and `image2video` execution contracts have been written in [docs/mcp-discoverability-v1.md](./mcp-discoverability-v1.md).

As of `2026-06-12`, all seven runtime-baseline skills are also exposed as built-in official contracts in `popiartcli`, and the currently bridged generation skills now execute through the main-site task pipeline:

- `skills list/get/schema` exposes a local contract even when the remote catalog entry is missing or still a placeholder
- `run popiskill-image-text2image-basic-v1` bridges to task-based image generation
- `run popiskill-image-img2img-basic-v1` bridges to task-based image editing
- `run popiskill-video-image2video-basic-v1` bridges to task-based image-to-video
- `run popiskill-audio-tts-multimodel-v1` bridges to task-based TTS
- installed local skills also participate in `skills list/get/schema`, and can execute through `run` when their `runtime_skill_id` maps to one of the bridged official skills

## Verified

The current repo-local implementation has been verified with:

- `go test ./internal/cmd ./internal/popiart`
- `go run ./cmd/popiart mcp serve --describe`
- `go run ./cmd/popiart artifacts upload --help`
- `go run ./cmd/popiart media upload --help`
- `go run ./cmd/popiart skills pull --help`
- `go run ./cmd/popiart skills install --help`
- `go run ./cmd/popiart skills use-local --help`

Tests currently cover:

- MCP `initialize`
- MCP `tools/list`
- MCP `tools/call` using `current_project`
- `Content-Length` response framing
- bootstrap generation for:
  - agent env files
  - agent MCP config snippets
  - agent skill wrappers
- native path resolution for:
  - `codex`
  - `claude-code`
  - `openclaw`
  - `opencode`
- native MCP config installation for Codex TOML and JSON-based agent configs
- local skill install / use-local linking into native agent skill directories by default
- installed local skill metadata parsing and activation
- artifact-compat upload client / command / MCP integration over media upload
- media upload client / command / MCP integration

## Deployed Validation

Against the current test environment, the following end-to-end paths have been validated:

- auth login / whoami
- skill listing / skill schema lookup
- artifact-compat upload over media upload
- media detail lookup
- text-to-image
- `img2img` using `source_artifact_id`
- `image2video` using `source_artifact_id`
- text-to-video submission
- audio / music task submission

Validated server-side `img2img` route adapters include:

- `gemini-3-pro-image-preview`
- `seedream-4-5-251128`
- `image-01`

Validated test-environment `image2video` routing currently includes:

- `video.image2video -> viduq2-pro-fast`

Validated test-environment MiniMax image/video routing currently includes:

- `image-01` text-to-image
- `image-01` img2img
- `image-01-live` text-to-image with `style`
- `MiniMax-Hailuo-2.3` text-to-video
- `MiniMax-Hailuo-2.3` image-to-video
- `MiniMax-Hailuo-02` start-end video
- `S2V-01` subject-reference video submission

The CLI does not guarantee those provider-specific adapters by itself; they were validated against a deployed `popiartServer` plus `PopiNewAPI` environment.

## What Is Not Done Yet

### Not Done In `popiartcli`

- MCP `resources`
- MCP `prompts`
- MCP `sampling`
- richer artifact-aware tool results such as `primary_artifact_id` or artifact-role metadata
- direct local execution for arbitrary installed skills beyond `execution.mode=remote-runtime`
- built-in compatibility execution for arbitrary remote skills beyond the current bridged official skill set

### Not Done Outside This Repo

These items still belong to `popiartServer` or `PopiNewAPI` and are not solved by this repo alone:

- remote registration of the seven official runtime-baseline skills
- default route mapping for `text2image`, `img2img`, `image2video`, `audio.tts`, and `audio.stt`
- provider-specific execution for masks, motion controls, duration limits, voice selection, transcript shaping, output fetching, and billing attribution
- guaranteed end-to-end availability of the seven baseline skills

The current test deployment still needs explicit project-level overrides for some routes when the request is going through the server-managed runtime path. For example, the older server-side `image2video` validation was done only after setting `video.image2video -> viduq2-pro-fast`.

Because of that, the current state is:

- `popiartcli` can make `PopiArt` discoverable
- `popiartcli` can expose a usable MCP tool surface
- `popiartcli` can diagnose whether remote runtime pieces are present
- `popiartcli` can keep the bridged official generation skills usable even when the remote catalog entry is missing or still a placeholder
- `popiartcli` still cannot, by itself, guarantee that every baseline skill and every compatibility command will execute successfully end to end

Operationally, this means:

- `setup` / `bootstrap --discoverable` should be read as discoverability success, not end-to-end runtime success
- `mcp doctor` is the required follow-up because it separates `discoverability_status` from `runtime_status`
- adding more façade commands is lower priority than completing the agent-facing MCP surface and runtime-readiness diagnostics

## Recommended Next Steps

1. Complete the agent-facing MCP surface with `resources`, `prompts`, `sampling`, and richer artifact-aware tool results before adding more façade commands.
2. Validate the native MCP install path against real installed `Claude Code`, `OpenClaw`, and `OpenCode` binaries on each target OS.
3. Publish the tested `popiartServer` route adapters and defaults as a real tracked server release, including `video.image2video -> viduq2-pro-fast`.
4. Register the seven baseline runtime skills by default in `popiartServer`.
5. Validate that `popiart mcp doctor` passes against a real deployed environment with the intended default route table.
