---
name: popiskill-image-character-three-view-v1
description: Generate a consistent full-body character three-view sheet with front, side, and back views from a character brief, with optional reference image and optional extras such as items, palette, expressions, seasonal outfits, and action poses.
tags:
  - seed
  - local
  - image
  - character
  - three-view
version: v1
model_type: image
estimated_duration_s: 180
---

# Character Three View

Use this skill when the user asks for `三视图`, `角色三视图`, `front/side/back`, `turnaround`, or a clean full-body character sheet for design lock, model sheet review, or storyboard continuity.

This is a runtime catalog skill. The authoring file belongs in `Popiart_skillhub`, the runtime registration and execution belong to `popiartServer`, and the CLI path for discovery, run, jobs, and artifacts belongs to `popiartcli`.

## Required Input

- `character_prompt`: one concise but specific brief describing the same character identity across all views. Include silhouette, age range, outfit, key accessories, color direction, and style cues.

## Optional Input

- `reference_artifact_ids`: prior PopiArt artifact IDs to anchor likeness, outfit, palette, or line style.
- `style`: target visual style such as `anime`, `semi-realistic`, `stylized 3d`, or `storybook`.
- `background_mode`: `clean-card`, `plain`, or `transparent`.
- `pose_mode`: `neutral-a-pose`, `neutral-t-pose`, or `natural-standing`.
- `views`: defaults to `["front","side","back"]`.
- `include_items`: include a small prop cluster on the sheet.
- `include_palette`: include solid palette swatches.
- `expression_count`: integer from `0` to `10`.
- `include_seasonal_outfits`: include `winter` and `summer` outfit callouts.
- `action_count`: integer from `0` to `5`.
- `aspect_ratio`: recommended `4:5` or `3:4`.
- `notes`: extra constraints such as turnaround accuracy, no foreshortening, or production-safe line cleanup.

## Workflow

1. Normalize the brief into one stable character identity sheet.
2. Lock body proportion, costume breakdown, silhouette, and color direction before expanding to multiple views.
3. Generate full-body `front`, `side`, and `back` views with the same character, lighting logic, and render style.
4. If toggles are enabled, add sheet extras derived from the same identity:
   `items`, `palette`, `expressions`, `seasonal_outfits`, `actions`.
5. Return a clean character-sheet artifact suitable for review, plus any supporting metadata artifact the runtime emits.

## Payload Template

```json
{
  "character_prompt": "A cheerful fox-themed teenage courier girl with short orange hair, oversized cream hoodie, utility skirt, striped socks, canvas satchel, and a small bell charm. Clean anime linework, warm autumn palette, full-body turnaround sheet, consistent proportions, no dramatic perspective.",
  "style": "anime",
  "background_mode": "clean-card",
  "pose_mode": "neutral-a-pose",
  "views": ["front", "side", "back"],
  "include_items": true,
  "include_palette": true,
  "expression_count": 4,
  "include_seasonal_outfits": false,
  "action_count": 2,
  "aspect_ratio": "4:5",
  "notes": "Prioritize production-friendly readability and silhouette clarity over cinematic lighting."
}
```

## Real `popiart` Run Pattern

```sh
popiart run popiskill-image-character-three-view-v1 --input @three-view.params.json --wait
```

If the runtime registry uses a different internal ID, discover it first:

```sh
popiart skills list --search "three view"
popiart skills get <skill-id>
popiart skills schema <skill-id>
popiart run <skill-id> --input @three-view.params.json --wait
```

## Output Handling

After completion:

- read `job_id`
- read `artifact_ids`
- pull the sheet with `popiart artifacts pull <artifact-id>`
- pull all outputs with `popiart artifacts pull-all <job-id>`

Expected artifacts usually include:

- one main sheet image such as PNG or WEBP
- optional metadata JSON describing resolved views or prompt normalization

## When To Switch Skills

- Use a text-to-image skill instead when the user wants one hero image rather than a turnaround sheet.
- Use an image-to-image skill instead when the user already has a locked reference sheet and only needs refinement.
- Use a video skill instead when the user wants motion turnaround rather than front/side/back stills.
