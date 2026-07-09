## wwwtest Baseline

Date: `2026-06-12`

Environment:
- endpoint: `https://wwwtest.popi.art`
- auth: test token provided by user
- cli mode: main-site `task/create` flow

This document records two different baselines and should not mix them:

1. current CLI default model candidates
2. models and commands that have been verified to run successfully in `wwwtest`

Model-related examples in the historical probe table may show the pre-unification behavior where `--model` accepted model code/name-like values and task payloads echoed `model` / `aiModelCode`. Current CLI usage is `--model <aiModelId>`, and `task/create` sends `aiModelId` as the only model identity field.

## Current CLI Defaults

These are the current default candidate pools in [internal/popiart/defaults.go](/Users/ywlmac/Projects/popiartcli/internal/popiart/defaults.go:3).

| Command family | Current default candidates | Notes |
| --- | --- | --- |
| `image`, `image generate` | `Nano-banana-pro`, `gemini-3-pro-image-preview`, `seedream-4-5-251128` | now prefers the model that has passed real `wwwtest` runs |
| `image img2img`, `image transform` | `Nano-banana-pro`, `gemini-3-pro-image-preview`, `seedream-4-5-251128` | same default pool as image generation |
| `video`, `video generate`, `video img2video`, `video from-image` | `viduq2-pro`, `viduq2-pro-fast` | now prefers the model that has passed real `wwwtest` image-to-video runs |
| `video seedance` | `doubao-seedance-2-0-260128` | kept as Seedance-specific default |
| `video action-transfer` | `jimeng_dreamactor_m20_gen_video` | not revalidated in this pass |
| `audio tts`, `speech synthesize` | `speech-2.8-hd` | real TTS success confirmed |
| `music`, `music generate` | `music-2.6`, `music-2.6-free` | default candidate pool; check latest real validations before tightening default further |

## Verified Successes In wwwtest

The following commands have been observed to complete with `status=2` in the real `wwwtest` environment.

| Capability | Command example | Resolved model | Task ID | Status |
| --- | --- | --- | --- | --- |
| text-to-image | `go run ./cmd/popiart image generate --prompt '猫头鹰' --wait` | `Nano-banana-pro` | `2235` | success |
| text-to-image | `go run ./cmd/popiart image generate --prompt '一只小狗' --aspect-ratio '9:16' --wait` | `Nano-banana-pro` | `2215` | success |
| text-to-image | `go run ./cmd/popiart image generate --prompt '猫头鹰'` | `Nano-banana-pro` | `2248` | success |
| image-to-image | real request shape aligned to `Nano-banana-pro`, `subType=103`, `ratio=9:16`, `resolution=2K` | `Nano-banana-pro` | `2227` | success |
| image-to-image | same request family, previous successful run | `Nano-banana-pro` | `2226` | success |
| image-to-video | request shape aligned to `viduq2-pro`, `subType=202`, `duration=5`, source image URL | `viduq2-pro` | `2223` | success |
| image-to-video | same request family, previous successful run | `viduq2-pro` | `2219` | success |
| image-to-video | `go run ./cmd/popiart video img2video --image 'https://popitest-public-1313913486.cos.ap-guangzhou.myqcloud.com/media/image/2026/0129/1914.png' --prompt '生成一个兔子奔跑的视频'` | `viduq2-pro` | `2251` | success |
| image-to-video | historical pre-unification probe with a code-like `--model` value; current CLI would require the corresponding `aiModelId` instead | `kling-video-omni` | `2260` | created, still running in latest probe |
| TTS audio generation | `go run ./cmd/popiart audio tts --text '我是一只小白兔，白白的小白兔' --wait` | `minimax-speech-2.8-hd` | `2231` | success |
| TTS audio generation | previous real TTS validation | `minimax-speech-2.8-hd` | `2209` | success |

## Request Shape Notes

### Image generation

The current successful image path depends on these request-shape rules:

- image generation and image-to-image both use `type=1`, `subType=103`
- when `--aspect-ratio` is omitted, CLI now explicitly sends default `aspectRatio=16:9` and `ratio=16:9`
- when model `resolution[]` is present and `--size` is omitted, CLI now selects a default supported resolution
- when model `ratio[]` is present and `--aspect-ratio` is omitted, CLI now prefers a model-declared ratio and canonicalizes service labels such as `21:9` instead of shrinking them to `7:3`
- `width` and `height` are now derived from `resolution + ratio`
- current successful `Nano-banana-pro` examples include:
  - `9:16 + 2K -> 2880x5120`
  - default no-ratio case -> explicit `16:9`

