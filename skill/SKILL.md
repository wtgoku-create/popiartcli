---
name: popiart-cli
description: Use PopiArt to discover and run creator skills for image, video, animation, audio, jobs, artifacts, budgets, model routing, and per-project context from the terminal. Use when the user mentions popiart, popiskill-*, skillhub.popi.art, or asks to generate or transform multimodal content such as text-to-image, img2img, image-to-video, TTS, music, upscaling, or job/artifact management.
---

# PopiArt CLI

Use `popiart` as the agent-facing runtime for PopiArt creator workflows. The CLI handles authentication, skill discovery, job orchestration, artifact transport, budgeting, routing, and MCP discoverability, so agents should not call upstream model providers directly.

## Install And Setup

Install the CLI first:

```bash
brew tap wtgoku-create/popi
brew install wtgoku-create/popi/popiart
```

Or:

```bash
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | sh
```

Then initialize the local agent integration:

```bash
popiart setup --agent codex --completion zsh
popiart auth login --key <product-key>
popiart mcp doctor --agent codex
```

Supported `--agent` values are `codex`, `claude-code`, `openclaw`, and `opencode`.

## Output Contract

Default stdout is JSON:

```json
{ "ok": true, "data": {} }
```

Failures use:

```json
{ "ok": false, "error": { "code": "VALIDATION_ERROR", "message": "..." } }
```

In agent or CI contexts, prefer:

```bash
--output json --quiet --non-interactive
```

Useful global flags:

| Flag | Purpose |
|---|---|
| `--output json` | Stable machine-readable output. This is the default. |
| `--output plain` / `--plain` | Human-readable output. |
| `--quiet` | Suppress non-result output. |
| `--non-interactive` | Fail instead of prompting. |
| `--dry-run` | Preview normalized requests without executing network writes. |
| `--async` | Return a job immediately. |
| `--wait` | Block until the job reaches a terminal state. |
| `--endpoint <url>` | Override the API endpoint. |
| `--project <id>` | Override the active project. |
| `--no-color` | Disable ANSI color in plain mode. |

Treat the JSON envelope and `error.code` as the source of truth. Do not pattern-match on human-readable messages.

## Intent Commands

Prefer these high-level commands for common user requests.

### Text To Image

```bash
popiart image generate \
  --prompt "a sunset over Tokyo, cinematic, 35mm" \
  --aspect-ratio 16:9 \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

### Image Description

```bash
popiart image describe \
  --image ./source.png \
  --model gemini-2.5-flash \
  --prompt "Write a reusable text-to-image prompt" \
  --output json \
  --quiet \
  --non-interactive
```

### Img2Img

```bash
popiart image img2img \
  --image ./source.png \
  --prompt "Keep the subject, recolor to dusk cinematic" \
  --strength 0.6 \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

For role-aware multi-image edits:

```bash
popiart image img2img \
  --image ./scene.png \
  --identity-reference-image ./character.png \
  --style-reference-image ./style.png \
  --prompt "Replace the person in the source scene with the character from the identity reference. Preserve action and camera framing. Apply only the visual style from the style reference." \
  --preserve-composition \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

### Image To Video

```bash
popiart video generate \
  --image ./source.png \
  --prompt "Hair and fabric drift in a soft breeze; slow camera push-in" \
  --duration 5 \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

`popiart video img2video` is an explicit alias for this flow.

Optional prompt enhancement:

```bash
popiart video generate \
  --image ./source.png \
  --prompt "Make the person naturally turn toward camera" \
  --prompt-enhancer-model gemini-2.5-flash \
  --model viduq2-pro-fast \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

### Action Transfer

```bash
popiart video action-transfer \
  --image ./face.jpg \
  --video https://example.com/source-action.mp4 \
  --cut-result-first-second-switch \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

### Speech And Music

```bash
popiart speech synthesize \
  --text "Today we are building a CLI for agents." \
  --voice narrator_female \
  --format mp3 \
  --output json \
  --quiet \
  --non-interactive
```

```bash
popiart music generate \
  --prompt "Upbeat pop" \
  --lyrics "La la la" \
  --output-format url \
  --format mp3 \
  --output json \
  --quiet \
  --non-interactive
```

## Platform Commands

Use lower-level commands when the agent needs exact skill control:

```bash
popiart skills list --search "image upscale" --output json --quiet --non-interactive
popiart skills get <skill-id> --output json --quiet --non-interactive
popiart skills schema <skill-id> --output json --quiet --non-interactive
popiart run <skill-id> --input @params.json --wait --output json --quiet --non-interactive
popiart jobs wait <job-id> --output json --quiet --non-interactive
popiart artifacts pull-all <job-id> --dir ./results --output json --quiet --non-interactive
```

When picking a skill, prefer `skills list --search` or `skills schema` over guessing ids. The returned `id` is the value to pass to `run`.

## File Transport

For intent commands, local files can usually be passed directly with flags like `--image`; the CLI uploads them when needed.

For low-level `popiart run`, do not place local file paths inside `--input`. Upload first:

