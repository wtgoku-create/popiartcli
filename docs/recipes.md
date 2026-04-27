# PopiArt Recipes

这份文档面向人类用户和 Coding Agent，提供可组合、可管道化的标准调用模式。

## stdout / stderr 约定

推荐在 agent / CI 环境中统一使用：

```sh
--output json --quiet --non-interactive
```

约定如下：

- stdout：只放结果数据或 dry-run 预览
- stderr：放提示、诊断、交互提示
- `--output json`：输出 `{ ok, data }` 或 `{ ok, error }`
- `--output plain`：输出人类可读文本
- `--dry-run`：输出规范化后的请求预览，不执行网络写操作

## 配置优先级

当前配置优先级是：

1. CLI flags
2. 环境变量
3. `~/.popiart/config.json`
4. 默认值

常用环境变量：

```sh
export POPIART_ENDPOINT=https://api.creatoragentos.io/v1
export POPIART_KEY=<product-key>
export POPIART_PROJECT=<project-id>
```

## 3 分钟接入 agent

```sh
popiart setup --agent codex --completion zsh
popiart auth login --key <product-key>
popiart image generate \
  --prompt "A cinematic portrait of a creator at sunset" \
  --output json \
  --quiet \
  --non-interactive
```

## Recipe: artifact -> run -> wait -> pull

适合显式使用平台命令面时的标准 job 流程。

```sh
ARTIFACT_JSON=$(popiart artifacts upload ./source.png \
  --role source \
  --output json \
  --quiet \
  --non-interactive)

ARTIFACT_ID=$(printf '%s' "$ARTIFACT_JSON" | jq -r '.data.artifact_id')

JOB_JSON=$(popiart run popiskill-video-image2video-basic-v1 \
  --input "{\"source_artifact_id\":\"$ARTIFACT_ID\",\"prompt\":\"slow push-in\"}" \
  --output json \
  --quiet \
  --non-interactive)

JOB_ID=$(printf '%s' "$JOB_JSON" | jq -r '.data.job_id')

popiart jobs wait "$JOB_ID" \
  --output json \
  --quiet \
  --non-interactive

popiart artifacts pull-all "$JOB_ID" \
  --output json \
  --quiet \
  --non-interactive
```

## Recipe: media upload -> img2img

适合先拿稳定 URL，再把 URL 交给下游 skill。

```sh
MEDIA_JSON=$(popiart media upload ./poster.png \
  --visibility unlisted \
  --output json \
  --quiet \
  --non-interactive)

MEDIA_URL=$(printf '%s' "$MEDIA_JSON" | jq -r '.data.url')

popiart run popiskill-image-img2img-basic-v1 \
  --input "{\"image\":\"$MEDIA_URL\",\"prompt\":\"turn this into a glossy product poster\"}" \
  --output json \
  --quiet \
  --non-interactive
```

## Recipe: intent-first image generation

```sh
popiart image generate \
  --prompt "A fashion editorial portrait in warm sunset light" \
  --aspect-ratio 9:16 \
  --output json \
  --quiet \
  --non-interactive
```

## Recipe: intent-first img2img

```sh
popiart image img2img \
  --image ./source.png \
  --prompt "Turn this into a watercolor illustration" \
  --strength 0.6 \
  --output json \
  --quiet \
  --non-interactive
```

## Recipe: intent-first video generation from local file

`video generate` 会自动完成本地文件上传，再提交 runtime job。

```sh
popiart video generate \
  --image ./source.png \
  --prompt "Slow push-in and subtle wind motion" \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

## Recipe: explicit img2video

如果你希望命令名直接体现 image-to-video，而不是通用 generate：

```sh
popiart video img2video \
  --image ./source.png \
  --prompt "Slow push-in and subtle wind motion" \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

如果你只想预览，不想真的提交：

```sh
popiart video generate \
  --image ./source.png \
  --prompt "Slow push-in and subtle wind motion" \
  --dry-run \
  --output json \
  --quiet \
  --non-interactive
```

## Recipe: Jimeng action transfer

即梦动作迁移需要一张身份图和一个动作参考视频。CLI 默认使用 `jimeng_dreamactor_m20_gen_video`，并提交 `metadata.action=actionGenerate`。

```sh
popiart video action-transfer \
  --image ./face.jpg \
  --video https://example.com/source-action.mp4 \
  --cut-result-first-second-switch \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

如果 `--image` 传入 `data:image/jpeg;base64,...`，CLI 会自动剥离 data URL 前缀，按即梦上游要求只提交纯 base64。

本地文件也可以直接传入；CLI 会先上传为 stable media URL，再提交给服务端：

```sh
popiart video action-transfer \
  --image ./face.jpg \
  --video ./source-action.mp4 \
  --cut-result-first-second-switch \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

成功返回 `artifact_ids` 后，可拉取结果：

```sh
popiart artifacts pull <artifact-id> --out ./action-transfer-preview.mp4
```

测试服验证过的 5 秒预览形态：

