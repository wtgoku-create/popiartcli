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
  --input "{\"image_url\":\"$MEDIA_URL\",\"prompt\":\"turn this into a glossy product poster\"}" \
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
  --output json \
  --quiet \
  --non-interactive
```

纯音乐：

```sh
popiart music "Warm morning folk" \
  --instrumental \
  --output json \
  --quiet \
  --non-interactive
```

从歌词文件读取：

```sh
popiart music generate \
  --prompt "Upbeat summer pop" \
  --lyrics-file ./lyrics.txt \
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
