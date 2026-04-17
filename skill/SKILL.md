---
name: popiart-cli
description: Use PopiArt when an agent needs creator skill discovery, multimodal runtime execution, jobs, artifacts, stable media URLs, or an MCP server entrypoint. Prefer the intent-first image/video/audio commands for common media tasks, and fall back to skills/run/jobs/artifacts for platform-level control.
---

# PopiArt CLI Agent Contract

Use `popiart` as an agent-facing runtime for creator skills and multimodal jobs.

## Standard Agent Flags

In agent or CI contexts, prefer these flags consistently:

| Flag | Purpose |
|---|---|
| `--output json` | Stable machine-readable stdout envelope |
| `--quiet` | Reserve stderr for unavoidable prompts or diagnostics |
| `--non-interactive` | Fail fast instead of prompting for missing input |
| `--dry-run` | Preview the normalized network request without executing writes |
| `--async` | Be explicit that the command should return a job immediately |
| `--wait` | Block until the job finishes |

Notes:

- `--output plain` is supported for human-readable output; `--plain` remains as a compatibility alias.
- PopiArt keeps stdout data-oriented. When a command supports `--dry-run`, the preview is emitted on stdout using the selected output format.
- `--yes` is available as a stable confirmation flag for future prompts; current media/runtime flows do not require it.

## Intent-First Commands

Prefer these commands for common user intents:

### Generate an image

```bash
popiart image generate \
  --prompt "A cinematic portrait of a young creator in warm sunset light" \
  --aspect-ratio 9:16 \
  --output json \
  --quiet \
  --non-interactive
```

Maps to: `popiskill-image-text2image-basic-v1`

### Transform an image into a new image

```bash
popiart image img2img \
  --image ./source.png \
  --prompt "Turn this into a watercolor illustration" \
  --strength 0.6 \
  --output json \
  --quiet \
  --non-interactive
```

Maps to: `popiskill-image-img2img-basic-v1`

Input rules:

- `--image` accepts a local file path, stable media URL, or supported data URL.
- `--identity-reference-image` and `--style-reference-image` accept local file paths or URLs and may be repeated.
- `--source-artifact-id`, `--identity-reference-artifact-id`, and `--style-reference-artifact-id` are preferred when the images already exist inside PopiArt and should be reused across steps.
- Local files are uploaded automatically when the selected runtime path requires artifacts.

Role-aware multi-image pattern:

```bash
popiart image img2img \
  --image https://server.popi.art/v1/media/med_scene/content \
  --identity-reference-image https://example.com/identity.jpg \
  --style-reference-image https://example.com/style.png \
  --prompt "Replace the person in the source scene with the main character from the identity reference. Keep the exact action and framing from the source scene. Apply only the style from the style reference." \
  --preserve-composition \
  --output json \
  --quiet \
  --non-interactive
```

Agent guidance:

- Prefer `source + identity` first when character consistency is the highest priority.
- Add `style` as a second step when a single-pass three-image edit drifts too far from the source action or subject identity.
- Prefer `--dry-run` before unfamiliar multi-image edits to inspect the normalized request body.

### Generate a video from a source image

```bash
popiart video generate \
  --image ./source.png \
  --prompt "Subtle camera push-in and natural hair movement" \
  --duration 5 \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

Maps to: `popiskill-video-image2video-basic-v1`

Behavior:

- `--image` accepts either an `https://...` URL or a local file path.
- When `--image` is a local file, PopiArt uploads it first as a source artifact, then submits the runtime job.
- `--source-artifact-id` can be used instead of `--image` when the source is already uploaded.

For an explicit command name, `popiart video img2video ...` is equivalent to `popiart video generate ...`.

### Text-to-speech

```bash
popiart speech synthesize \
  --text "今天我们来做一个更适合 agent 调用的 CLI。" \
  --voice narrator_female \
  --format mp3 \
  --output json \
  --quiet \
  --non-interactive
```

Default model: `speech-2.8-hd` via direct model infer

### Music generation

```bash
popiart music generate \
  --prompt "Upbeat pop" \
  --lyrics "La la la" \
  --output json \
  --quiet \
  --non-interactive
```

Or instrumental:

```bash
popiart music "Warm morning folk" \
  --instrumental \
  --output json \
  --quiet \
  --non-interactive
```

Default model: `music-2.6-free` via direct model infer

## Platform Commands

Use the lower-level platform surface when the agent needs precise control:

```bash
popiart skills get <skill-id> --output json --quiet --non-interactive
popiart skills schema <skill-id> --output json --quiet --non-interactive
popiart run <skill-id> --input @params.json --output json --quiet --non-interactive
popiart jobs wait <job-id> --output json --quiet --non-interactive
popiart artifacts pull-all <job-id> --output json --quiet --non-interactive
```

## Dry-Run Pattern

Use `--dry-run` before any write when the agent needs to inspect the exact request shape:

```bash
popiart image generate \
  --prompt "Editorial skincare product shot" \
  --aspect-ratio 1:1 \
  --dry-run \
  --output json \
  --quiet \
  --non-interactive
```

PopiArt returns:

- normalized `skill_id`
- request `method`, `path`, and `body`
- resolved agent protocol metadata such as output mode and wait/async behavior

## Official Runtime Baseline

- `popiskill-image-text2image-basic-v1`
- `popiskill-image-img2img-basic-v1`
- `popiskill-image-img2img-popistudio-alice-showcase-v1`
- `popiskill-video-image2video-basic-v1`
- `popiskill-video-image2video-popistudio-alice-showcase-v1`
- `popiskill-audio-tts-multimodel-v1`
- `popiskill-audio-stt-local-v1`

## MCP Entrypoint

```bash
popiart mcp serve
```

Use `popiart mcp print-config --agent codex` or `popiart bootstrap --agent codex --discoverable` when the agent host needs discoverability scaffolding.
