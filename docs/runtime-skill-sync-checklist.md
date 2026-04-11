# Runtime Skill Sync Checklist

Date: `2026-04-08`

This document defines how `popiartServer /skills` and `wtgoku-create/Popiart_skillhub` should stay aligned for the current seven official runtime skills.

## Source Of Truth

- `wtgoku-create/Popiart_skillhub`: public runtime skill definitions
- `popiartServer /skills`: runtime registry exposed to clients and agents
- `popiartcli`: discovery and execution client; it should treat `popiartServer /skills` as runtime truth and only use local official-runtime contracts as fallback

The intended relationship is:

```text
wtgoku-create/Popiart_skillhub
            ->
    popiartServer sync / register
            ->
        /skills API
            ->
         popiartcli
```

## Current Target Set

These seven skills should exist in both places with matching public metadata:

1. `popiskill-image-text2image-basic-v1`
2. `popiskill-image-img2img-basic-v1`
3. `popiskill-image-img2img-popistudio-alice-showcase-v1`
4. `popiskill-video-image2video-basic-v1`
5. `popiskill-video-image2video-popistudio-alice-showcase-v1`
6. `popiskill-audio-tts-multimodel-v1`
7. `popiskill-audio-stt-local-v1`

## Fields That Must Match Exactly

For each of the seven skill ids, keep these fields identical between skillhub and `popiartServer /skills`:

- `id`
- `name`
- `description`
- `tags`
- `version`
- `model_type`
- `estimated_duration_s`
- `input_schema`
- `output_schema`

These fields are runtime-only and do not need to exist in skillhub:

- enablement state
- project visibility
- route key / model mapping
- provider adapter details
- rollout flags
- billing and attribution data

## Current Observed Gap

From `go run ./cmd/popiart skills list --limit 100` against the locally configured server on `2026-04-08`, the merged result shows:

- `source=remote`
  - `popiskill-image-text2image-basic-v1`
  - `popiskill-image-img2img-basic-v1`
  - `popiskill-video-image2video-basic-v1`
- `source=official-runtime`
  - `popiskill-image-img2img-popistudio-alice-showcase-v1`
  - `popiskill-video-image2video-popistudio-alice-showcase-v1`
  - `popiskill-audio-tts-multimodel-v1`
  - `popiskill-audio-stt-local-v1`

That means the current runtime registry is still missing four server-registered entries, and at least the overlapping three still need metadata alignment.

## Per-Skill Sync Checklist

### Image

- Register `popiskill-image-text2image-basic-v1` in `popiartServer` with the same public metadata as skillhub.
- Register `popiskill-image-img2img-basic-v1` in `popiartServer` with the same public metadata as skillhub.
- Register `popiskill-image-img2img-popistudio-alice-showcase-v1` in `popiartServer` with the same public metadata as skillhub.

### Video

- Register `popiskill-video-image2video-basic-v1` in `popiartServer` with the same public metadata as skillhub.
- Remove placeholder behavior for `popiskill-video-image2video-basic-v1` so `popiartcli` no longer needs to overlay description or schema for discovery.
- Register `popiskill-video-image2video-popistudio-alice-showcase-v1` in `popiartServer` with the same public metadata as skillhub.

### Audio

- Register `popiskill-audio-tts-multimodel-v1` in `popiartServer` with the same public metadata as skillhub.
- Register `popiskill-audio-stt-local-v1` in `popiartServer` with the same public metadata as skillhub.

## Validation Steps

After server sync lands, validate in this order:

1. Raw registry check
   - `GET /skills?limit=100` should contain all seven ids.
2. CLI merged list check
   - `popiart skills list --limit 100` should show the same seven ids.
   - None of the seven should appear as `source=official-runtime`.
   - All seven should appear as `source=remote` unless an installed local skill intentionally overrides them.
3. Single-skill metadata check
   - `popiart skills get <skill-id>` should return remote metadata identical to skillhub for each of the seven skills.
4. Schema check
   - `popiart skills schema <skill-id>` should return the same schema shape as skillhub for each of the seven skills.
5. Execution smoke check
   - Run one minimal payload for each capability family:
     - `text2image`
     - `img2img`
     - `image2video`
     - `tts`
     - `stt`

## Exit Criteria

The sync is complete when all of the following are true:

- skillhub contains exactly these seven runtime skills
- `popiartServer /skills` returns the same seven runtime skills
- public metadata matches exactly for each overlapping id
- `popiartcli` no longer needs local official-runtime fallback for the four currently missing server entries
- `popiart skills list/get/schema` shows the seven skills as remote runtime skills in the configured environment