```bash
ARTIFACT_ID=$(popiart artifacts upload ./source.png --role source --output json --quiet --non-interactive | jq -r '.data.artifact_id')

popiart run popiskill-image-img2img-basic-v1 \
  --input "{\"source_artifact_id\":\"$ARTIFACT_ID\",\"prompt\":\"keep the subject, cinematic dusk\"}" \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

If the agent already has a public HTTPS URL or stable PopiArt media URL, pass that URL directly when the target command or schema supports it.

## Jobs

Prefer `run --wait` or `jobs wait` over manual polling:

```bash
popiart jobs get <job-id> --output json --quiet --non-interactive
popiart jobs wait <job-id> --output json --quiet --non-interactive
popiart jobs logs <job-id> --output json --quiet --non-interactive
popiart jobs cancel <job-id> --output json --quiet --non-interactive
```

If a wait command returns `POLL_TIMEOUT`, the server-side job may still be running. Resume with `popiart jobs wait <same-job-id>`; do not submit the same generation again.

## Budget, Project, Models

Re-query these values instead of assuming they are current:

```bash
popiart budget status --output json --quiet --non-interactive
popiart budget usage --group-by skill --output json --quiet --non-interactive
popiart project current --output json --quiet --non-interactive
popiart project context --output json --quiet --non-interactive
popiart models list --output json --quiet --non-interactive
popiart models routes --output json --quiet --non-interactive
```

Default to `popiart run` with a skill. Use `popiart models infer` only when the user explicitly asks for a specific model.

## Model Switching

Use the lightest model-switching surface that matches the user's intent:

| Intent | Use |
|---|---|
| One request should use a named model | Pass `--model <model-id>` on intent commands that support it. |
| A project should keep using a model for a whole skill type | Use `popiart models route-override set`. |
| The user wants to call a raw model directly | Use `popiart models infer <model-id>`. |
| The user only asks for "best/default" behavior | Do not switch models; let PopiArt route through the skill. |

Single-request override examples:

```bash
popiart image generate \
  --model image-01 \
  --prompt "A clean editorial product photo" \
  --aspect-ratio 1:1 \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

```bash
popiart video generate \
  --model viduq2-pro-fast \
  --image ./source.png \
  --prompt "Subtle camera push-in and natural motion" \
  --duration 5 \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

Project-level route override examples:

```bash
popiart models route-override set \
  --project proj_abc123 \
  --skill-type image.img2img \
  --model seedream-4-5-251128 \
  --output json \
  --quiet \
  --non-interactive
```

```bash
popiart models route-override set \
  --project proj_abc123 \
  --skill-type video.image2video \
  --model viduq2-pro-fast \
  --output json \
  --quiet \
  --non-interactive
```

Inspect and undo overrides:

```bash
popiart models route-override list --project proj_abc123 --output json --quiet --non-interactive
popiart models route-override unset --project proj_abc123 --skill-type video.image2video --output json --quiet --non-interactive
```

Agent rules:

- Prefer `--model` for a single user request.
- Prefer `route-override set` only when the user asks to pin future project behavior.
- Run `models list` or `models routes` before recommending a model, because available models and pricing can change.
- Avoid `models infer` when an intent command or skill exists, unless the user explicitly names a raw model or asks for low-level routing/debugging.

## MCP And Tool Schemas

Expose PopiArt to MCP-capable agents with:

```bash
popiart mcp serve
```

For agents that register CLI commands as native tools:

```bash
popiart export-schema --format openai
popiart export-schema --format anthropic
popiart export-schema --command "video generate" --format openai
```

`export-schema` emits raw tool schema JSON rather than the normal `{ ok, data }` envelope.

## Error Handling

Common `error.code` values:

| Code | Agent response |
|---|---|
| `UNAUTHENTICATED` | Stop and ask the user to run `popiart auth login`. |
| `FORBIDDEN` | Stop; surface the permission or project issue. |
| `NOT_FOUND` | Re-check ids with `skills list`, `jobs list`, or `artifacts list`. |
| `VALIDATION_ERROR` | Fix the invalid fields before retrying. |
| `RATE_LIMITED` | Back off, then retry. Optionally check `budget status`. |
| `JOB_FAILED` | Surface provider details; do not blindly retry. |
| `POLL_TIMEOUT` | Resume waiting on the same job id. |
| `NETWORK_ERROR` | Retry with backoff. After repeated failures, surface the issue. |
| `INPUT_PARSE_ERROR` | Fix JSON syntax. |
| `INPUT_NOT_FOUND` | Stop and check the local path. |
| `CONFLICT` | Reuse the existing result or choose a fresh idempotency key. |
| `SERVER_ERROR` | Retry once; if it persists, surface the issue. |
| `CLI_ERROR` / `FATAL` | Stop and report the full JSON envelope. |

## Anti-Patterns

- Do not call upstream providers directly when a PopiArt skill or command exists.
- Do not inline local file paths inside low-level `run --input`; upload first.
- Do not loop `sleep + jobs get`; use `jobs wait`.
- Do not re-run a job after `POLL_TIMEOUT`; wait on the same job id.
- Do not assume budget, pricing, routes, or available models; query them.
- Do not echo or commit `pk-...` keys.
- Do not run bundled seed or authoring-only skills as remote runtime skills.
