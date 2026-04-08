# PopiArt Platform Reference

Use this reference when you need current public facts about installation, authentication, environment variables, repo ownership, skillhub layout, media/artifact behavior, stable media URLs, or naming conventions.

## Public Repositories

- `popiartcli`: `https://github.com/wtgoku-create/popiartcli`
- `Popiart_skillhub`: `https://github.com/wtgoku-create/Popiart_skillhub`

## `popiartcli` Install Surface

Current public install guidance is source-first.

From GitHub:

```sh
git clone https://github.com/wtgoku-create/popiartcli.git
cd popiartcli
go install ./cmd/popiart
```

Alternative local build:

```sh
go build -o ./dist/popiart ./cmd/popiart
./dist/popiart --help
```

Inside the repo, these are also valid:

```sh
go run ./cmd/popiart --help
make build
make help
```

Do not promise a packaged installer unless the repo adds one.

## Current Auth Commands

User-facing language should prefer `key`.

```sh
popiart auth login --key <product-key>
popiart auth whoami
popiart auth key show
popiart auth key rotate
popiart auth logout
```

Compatibility note:

- `--token` exists as an alias in the CLI, but it should not be the main term in user guidance.
- `POPIART_TOKEN` also exists as an env alias, but `POPIART_KEY` should be documented first.
- A product-layer key may be issued with prefixes such as `pk-...` or `sk-...`; after `auth login`, the stored config may also contain a server-issued `sess_...` token.

## Current Environment Variables

```text
POPIART_KEY
POPIART_TOKEN
POPIART_ENDPOINT
POPIART_PROJECT
POPIART_CONFIG_DIR
```

Current default endpoint in the CLI config layer:

```text
https://api.creatoragentos.io/v1
```

## Ownership Boundary

Use this simplified model when explaining the platform:

- `popiartcli`: unified CLI entrypoint for coding agents and creators
- `popiartServer`: product backend for auth, project context, skill registration, execution, jobs, artifacts, media binding, stable URL serving, routing, and billing attribution
- `PopiNewAPI`: model gateway for upstream providers, provider channels, and provider key management

Key rule:

- the CLI stores a product-layer key
- provider keys stay behind the server and gateway boundary

## Media And Stable URL Surface

Current public CLI/media flow includes:

```sh
popiart media upload ./source.png
popiart media get <media-id>
popiart artifacts upload ./source.png --role source
popiart artifacts get <artifact-id>
```

Use this mental model:

- `artifact_id` is the main PopiArt skill-facing identifier
- `media_id` is the backing storage/media record when the server exposes one
- `url` returned by `media get` or `artifacts get` is the stable media URL when storage is ready

When teaching multimodal flows such as `img2img` or `image2video`, explain that PopiArt skills still prefer `source_artifact_id`, but the platform now also supports querying stable media URLs for reuse and inspection.

## Skillhub Layout

Public layout:

```text
skills/
index.json
README.md
```

The `skills/` directory currently includes:

- `skill-creator` as the upstream-style authoring reference skill
- many `popiskill-image-*`, `popiskill-video-*`, and `popiskill-audio-*` runtime skills

## Runtime Skill Naming

Current skillhub naming convention:

```text
popiskill-<category>-<capability>-<slug>-v<major>
```

Examples:

- `popiskill-image-text2image-basic-v1`
- `popiskill-image-img2img-basic-v1`
- `popiskill-video-image2video-basic-v1`
- `popiskill-audio-tts-multimodel-v1`

Use this pattern for runtime catalog skills. The local authoring helper `popiskill-creator` is a meta skill and may remain a shorter name.

## Minimal End-To-End Flow

This is the default loop to teach:

```sh
popiart auth login --key pk-...
popiart skills list --search "image"
popiart skills get <skill-id>
popiart run <skill-id> --input @params.json --wait
popiart artifacts pull <artifact-id> --out ./result.bin
popiart artifacts get <artifact-id>
popiart media get <media-id>
```

If a job fails, inspect the job first before changing the skill:

```sh
popiart jobs get <job-id>
popiart jobs wait <job-id>
popiart jobs logs <job-id>
```

## Authoring Guidance

When drafting or updating a runtime skill in the catalog:

- use a trigger-friendly description
- include one real command example
- include the smallest useful payload template
- explain how output artifacts are retrieved, and whether the workflow should also surface a stable media URL
- keep the skill concise and operational
- add `references/` only when the detail is too large for `SKILL.md`
