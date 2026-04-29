---
name: popiskill-audio-tts-multimodel-v1
description: Convert text to speech through the PopiArt runtime baseline for multi-model TTS. Use this when the user wants one general text-to-speech entry point without handling upstream provider credentials directly.
tags:
  - official
  - runtime
  - audio
  - tts
  - multimodel
version: v1
model_type: audio
estimated_duration_s: 90
default_profile: true
profile_description: Official PopiArt runtime baseline for text-to-speech generation.
---

# PopiArt TTS Multimodel

This is an official PopiArt runtime catalog skill.

- `Popiart_skillhub` owns the public skill definition.
- `popiartServer` owns registration, execution, jobs, artifacts, and stable media URLs.
- `PopiNewAPI` owns provider routing and upstream voice models.
- `popiartcli` owns discovery, `run`, `jobs`, and artifact retrieval.

Use it when the goal is to:

- turn one text block into one spoken-audio result
- route through PopiArt's managed TTS stack instead of a raw provider SDK
- generate narration, prompts, or creator-agent audio previews

Do not use it for:

- speech-to-text transcription
- multi-step dubbing workflows
- direct provider-key experiments outside PopiArt

## Required input

- `text`: the content to speak

## Optional input

- `voice`
- `language`
- `provider`
- `voice_style`
- `speed`
- `emotion`
- `format`
- `sample_rate_hz`
- `seed`
- `notes`

## Workflow

1. Authenticate with a PopiArt product-layer key if needed.
2. Build the smallest valid JSON payload.
3. Run the skill through `popiart`.
4. Wait for completion.
5. Pull the returned audio artifact if a local file is needed.

## Command pattern

```sh
popiart run popiskill-audio-tts-multimodel-v1 --input @params.json --wait
```

Inline example:

```sh
popiart run popiskill-audio-tts-multimodel-v1 --input '{"text":"Hello, welcome to our product demo.","voice":"warm-female-cn","language":"zh-CN","format":"mp3"}' --wait
```

## Payload template

```json
{
  "text": "Hello, welcome to our product demo.",
  "voice": "warm-female-cn",
  "language": "zh-CN",
  "provider": "auto",
  "voice_style": "warm and clear",
  "speed": 1.0,
  "format": "mp3"
}
```

## Output handling

After the job finishes:

- read `job_id`
- inspect `artifact_ids`
- inspect `url` or `media_id` when a stable media URL is returned
- use `popiart artifacts pull <artifact-id>` to save the audio locally

## Operating guidance

- Use the PopiArt product-layer key only; do not ask the user for raw provider keys.
- Keep `provider` optional unless the caller explicitly needs to pin a managed route.
- If the user needs transcript text from an existing clip, switch to `popiskill-audio-stt-local-v1`.
