---
name: popiskill-audio-stt-local-v1
description: Transcribe audio or video through the PopiArt runtime baseline for local-first speech-to-text. Use this when the user wants a PopiArt-managed STT path for transcripts, captions, or analysis without manually running local scripts.
tags:
  - official
  - runtime
  - audio
  - stt
  - local
version: v1
model_type: audio
estimated_duration_s: 120
default_profile: true
profile_description: Official PopiArt runtime baseline for local-first speech-to-text transcription.
---

# PopiArt STT Local

This is an official PopiArt runtime catalog skill.

- `Popiart_skillhub` owns the public skill definition.
- `popiartServer` owns registration, execution, jobs, artifacts, and transcript outputs.
- `PopiNewAPI` owns any managed speech-model routing that sits behind the server boundary.
- `popiartcli` owns source upload ergonomics, `run`, `jobs`, and artifact retrieval.

Use it when the goal is to:

- transcribe one audio or video source into text
- produce a quick transcript or caption draft through PopiArt
- keep the workflow inside PopiArt instead of manually running local STT scripts

Do not use it for:

- text-to-speech output
- full dubbing workflows
- direct provider-key experiments outside PopiArt

## Required input

- one media source:
  - `source_artifact_id`, or
  - `audio_url`, or
  - `video_url`

## Optional input

- `language`
- `backend`
- `model`
- `diarization`
- `timestamps`
- `format`
- `prompt`
- `notes`

## Workflow

1. Prefer `source_artifact_id` when the source clip already comes from PopiArt.
2. Use `audio_url` or `video_url` only when the source already lives at a stable URL.
3. Run the skill through `popiart`.
4. Wait for completion.
5. Pull transcript artifacts or read direct text fields from the result.

## Command pattern

```sh
popiart run popiskill-audio-stt-local-v1 --input @params.json --wait
```

Inline example:

```sh
popiart run popiskill-audio-stt-local-v1 --input '{"source_artifact_id":"art_123","language":"zh","timestamps":true}' --wait
```

## Payload template

```json
{
  "source_artifact_id": "art_123",
  "language": "zh",
  "backend": "auto",
  "model": "default",
  "diarization": false,
  "timestamps": true,
  "format": "text"
}
```

## Output handling

After the job finishes:

- read `job_id`
- inspect `text` when the runtime returns inline transcript text
- inspect `artifact_ids` for transcript, subtitle, or segment artifacts
- use `popiart artifacts pull <artifact-id>` when a local transcript file is needed

## Operating guidance

- For local media files, upload first with `popiart artifacts upload ./clip.wav --role source`.
- `video_url` is acceptable when the runtime extracts audio server-side.
- If the user needs spoken audio output instead, switch to `popiskill-audio-tts-multimodel-v1`.