### Image-to-image

The current CLI sends both:

- `image`
- `images`

for single-image image tasks, to stay compatible with the successful request shape observed in `wwwtest`.

### Image-to-video

The historical successful `viduq2-pro` request family was closest to this old request shape. Current task creation strips the model code/name fields and keeps only `aiModelId`:

```json
{
  "projectId": -1,
  "type": 2,
  "chatPrompt": "生成一个兔子奔跑的视频",
  "styleId": 0,
  "width": 1280,
  "height": 720,
  "resolution": "720P",
  "aiModelId": 15,
  "subType": 202,
  "batchSize": 1,
  "duration": 5,
  "images": [
    "https://popitest-public-1313913486.cos.ap-guangzhou.myqcloud.com/media/image/2026/0129/1914.png"
  ]
}
```

Notes:

- `viduq2-pro` has real success records in `wwwtest`
- some other `202` models were seen failing due to third-party gateway file-fetch issues
- `uploadImageLimit=0` in `model/list` means unlimited; CLI should not reject these models before submit
- the successful historical task records for `viduq2-pro` in `wwwtest` do not always echo back all request-side fields such as `aspectRatio`

### Image-to-video model templates from `model/list`

As of `2026-06-12`, the current `image2video` capability slice in `wwwtest` suggests these minimum request templates:

| Model | Typical subtype window | Suggested minimal shape |
| --- | --- | --- |
| `viduq2-pro` | `202`, `204`; another record also exposes `203` | safest baseline is 1 source image + `subType=202` + `resolution=720P` + `duration=5` |
| `viduq3-turbo` | `202`, `204` | model-list capability template only; not the current baseline default |
| `kling-video-omni` | `202`, `203`, `204` | 1 source image + `subType=202` or `203` + ratio from `16:9/9:16` + `resolution=720P` + `duration=5` |
| `kling-v3` | `202` | 1 source image + `subType=202` + resolution from `1K/2K` + duration from model list; CLI should allow submit because `uploadImageLimit=0` means unlimited |
| `kling-v3-omni` | `202`, `203`, `204` | 1 source image + `subType=202` + ratio from `16:9/9:16` + `resolution=720P` + `duration=5` |
| `MiniMax-Hailuo-2.3-Fast` | `202` | CLI should allow submit when other required fields are auto-filled; real success still depends on upstream task execution |
| `seedance 2.0` | `202`, `203`, `204` | use dedicated `video seedance` flow; baseline request is ratio from model list + `resolution=480P/720P/1K` + `duration=5` |
| `viduq3-mix` | `203` only | not baseline-safe for generic prompt-only video; current real attempt without image fell into `reference-to-video requires at least 1 image` |
| `jimeng_dreamactor_m20_gen_video` | `205` only | use `video action-transfer`, not generic `video generate` |

## Not Yet Counted As Baseline Success

The following are not currently counted as baseline-success capabilities:

- pure prompt text-to-video
- music generation
- arbitrary `202` video models other than the now-confirmed `viduq2-pro` family

## Practical Guidance

- If the user does not pass `--model`, the CLI should now prefer defaults that have already succeeded at least once in `wwwtest` for:
  - image generation
  - image-to-image
  - image-to-video
  - TTS
- If the user does pass `--model`, CLI treats it as the main-site `aiModelId`, then uses that model's `model/list` capabilities to shape the request:
  - `aiModelId` as the only model identity field sent to `task/create`
  - subtype-compatible request shaping for music `304/305`
  - default `resolution` when supported and omitted
  - model-backed ratio selection when supported and omitted
  - derived `width` and `height`
- For generic `video generate`, the CLI still keeps the current safety boundary:
  - image-backed video requests can auto-fill model-backed `resolution`, `duration`, and compatible request fields
  - pure prompt video is still not treated as baseline-supported, because a real `wwwtest` run on `2026-06-12` returned `reference-to-video requires at least 1 image` for a `subType=203` fallback attempt
