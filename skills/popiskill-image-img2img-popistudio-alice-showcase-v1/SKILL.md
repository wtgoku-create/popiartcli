---
name: popiskill-image-img2img-popistudio-alice-showcase-v1
description: Generate one showcase image for PopiStudio Alice with strict character consistency. Use this when a creator agent needs a single Alice proof frame, demo still, or presentation keyframe from the current Alice master reference.
tags:
  - official
  - runtime
  - image
  - img2img
  - showcase
  - alice
version: v1
model_type: image
estimated_duration_s: 180
default_profile: true
profile_description: Official PopiStudio Alice showcase still-image skill.
---

# PopiStudio Alice Image Showcase

This is an official PopiArt showcase runtime skill for the PopiStudio Alice character.

- `Popiart_skillhub` owns the public skill definition.
- `popiartServer` owns the Alice reference distribution for each environment, registration, jobs, artifacts, and stable media URLs.
- `PopiNewAPI` owns provider routing and image-generation channels.
- `popiartcli` owns discovery, `run`, `jobs`, and artifact retrieval.

Use it when the goal is to:

- generate one Alice showcase frame for demos or presentations
- keep Alice as the fixed protagonist across a new scene
- validate character-consistent img2img flow with the current Alice master reference

Do not use it for:

- text-only generation without an Alice reference
- changing Alice into a different protagonist
- multi-image batch generation
- long-form video output

## Required input

- `scene_prompt`: the new scene or action for Alice

## Source input

Prefer one Alice reference source:

- `source_artifact_id`: preferred when the current Alice master already exists as a PopiArt artifact
- `reference_image_url` or `image_url`: use the stable Alice media URL published by `popiartServer` for the current environment

## Optional input

- `shot_type`
- `camera`
- `mood`
- `aspect_ratio`
- `size`
- `seed`
- `retry_on_character_drift`
- `notes`

## Character guardrails

Preserve:

- Alice identity and recognizability
- hairstyle, hair color, and facial structure
- main clothing language and palette
- Alice as the visual center of the frame

## Workflow

1. Resolve the Alice reference from `source_artifact_id` or a stable Alice media URL.
2. Build one prompt that combines Alice consistency with the requested scene.
3. Run one img2img generation only.
4. Wait for completion.
5. Pull the artifact if a local file is needed.
6. Retry once only when character drift is obvious and the request explicitly allows it.

## Command pattern

```sh
popiart run popiskill-image-img2img-popistudio-alice-showcase-v1 --input @params.json --wait
```

Inline example:

```sh
popiart run popiskill-image-img2img-popistudio-alice-showcase-v1 --input '{"scene_prompt":"Alice waits outside a neighborhood convenience store at dusk, holding milk tea","reference_image_url":"https://media.example.com/popistudio/alice-master.jpg","shot_type":"medium shot","mood":"quiet and warm","aspect_ratio":"16:9"}' --wait
```

## Payload template

```json
{
  "scene_prompt": "Alice waits outside a neighborhood convenience store at dusk, holding milk tea",
  "reference_image_url": "https://media.example.com/popistudio/alice-master.jpg",
  "shot_type": "medium shot",
  "camera": "eye level",
  "mood": "quiet and warm",
  "aspect_ratio": "16:9",
  "retry_on_character_drift": true
}
```

## Output handling

- read `job_id`
- read `artifact_ids`
- inspect `url` or `media_id` when a stable media URL is returned
- use `popiart artifacts pull <artifact-id>` to save the showcase frame locally

## Operating guidance

- Prefer a current Alice master artifact over a raw external URL when both are available.
- Keep the request to one frame only.
- If the user wants a short motion teaser, switch to `popiskill-video-image2video-popistudio-alice-showcase-v1`.
- If the user wants a different protagonist, do not use this skill.