- 输入：一张身份图 + 一个约 5 秒动作参考视频
- 输出：MP4 artifact
- 示例输出参数：约 5.04 秒，H.264 + AAC

## Recipe: Seedance video

Seedance / 豆包视频建议使用专门命令面。默认模型是 `doubao-seedance-2-0-260128`。
CLI 会直接提交到统一网关 `POST /v1/video/generations`，不会再包一层 `models/infer` 输入。

文生视频需要 `--prompt`：

```sh
popiart video seedance \
  --prompt "一只猫追蝴蝶" \
  --ratio 16:9 \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

参考图、首尾帧、参考图和参考视频模式下 `--prompt` 可选；只传 `--audio` 不合法，参考音频必须同时搭配图片或视频。

参考视频：

```sh
popiart video seedance \
  --prompt "保持主体动作风格一致" \
  --video https://example.com/ref.mp4 \
  --ratio 16:9 \
  --return-last-frame \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

带 Seedance 2.0 扩展 metadata：

```sh
popiart video seedance \
  --prompt "保持主体动作风格一致" \
  --video https://example.com/ref.mp4 \
  --frames 120 \
  --ratio 16:9 \
  --return-last-frame \
  --generate-audio \
  --tools-json '[{"type":"camera_control"}]' \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

`--wait` 会查询 `GET /v1/video/generations/{task_id}`。如果网关返回 `metadata.url` / `metadata.last_frame_url`，CLI 会同时透出 `result_url` / `last_frame_url`。

本地文件也可以直接传入；CLI 会先上传成 stable media URL：

```sh
popiart video seedance \
  --prompt "多图融合参考" \
  --image ./a.jpg \
  --image ./b.jpg \
  --video ./ref.mp4 \
  --audio ./ref.mp3 \
  --generate-audio \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

## Recipe: text-to-speech

```sh
popiart speech synthesize \
  --text "今天我们来做一个更适合 agent 调用的 CLI。" \
  --voice narrator_female \
  --format mp3 \
  --output json \
  --quiet \
  --non-interactive
```

也可以从文件读取：

```sh
popiart speech synthesize \
  --text-file ./speech.txt \
  --format mp3 \
  --output json \
  --quiet \
  --non-interactive
```

## Recipe: MiniMax music generate

```sh
popiart music generate \
  --prompt "Upbeat pop" \
  --lyrics "La la la" \
  --output-format url \
  --format mp3 \
  --output json \
  --quiet \
  --non-interactive
```

纯音乐：

```sh
popiart music "Warm morning folk" \
  --instrumental \
  --output-format url \
  --format mp3 \
  --output json \
  --quiet \
  --non-interactive
```

从歌词文件读取：

```sh
popiart music generate \
  --prompt "Upbeat summer pop" \
  --lyrics-file ./lyrics.txt \
  --output-format url \
  --format mp3 \
  --output json \
  --quiet \
  --non-interactive
```

`--output-format` maps to the MiniMax gateway `output_format` (`hex` or `url`). `--format`,
`--sample-rate-hz`, and `--bitrate` are sent under the gateway `audio_setting` object.

## Recipe: MiniMax image generate

```sh
popiart image generate \
  --model image-01 \
  --prompt "A studio portrait of a corgi" \
  --aspect-ratio 4:3 \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

## Recipe: MiniMax img2img

```sh
popiart image img2img \
  --model image-01 \
  --image https://example.com/reference.jpg \
  --prompt "Turn this into a poster-style portrait" \
  --aspect-ratio 3:4 \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

## Recipe: Hailuo video via MiniMax models

文生视频：

```sh
popiart models infer MiniMax-Hailuo-2.3 \
  --input '{"prompt":"A person picks up a book and reads it quietly.","duration":6,"size":"1080P"}' \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

图生视频：

```sh
popiart models infer MiniMax-Hailuo-2.3 \
  --input '{"prompt":"The subject smiles and blinks while the camera slowly pushes in.","duration":6,"size":"1080P","images":["https://example.com/reference.jpg"]}' \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

首尾帧视频：

```sh
popiart models infer MiniMax-Hailuo-02 \
  --input '{"prompt":"A little girl grows up into adulthood.","duration":6,"size":"768P","images":["https://example.com/first-frame.jpg","https://example.com/last-frame.jpg"]}' \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

## Recipe: schema-first execution

当 agent 需要先理解 skill 契约，再构造输入时：

```sh
popiart skills get popiskill-image-text2image-basic-v1 \
  --output json \
  --quiet \
  --non-interactive

popiart skills schema popiskill-image-text2image-basic-v1 \
  --output json \
  --quiet \
  --non-interactive
```

然后再执行：

```sh
popiart run popiskill-image-text2image-basic-v1 \
  --input @params.json \
  --output json \
  --quiet \
  --non-interactive
```

## Recipe: discoverability / MCP

```sh
popiart setup --agent codex --completion zsh

popiart mcp print-config --agent codex \
  --output json \
  --quiet \
  --non-interactive

popiart mcp doctor --agent codex \
  --output json \
  --quiet \
  --non-interactive
```
