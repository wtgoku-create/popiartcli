---
name: popiskill-image-text2image-basic-v1
description: Generate one image from text through the PopiArt runtime baseline. Use this when the user wants the most direct text-to-image path for smoke tests, concept frames, or simple creator-agent image generation.
tags:
  - official
  - runtime
  - image
  - text2image
  - basic
version: v1
model_type: image
estimated_duration_s: 120
default_profile: true
profile_description: Official PopiArt runtime baseline for single-image text-to-image generation.
---

# PopiArt Text To Image Basic

This is an official PopiArt runtime catalog skill.

- `Popiart_skillhub` owns the public skill definition.
- `popiartServer` owns registration, execution, jobs, artifacts, and stable media URLs.
- `PopiNewAPI` owns provider routing and upstream model access.
- `popiartcli` owns discovery, auth UX, `run`, `jobs`, and `artifacts`.

Use it when the goal is to:

- turn one prompt into one image
- validate that the PopiArt image path is alive end to end
- generate one quick concept frame without a multi-step workflow

Do not use it for:

- source-image editing
- multi-shot storyboards
- video output

## Required input

- `prompt`: the main image request

## Optional input

- `negative_prompt`
- `style`
- `size`
- `aspect_ratio`
- `seed`
- `notes`

## Workflow

1. Authenticate with a PopiArt product-layer key if needed.
2. Build the smallest valid JSON payload.
3. Run the skill through `popiart`.
4. Wait for completion.
5. Pull the returned artifact when a local file is needed.

## Command pattern

```sh
popiart run popiskill-image-text2image-basic-v1 --input @params.json --wait
```

Inline example:

```sh
popiart run popiskill-image-text2image-basic-v1 --input '{"prompt":"a cinematic tea shop at sunset","aspect_ratio":"16:9","style":"soft anime lighting"}' --wait
```

## Payload template

```json
{
  "prompt": "a cinematic tea shop at sunset",
  "style": "soft anime lighting",
  "aspect_ratio": "16:9",
  "seed": 42
}
```

## Output handling

After the job finishes:

- read `job_id`
- read `artifact_ids`
- inspect `url` or `media_id` when a stable media URL is returned
- use `popiart artifacts pull <artifact-id>` when a local file is needed

## Operating guidance

- Prefer `size` when the runtime contract already exposes exact dimensions.
- Use `aspect_ratio` when you want a portable request that the server can map safely.
- If the user needs source-image conditioning, switch to `popiskill-image-img2img-basic-v1`.
- If the user needs motion, switch to `popiskill-video-image2video-basic-v1`.
