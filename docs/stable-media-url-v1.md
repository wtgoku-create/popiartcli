# Stable Media URL V1

Date: `2026-04-08`

This document defines the V1 plan for upgrading PopiArt from a download-oriented artifact system to a stable-media-url workflow suitable for `img2img`, `image2video`, and future `video2video` skills.

It complements, rather than replaces:

- [docs/project-relationship.md](./project-relationship.md)
- [docs/mcp-discoverability-v1.md](./mcp-discoverability-v1.md)

## Goal

Every PopiArt input or output file that may be reused by multimodal models should be available through a stable URL owned by PopiArt, instead of relying on provider-temporary signed URLs or repeated local base64 conversion.

The product language may say "permanent URL", but the engineering contract is:

- long-lived
- PopiArt-owned
- revocable
- auditable
- reusable across jobs and skills

## Scope

This V1 plan covers:

- `popiartcli` command and MCP surfaces for media upload
- `popiartServer` media storage and artifact binding
- runtime resolution from `source_artifact_id` to `image_url`
- adapter responsibilities for providers that do not accept URL input directly

This V1 plan does not require:

- exposing raw object-store URLs to clients
- irreversible public publishing
- changing existing runtime skill inputs away from `source_artifact_id`

## Cross-Repo Decisions

### Ownership

- `popiartcli`
  - user and agent command surface
  - MCP tool surface
  - local UX and output formatting
- `popiartServer`
  - media storage abstraction
  - artifact to media binding
  - URL generation
  - runtime dispatch resolution
  - output persistence policy
- `PopiNewAPI`
  - provider-specific input adaptation
  - fallback fetch-and-transform when a provider cannot consume PopiArt media URLs directly

### Primary Business Identifier

`artifact_id` remains the primary skill-facing identifier in V1.

`media` is introduced as a lower-level storage entity. Each artifact may bind to one media record through `media_id`.

### URL Semantics

The system should use "stable URL" semantics:

- available long enough for normal creator workflows
- backed by PopiArt-managed storage
- not tied to provider-generated expiry
- revocable by policy

### Visibility

Default visibility should be `unlisted`, not globally enumerable public assets.

The URL must still be anonymously retrievable by model providers.

## Data Model

### Artifact

V1 artifact metadata should include:

- `id`
- `job_id`
- `filename`
- `content_type`
- `size_bytes`
- `created_at`
- `expires_at`
- `media_id`
- `url`
- `visibility`
- `sha256`
- `storage_status`

### Media

V1 media metadata should include:

- `id`
- `artifact_id`
- `project_id`
- `filename`
- `content_type`
- `size_bytes`
- `created_at`
- `url`
- `visibility`
- `sha256`

## API Surface

### popiartServer

Extend existing APIs:

- `POST /v1/artifacts/upload`
  - continue returning `artifact_id`
  - also return `media_id`, `url`, and `visibility` when supported
- `GET /v1/artifacts/:id`
  - include stable URL metadata
- `GET /v1/jobs/:id/artifacts`
  - include stable URL metadata for each artifact

Add media APIs:

- `POST /v1/media/upload`
- `GET /v1/media/:id`

### popiartcli

Add:

- `popiart media upload <path>`
- `popiart media get <media-id>`

Extend:

- `popiart artifacts upload <path>`
  - now surfaces `url` and `media_id` when the server returns them

### MCP

Add:

- `upload_media`
- `get_media`

Extend:

- `upload_artifact`
  - include `media_id` and `url` when available

## Runtime Dispatch Rules

For multimodal runtime skills, `popiartServer` should resolve artifact references before calling provider adapters.

Examples:

- `source_artifact_id` -> `image_url`
- `mask_artifact_id` -> `mask_url`
- `reference_artifact_ids` -> `reference_urls`

The skill contract does not need to change. Agents may continue sending `source_artifact_id`.

## Output Persistence Rules

Provider-returned outputs must be copied into PopiArt-managed storage before being surfaced as reusable artifacts.

That means:

- `txt2img` outputs are re-hosted by PopiArt
- `img2img` outputs are re-hosted by PopiArt
- `image2video` outputs are re-hosted by PopiArt

Provider-temporary signed URLs must not be stored as the final reusable artifact URL.

## Provider Adaptation Rules

If a provider accepts a remote URL directly:

- pass the PopiArt stable URL through unchanged

If a provider requires multipart upload or base64:

- fetch the PopiArt stable URL server-side
- transform inside `PopiNewAPI` or the server adapter
- do not push that complexity into CLI or agent clients

## Rollout Plan

1. Finalize the shared ADR and schema.
2. Implement media storage in `popiartServer`.
3. Bind new artifacts to media records.
4. Update runtime dispatch to resolve artifact IDs into URLs.
5. Re-host all model outputs before artifact creation.
6. Ship `popiartcli` media commands and MCP tools.
7. Backfill recent artifact records where recoverable.

## Acceptance Criteria

- `popiart artifacts upload` returns a stable URL when the server supports it.
- `popiart media upload` returns a stable URL.
- `img2img` and `image2video` can reuse prior outputs without local download and re-upload.
- `GET /artifacts/:id` returns stable URL metadata.
- old `artifacts pull` flows continue to work.

## Risks

- public URL exposure if tokenization or visibility defaults are wrong
- provider fetch instability for some regions or vendors
- storage and egress cost growth for video outputs
- historical artifacts may not be fully backfillable if only upstream expiring URLs remain
