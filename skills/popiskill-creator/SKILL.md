---
name: popiskill-creator
description: Create, adapt, bootstrap, and validate PopiArt skills through popiartcli and Popiart_skillhub. Use this whenever the user wants to install or configure popiartcli, authenticate with a PopiArt API key, understand the unified gateway or 统一网关 boundary, turn a creator workflow into a PopiArt skill, update a skill in Popiart_skillhub, or run a PopiArt skill end to end with jobs and artifacts.
---

# PopiSkill Creator

## Overview

Use this skill to turn a user request into a real PopiArt operating path, not a hypothetical one.

Ground everything in the current public repos:

- `https://github.com/wtgoku-create/popiartcli`
- `https://github.com/wtgoku-create/Popiart_skillhub`

Read [references/popiart-platform.md](references/popiart-platform.md) whenever you need the exact install commands, environment variables, naming rules, or repo and gateway boundary language.

## Ground Rules

- Prefer the current `popiartcli` command surface over invented commands.
- Teach the user in terms of `key`, not `token`. Mention `token` only as a backward-compatible alias when needed.
- Treat `popiartcli` as the unified CLI entrypoint, not the place where upstream provider keys live.
- Never ask the user to paste OpenAI, Gemini, Sora, Kling, Runway, or other provider keys into `popiartcli`.
- Keep repository ownership and system ownership explicit so the user knows where a change belongs.

## Decide The Path

Figure out which of these jobs the user actually needs:

- **Bootstrap and run**: install `popiart`, authenticate, discover an existing skill, run it, wait for the job, and pull artifacts.
- **Create or update a skill**: inspect similar skills first, then draft or modify a `skills/<skill-name>/SKILL.md` entry for PopiArt.
- **Explain architecture or debug ownership**: clarify whether the problem belongs to `popiartcli`, the skill catalog, `popiartServer`, or `PopiNewAPI`.

If the user is vague, start with the bootstrap path and only switch to authoring once the execution path and ownership model are clear.

## Bootstrap `popiartcli`

Start by checking whether `popiart` is already available. If it is missing, guide the user through the current real install path from GitHub source.

Use this default sequence:

```sh
git clone https://github.com/wtgoku-create/popiartcli.git
cd popiartcli
go install ./cmd/popiart
popiart --help
```

Alternative local build path:

```sh
go build -o ./dist/popiart ./cmd/popiart
./dist/popiart --help
```

Important:

- Do not invent a `brew`, `curl | sh`, or package-manager install flow unless the public repo actually adds one.
- If the user asks for "download the CLI", explain the current public repo documents source install and local build.
- If the user is already inside the repo, prefer local build or `go run ./cmd/popiart --help`.

## Authenticate With A PopiArt Key

The user needs a PopiArt product-layer API key, not a raw provider key.

Default command pattern:

```sh
popiart auth login --key pk-...
popiart auth whoami
popiart auth key show
```

Environment overrides are acceptable when the user already manages secrets that way:

```sh
export POPIART_KEY=pk-...
export POPIART_ENDPOINT=https://api.creatoragentos.io/v1
```

Use `popiart auth key set <key>` only as a fallback when the user explicitly wants a local write without validation. Prefer `auth login --key` because it proves the key works.

## Explain The Unified Gateway Boundary

Keep this mental model simple and repeatable:

- `popiartcli`: local CLI entrypoint, config, auth UX, skill discovery, run commands, jobs, artifacts.
- `popiartServer`: product backend for auth, project context, skill registration and execution, job lifecycle, artifact management, routing decisions, and billing attribution.
- `PopiNewAPI`: model gateway for upstream providers, channel and provider key management, and raw model usage.

State the core rule plainly:

- `popiartcli` should hold the product-layer key.
- Real provider keys should stay behind the gateway and server boundary.

When the user is trying to fix a problem, route it to the correct layer instead of mixing concerns.

## Discover Before You Create

Before drafting a new skill, inspect what already exists in the catalog and repo.

CLI discovery flow:

```sh
popiart skills list --search "image"
popiart skills get <skill-id>
popiart skills schema <skill-id>
```

Repo discovery flow:

- inspect `Popiart_skillhub/skills/`
- look for the closest existing PopiArt skill and reuse its structure before inventing a new one
- keep the new skill aligned with the current catalog vocabulary

For runtime catalog skills, follow the current pattern:

```text
popiskill-<category>-<capability>-<slug>-v<major>
```

This authoring helper itself may stay named `popiskill-creator`, but when it generates or updates actual catalog skills, it should use the catalog naming convention.

## Draft Or Update A PopiArt Skill

When you produce or edit a PopiArt runtime skill, keep it compact and operational. A good PopiArt skill usually includes:

- a frontmatter `name`
- a trigger-oriented `description`
- clear required and optional inputs
- a short workflow
- one real `popiart run ...` command pattern
- a payload template when JSON input matters
- output handling with `jobs` and `artifacts`
- operating guidance for when to switch to a different skill

Write the skill around the current public CLI surface, not around an imagined future platform.

If the user wants a new catalog entry, place it under:

```text
skills/<skill-name>/
  SKILL.md
  agents/openai.yaml
  references/...   # only when needed
```

Do not add extra repository clutter such as `README.md` or `CHANGELOG.md` inside the skill folder.

## Run And Validate End To End

Once the skill or workflow is defined, close the loop through the real CLI:

```sh
popiart run <skill-id> --input @params.json --wait
```

After completion:

- inspect the returned `job_id`
- read `artifact_ids`
- pull outputs with `popiart artifacts pull` or `popiart artifacts pull-all`

If the user is iterating on a new skill spec, help them produce:

- the exact sample payload
- the exact CLI command to test it
- the expected artifact shape or completion signal

## Route Issues To The Right Repo

Use these ownership defaults:

- `popiartcli` repo: installation, command UX, config behavior, auth flow, environment variables, output format, local run ergonomics.
- `Popiart_skillhub` repo: public skill definitions, naming, descriptions, skill examples, catalog structure.
- `popiartServer`: registration sync, execution orchestration, jobs, artifacts, routing policy, billing attribution.
- `PopiNewAPI`: upstream channel management, provider model access, provider keys, raw model metering.

If a user says "the gateway is broken", pin down whether they mean CLI auth, server routing, or provider access before proposing a fix.

## Output Expectations

When using this skill, return the smallest useful package for the user's stage:

- for setup: exact install, auth, and verification commands
- for architecture questions: one clear boundary explanation and the responsible repo or layer
- for skill authoring: the draft skill files and the real test command
- for debugging: the next verification step, not a vague platform summary
