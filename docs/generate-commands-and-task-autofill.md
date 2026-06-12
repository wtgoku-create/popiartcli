# Generate Commands And Task Autofill

Date: `2026-06-12`

This document summarizes the current generation-oriented `go run ./cmd/popiart ...` command surface and how `model/list` fields are auto-filled into `/api_client/anime/task/create`.

## Command Matrix

| Command | Capability | Task path | Notes |
| --- | --- | --- | --- |
| `popiart image [prompt]` | 文生图 | `type=1`, `subType=103` | root sugar for `image generate` |
| `popiart image generate` | 文生图 | `type=1`, `subType=103` | supports `--model` override |
| `popiart image img2img` | 图生图 | `type=1`, `subType=103` | source image required |
| `popiart image transform` | 图生图 | `type=1`, `subType=103` | alias of `img2img` |
| `popiart video [prompt]` | 图生视频 | `type=2`, current generic flow | root sugar for `video generate` |
| `popiart video generate` | 图生视频 | `type=2`, current generic flow | still not baseline-open for pure prompt text-to-video |
| `popiart video img2video` | 图生视频 | `type=2`, model-backed `subType` | explicit image-to-video entry |
| `popiart video from-image` | 图生视频 | `type=2`, model-backed `subType` | alias of `img2video` |
| `popiart video action-transfer` | 动作迁移 | `type=2`, `subType=205` | DreamActor-specific |
| `popiart video seedance` | Seedance 视频 | `type=2`, `subType=202/203/204` | dedicated Seedance flow |
| `popiart audio tts` | 文本转语音 | `type=3`, `subType=301` | official TTS flow |
| `popiart speech synthesize` | 文本转语音 | `type=3`, `subType=301` | alias of `audio tts` |
| `popiart music [prompt]` | 音乐生成 | `type=3`, `subType=304/305` | root sugar for `music generate` |
| `popiart music generate` | 音乐生成 | `type=3`, `subType=304/305` | model-backed music subtype |

## Autofill Mapping

When a command resolves a model from `/api_client/anime/ai/model/list?origin=web`, the CLI now auto-fills these task fields:

| Task field | Model source | Behavior |
| --- | --- | --- |
| `model` | `data[*].code` | always filled |
| `aiModelCode` | `data[*].code` | always filled |
| `aiModelCodeAlias` | `data[*].aiModelCodeAlias[0]` | first alias if present, otherwise falls back to `code` |
| `aiModelname` | `data[*].name` | normalized to task-compatible slug |
| `aiModelId` | `data[*].id` | always filled |
| `subType` | `data[*].categories[*].taskSubType` | chosen from command-compatible subtype set |
| `aspectRatio` / `ratio` | `data[*].ratio[]` or `data[*].videoRatio[]` | auto-filled when user omits ratio |
| `resolution` | `data[*].resolution[]` | auto-filled when user omits size |
| `duration` | `data[*].duration[]` | auto-filled for video when user omits duration |
| `width` / `height` | derived from `resolution + ratio` | generated after ratio/resolution selection |

## Validation Rules

Before creating a task, the CLI also validates model capability constraints from `model/list`:

| Validation | Model source | Current behavior |
| --- | --- | --- |
| subtype support | `categories[*].taskSubType` | command must match a supported subtype |
| image input limit | `uploadImageLimit` | rejects image-backed command when input count exceeds limit |
| video input limit | `uploadVideoLimit` | rejects when video count exceeds limit |
| audio input limit | `uploadAudioLimit` | rejects when audio count exceeds limit |
| image/video/audio support | `isSupportImages` / `isSupportVideos` / `isSupportAudios` | rejects unsupported media inputs |

Important:

- subtype support alone is not enough
- `uploadImageLimit=0` / `uploadVideoLimit=0` / `uploadAudioLimit=0` means unlimited, not forbidden
- only positive upload limits should trigger count-based rejection

## Current Real-World Baseline

As of `2026-06-12` in `https://wwwtest.popi.art`:

- stable success baseline:
  - `image generate`
  - `image img2img`
  - `video img2video`
  - `audio tts`
- partially validated:
  - `kling-video-o1` can create a `video img2video` task through generic task flow
- not yet baseline-open:
  - generic pure prompt `video generate`
  - music generation in this pass

## Related Files

- [internal/cmd/intent_commands.go](/Users/ywlmac/Projects/popiartcli/internal/cmd/intent_commands.go:1)
- [internal/popiart/mapper.go](/Users/ywlmac/Projects/popiartcli/internal/popiart/mapper.go:1)
- [internal/popiart/models.go](/Users/ywlmac/Projects/popiartcli/internal/popiart/models.go:1)
- [internal/cmd/intent_commands_test.go](/Users/ywlmac/Projects/popiartcli/internal/cmd/intent_commands_test.go:1)
