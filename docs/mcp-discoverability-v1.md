# PopiArt MCP Discoverability V1

This document defines the first repository-local implementation slice for making `popiart` discoverable to coding agents after installation.

It covers four tracks:

1. a discoverable `PopiArt` MCP/server identity
2. bootstrap-generated agent assets for MCP and skill directories
3. a runtime baseline for the first three official multimodal skills
4. the execution contract for `img2img` and `image2video`

This is a cross-repo design. `popiartcli` owns the CLI surface, bootstrap assets, and local diagnostics. `popiartServer` owns runtime skill registration, routing, jobs, artifacts, and billing. `PopiNewAPI` owns provider channels, upstream model access, and provider-specific adaptation.

## Goals

- After install, users should be able to find `PopiArt` in an agent-facing MCP or skill directory.
- The official multimodal baseline should be easy to understand:
  `text2image`, `img2img`, and `image2video`.
- Agent integrations should target one stable product-layer tool surface instead of provider-specific APIs.
- Diagnostics should make it obvious whether the system is merely discoverable or actually runnable end to end.

## Scope In `popiartcli`

`popiartcli` owns the following V1 work:

- `popiart mcp print-config`
- `popiart mcp doctor`
- `popiart mcp serve` stdio server for `initialize`, `ping`, `tools/list`, and `tools/call`
- `popiart bootstrap --install-mcp`
- `popiart bootstrap --install-skill`
- `popiart bootstrap --with-runtime-baseline`
- `popiart bootstrap --discoverable`
- generated bootstrap assets under `~/.popiart/agents/<agent>/`

`popiartcli` does not own:

- provider execution logic
- skill registration sync
- route selection
- upstream provider keys
- provider job polling

## Discoverability Model

The public name presented to agents is:

- server name: `PopiArt`
- server id / slug: `popiart`

Bootstrap should generate per-agent assets under:

```text
~/.popiart/agents/<agent>/
  env.sh
  env.ps1
  mcp.json
  SKILL.md
```

These files are bootstrap artifacts. They are not a guarantee that a given agent has already ingested them into its own native config format.

## MCP Tool Surface

The V1 MCP tool surface is product-layer, not provider-layer:

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

The tools mirror the existing CLI command tree and should not expose provider keys, raw gateway routes, or provider-native request shapes.

The repository-local transport supports:

- newline-delimited JSON-RPC over stdio
- header-framed `Content-Length` JSON-RPC compatibility mode for older clients

## Official Runtime Baseline

The V1 official runtime baseline consists of three remote runtime skills:

1. `popiskill-image-text2image-basic-v1`
2. `popiskill-image-img2img-basic-v1`
3. `popiskill-video-image2video-basic-v1`

These are official runtime skills, not local-only bundled seed skills.

Expected product behavior:

- `popiartServer` registers all three runtime skills by default
- `popiartServer` exposes them through `/skills`
- `popiartServer` resolves each skill to a valid route
- `PopiNewAPI` has at least one working upstream channel for each needed capability

## Img2Img Contract

Recommended runtime skill id:

```text
popiskill-image-img2img-basic-v1
```

Required input:

- `image_artifact_id`

Optional input:

- `prompt`
- `negative_prompt`
- `mask_artifact_id`
- `strength`
- `preserve_composition`
- `style`
- `reference_artifact_ids`
- `seed`
- `output_format`
- `notes`

Execution outline:

1. validate the source image artifact
2. validate the mask artifact when present
3. normalize the prompt and defaults
4. choose either transform or edit/inpaint routing
5. submit the upstream task through `PopiNewAPI`
6. poll upstream state through `popiartServer`
7. persist the result image as an artifact
8. persist one metadata JSON artifact
9. surface progress through `jobs logs`

Output shape:

- one main image artifact, for example `result.png`
- optional preserved mask artifact, for example `mask.png`
- one metadata artifact, `result.json`

The metadata artifact should include:

- `skill_id`
- `source_artifact_id`
- `mask_artifact_id`
- `result_filename`
- `resolved_prompt`
- `route_key`
- `provider`
- `model_id`
- `width`
- `height`
- `seed`

## Image2Video Contract

Recommended runtime skill id:

```text
popiskill-video-image2video-basic-v1
```

Required input:

- `image_artifact_id`

Optional input:

- `prompt`
- `negative_prompt`
- `duration_s`
- `fps`
- `camera_motion`
- `motion_intensity`
- `loop`
- `style`
- `seed`
- `notes`

Execution outline:

1. validate the source image artifact
2. normalize duration, ratio, and motion controls
3. submit the video task through `PopiNewAPI`
4. track provider progress in `jobs logs`
5. persist the main video as an artifact
6. persist an optional poster image
7. persist one metadata JSON artifact

Output shape:

- one main video artifact, `result.mp4`
- one optional poster artifact, `poster.jpg`
- one metadata artifact, `result.json`

The metadata artifact should include:

- `skill_id`
- `source_artifact_id`
- `video_filename`
- `poster_filename`
- `resolved_prompt`
- `route_key`
- `provider`
- `model_id`
- `duration_s`
- `fps`
- `width`
- `height`
- `seed`

## Doctor Checks

`popiart mcp doctor` should distinguish between discoverability and runtime readiness.

The V1 checks are:

- local config dir exists
- endpoint is configured
- key is present
- `auth/me` responds
- `/skills` responds
- each baseline runtime skill resolves from `/skills/<id>`
- `/models/routes` responds
- generated agent bootstrap files exist when an `--agent` is provided

Recommended statuses:

- `pass`
- `warn`
- `fail`

The command should return structured output even when some checks fail.

## External Dependencies For Step 4

The following items are intentionally out of scope for this repo and must be completed in `popiartServer` or `PopiNewAPI`:

- remote registration of the three official runtime baseline skills
- route mapping for `text2image`, `img2img`, and `image2video`
- provider-specific adaptation for masks, motion controls, duration limits, and output fetching
- stable artifact naming conventions in the server runtime
- billing attribution across skill, project, and user scopes

Until those pieces land, `popiartcli` can make `PopiArt` visible and diagnosable, but not fully runnable end to end for every baseline skill.
