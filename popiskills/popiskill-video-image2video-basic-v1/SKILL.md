---
name: popiskill-video-image2video-basic-v1
description: Turn one source image into a short video through the PopiArt runtime baseline. Use this when the user wants the most direct image-to-video path for motion previews, short teaser clips, or runtime validation.
tags:
  - official
  - runtime
  - video
  - image2video
  - basic
version: v1
model_type: video
estimated_duration_s: 180
default_profile: true
profile_description: Official PopiArt runtime baseline for single-shot image-to-video generation.
---

# PopiArt Image To Video Basic

This is an official PopiArt runtime catalog skill.

- `Popiart_skillhub` owns the public skill definition.
- `popiartServer` owns source resolution, jobs, artifacts, and stable media URLs.
- `PopiNewAPI` owns provider routing and image-to-video model access.
- `popiartcli` owns discovery, `run`, `jobs`, `artifacts`, and the current built-in direct-fallback bridge for this baseline skill when the remote catalog entry is still a placeholder.

Use it when the goal is to:

- animate one still image into one short clip
- validate the basic PopiArt video path
- generate a quick motion preview from an artifact or image URL

Do not use it for:

- long-form editing
- multi-shot generation
- text-only video generation

## Required input

- one image source:
  - `source_artifact_id`, or
  - `image_url`, or
  - `reference_image_url`

## Optional input

- `prompt`
- `negative_prompt`
- `duration_s`
- `seconds`
- `fps`
- `camera_motion`
- `motion_intensity`
- `style`
- `aspect_ratio`
- `seed`
- `notes`

## Workflow

1. Prefer `source_artifact_id` when the source image already comes from PopiArt.
2. Use `image_url` only when the source already lives at a stable URL.
3. Keep the clip short and the motion instruction singular.
4. Run through `popiart` and wait for the job.
5. Pull the returned artifact when a local file is needed.

## Command pattern

```sh
popiart run popiskill-video-image2video-basic-v1 --input @params.json --wait
```

Inline example:

```sh
popiart run popiskill-video-image2video-basic-v1 --input '{"source_artifact_id":"art_123","prompt":"the camera slowly pushes in while the hair moves in the wind","duration_s":5}' --wait
```

## Payload template

```json
{
  "source_artifact_id": "art_123",
  "prompt": "the camera slowly pushes in while the hair moves in the wind",
  "duration_s": 5,
  "camera_motion": "slow push-in",
  "seed": 42
}
```

## Output handling

After the job finishes:

- read `job_id`
- inspect `artifact_ids`
- inspect `execution_mode` and `model_id` when the CLI had to use the built-in direct-fallback bridge
- use `popiart artifacts pull <artifact-id>` to save the video locally

## Operating guidance

- For local source files, upload first with `popiart artifacts upload ./source.png --role source`.
- `reference_image_url` is a compatibility alias for `image_url`.
- `seconds` is a compatibility alias for `duration_s`.
- If the user only has text, run `popiskill-image-text2image-basic-v1` first.
