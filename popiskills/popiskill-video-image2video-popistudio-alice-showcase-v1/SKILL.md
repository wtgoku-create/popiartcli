---
name: popiskill-video-image2video-popistudio-alice-showcase-v1
description: Generate one short showcase video for PopiStudio Alice with strict character consistency. Use this when a creator agent needs a single Alice teaser shot, demo clip, or animated proof video from the current Alice master reference or Alice keyframe artifact.
tags:
  - official
  - runtime
  - video
  - image2video
  - showcase
  - alice
version: v1
model_type: video
estimated_duration_s: 210
default_profile: true
profile_description: Official PopiStudio Alice showcase motion skill.
---

# PopiStudio Alice Video Showcase

This is an official PopiArt showcase runtime skill for the PopiStudio Alice character.

- `Popiart_skillhub` owns the public skill definition.
- `popiartServer` owns the Alice reference distribution for each environment, jobs, artifacts, and stable media URLs.
- `PopiNewAPI` owns provider routing and image-to-video channels.
- `popiartcli` owns discovery, `run`, `jobs`, and artifact retrieval.

Use it when the goal is to:

- turn one Alice frame into one short motion clip
- create a teaser shot or demo clip for PopiStudio
- validate Alice character consistency in a simple image-to-video flow

Do not use it for:

- long-form editing
- multi-shot sequences
- text-only video generation
- changing Alice into another protagonist

## Source input

Prefer one Alice source:

- `source_artifact_id`: preferred when the current Alice keyframe already exists as a PopiArt artifact
- `image_url` or `reference_image_url`: use the stable Alice media URL published by `popiartServer` for the current environment

## Optional input

- `motion_prompt`
- `duration_s`
- `seconds`
- `camera_motion`
- `mood`
- `aspect_ratio`
- `seed`
- `retry_on_character_drift`
- `notes`

## Character guardrails

Preserve:

- Alice identity and recognizability
- hairstyle, hair color, and facial structure
- main clothing language and palette
- Alice as the clear protagonist through the shot

## Workflow

1. Resolve the Alice source from `source_artifact_id` or a stable Alice media URL.
2. Build one short motion prompt that keeps Alice stable and the shot readable.
3. Run one image-to-video generation only.
4. Wait for completion.
5. Pull the returned artifact when a local file is needed.
6. Retry once only when character drift is obvious and the request explicitly allows it.

## Command pattern

```sh
popiart run popiskill-video-image2video-popistudio-alice-showcase-v1 --input @params.json --wait
```

Inline example:

```sh
popiart run popiskill-video-image2video-popistudio-alice-showcase-v1 --input '{"image_url":"https://media.example.com/popistudio/alice-master.jpg","motion_prompt":"Alice looks up and smiles while the camera slowly pushes in","duration_s":5,"camera_motion":"slow push-in","aspect_ratio":"16:9"}' --wait
```

## Payload template

```json
{
  "image_url": "https://media.example.com/popistudio/alice-master.jpg",
  "motion_prompt": "Alice looks up and smiles while the camera slowly pushes in",
  "duration_s": 5,
  "camera_motion": "slow push-in",
  "mood": "warm and hopeful",
  "aspect_ratio": "16:9",
  "retry_on_character_drift": true
}
```

## Output handling

- read `job_id`
- read `artifact_ids`
- inspect `url` or `media_id` when a stable media URL is returned
- use `popiart artifacts pull <artifact-id>` to save the video locally

## Operating guidance

- Keep the clip short and the motion instruction singular.
- Prefer a current Alice keyframe artifact over a raw external URL when both are available.
- If the user first needs a still frame, run `popiskill-image-img2img-popistudio-alice-showcase-v1`.
