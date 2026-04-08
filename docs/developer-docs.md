# PopiArt CLI 开发者文档

`popiart` 是面向 Coding Agent 的创作者技能 CLI。
它既可以作为独立命令行工具在终端中使用，也可以作为 Codex、Claude Code、OpenClaw、OpenCode 等 agent 的统一技能入口。

`popiart` 负责四件事：

- 发现可用的创作者 skill
- 提交 skill 执行并跟踪 job 生命周期
- 拉取运行结果 artifacts
- 为多模态能力统一处理项目上下文、授权、路由和计费

如果你想先理解系统边界，而不是立即接入，请先看 [docs/project-relationship.md](./project-relationship.md)。

## 快速开始

安装 `popiart`：

```sh
# macOS / Linux
brew tap wtgoku-create/popi
brew install wtgoku-create/popi/popiart
```

或使用安装脚本：

```sh
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | sh
```

配置 API key 并检查身份：

```sh
popiart auth login --key pk-...
popiart auth whoami
```

选择当前项目：

```sh
popiart project list
popiart project use <project-id>
popiart project current
```

为 Coding Agent 生成 discoverability 资产：

```sh
popiart bootstrap --agent codex --completion zsh --discoverable
```

这条命令现在不只是生成 `~/.popiart/agents/codex/` 下的中间资产，还会把 `PopiArt` 直接写进 agent 的原生 MCP 和 skill 目录。

如果你只想先验证 CLI 是否可用，可以直接运行：

```sh
popiart --help
popiart skills list
```

## 能力一览

### 技能发现

```sh
popiart skills list
```

列出当前可用的远程 runtime skills，并合并 CLI 内置的 bundled seed skills。

```sh
popiart skills list --tag image
```

按标签筛选，例如只看图像类技能。

```sh
popiart skills list --search "three view"
```

按关键词全文搜索技能。

```sh
popiart skills get <skill-id>
```

查看某个技能的完整说明、输入输出约束和描述信息。

```sh
popiart skills schema <skill-id>
```

查看技能的输入/输出 JSON schema，适合在 agent 或脚本里生成稳定 payload。

### 本地 skill 安装

```sh
popiart skills pull <skill-id> --url https://example.com/skill.zip
popiart skills install ./skill.zip
popiart skills use-local <skill-id>
```

当前本地安装 skill 的最小执行模型是：

- skill 以 zip 包分发
- 包内包含 `SKILL.md`
- 包内包含 `popiart-skill.yaml` / `popiart-skill.json`，或 `SKILL.md` 顶部 YAML frontmatter
- `popiart run` 当前只支持 `execution.mode=remote-runtime`
- `skills pull/install` 暂不支持 `.tar.gz`、目录安装、GitHub 页面 URL 自动解析

安装后：

- skill 会合并进 `skills list/get/schema`
- 若该本地 skill 未与远端同名 skill 冲突，`popiart run <slug>` 可直接使用
- 若与远端同名，可执行 `popiart skills use-local <slug>` 切换为本地优先

如果需要给 agent 直接放到原生 skills 目录：

```sh
popiart skills install ./skill.zip --agent codex
popiart skills install ./skill.zip --agent claude-code
popiart skills install ./skill.zip --agent openclaw
popiart skills install ./skill.zip --agent opencode
```

默认原生路径是：

- `codex`: `~/.codex/skills/`
- `claude-code`: `~/.claude/skills/`
- `openclaw`: `~/.openclaw/skills/`
- `opencode`: `~/.config/opencode/skill/`

### 技能执行

```sh
popiart run <skill-id> --input '{"prompt":"a sunset over Tokyo"}'
```

提交一个 skill 执行任务，默认立即返回 `job_id`。

```sh
popiart run <skill-id> --input @params.json --wait
```

从文件读取输入并阻塞等待任务完成。

```sh
cat params.json | popiart run <skill-id> --input -
```

从标准输入读取 JSON payload，适合 shell pipeline 和 agent 自动化。

```sh
popiart run <skill-id> --input @params.json --idempotency-key req-20260327-001
```

使用幂等键安全重试，避免网络抖动或 agent 重放时重复扣费。

### 作业管理

```sh
popiart jobs get <job-id>
```

获取作业当前状态。

```sh
popiart jobs wait <job-id>
```

轮询直到作业到达终止状态。

```sh
popiart jobs list --status running
```

查看近期作业，并按状态、技能或项目过滤。

```sh
popiart jobs logs <job-id>
```

查看作业日志。

```sh
popiart jobs logs <job-id> --follow
```

流式跟踪作业日志，适合长任务调试。

### 工件拉取

```sh
popiart artifacts list <job-id>
```

列出某个 job 产出的 artifacts。

```sh
popiart artifacts upload ./source.png --role source
```

上传一个本地文件并创建可复用 artifact，适合在 agent 聊天附件进入 `img2img` 前先做归档。

```sh
popiart artifacts pull <artifact-id>
```

下载单个 artifact 到本地磁盘。

```sh
popiart artifacts pull-all <job-id>
```

将一个 job 的全部 artifacts 一次性下载到目录中。

### 稳定媒体 URL

```sh
popiart media upload ./source.png
```

上传一个本地文件并获取稳定媒体 URL。这个命令适合下面两类场景：

- 你只想把本地文件变成一个可被模型直接 fetch 的 URL
- 你在 job 外部先做素材准备，再把 URL 传给后续 skill

当服务端支持稳定媒体 URL 时，`popiart artifacts upload` 也会在响应里返回 `url`，因此 artifact 既保留 PopiArt 的工件语义，也具备直接给模型消费的 URL 语义。

如果要把本地图片交给 `img2img`，建议走这条链路：

```sh
ART=$(popiart artifacts upload ./source.png --role source | jq -r '.data.artifact_id')

popiart run popiskill-image-img2img-basic-v1 --input "{
  \"source_artifact_id\":\"$ART\",
  \"prompt\":\"保留主体，改成黄昏电影感\"
}" --wait
```

当前已经在测试环境验证通过的服务端图像编辑适配包括：

- `gemini-3-pro-image-preview`
- `seedream-4-5-251128`

其中 `seedream-4-5-251128` 不是走旧的 `/v1/images/edits multipart` 语义，而是走 `/v1/images/generations` + 参考图输入；最小尺寸约束也由服务端路由适配负责处理。

`popiskill-video-image2video-basic-v1` 是安装后自带的官方 skill。CLI 会把它暴露在 `skills list/get/schema` 里；如果远端目录里的同名 skill 仍是占位符或尚未注册，`run` 会自动桥接到底层 `models infer`，优先使用 `viduq3-turbo`，失败再回落到 `viduq2-pro-fast`。

如果要把本地图片继续交给 `image2video`，推荐同样先上传成 artifact，再走 `source_artifact_id`：

```sh
ART=$(popiart artifacts upload ./source.png --role source | jq -r '.data.artifact_id')

popiart run popiskill-video-image2video-basic-v1 --project proj_local_dev --input "{
  \"source_artifact_id\":\"$ART\",
  \"prompt\":\"让人物衣摆和发丝在微风中轻轻摆动，镜头缓慢推进，整体保持真实电影感。\",
  \"aspect_ratio\":\"16:9\",
  \"seconds\":4
}" --wait
```

截至 `2026-04-08`，CLI 内置 `image2video` fallback 的模型顺序是 `viduq3-turbo -> viduq2-pro-fast`。截至 `2026-03-28`，测试环境里验证通过的服务端 `image2video` 路由是 `video.image2video -> viduq2-pro-fast`。如果服务端将来补齐真正 runtime skill，CLI 会优先走服务端 skill；否则继续走内置 fallback。

稳定媒体 URL 的完整跨仓架构与分阶段执行计划见 [docs/stable-media-url-v1.md](./stable-media-url-v1.md)。

### 项目上下文

```sh
popiart project current
```

查看当前活动项目。

```sh
popiart project context
```

读取当前项目的完整运行时上下文。

```sh
popiart project use <project-id>
```

切换当前项目，后续 `run`、`models infer`、预算查询都会继承这个上下文。

### 模型直连

```sh
popiart models list --type image
popiart models list --capability text2image
```

列出当前已注册的可用模型库存。

```sh
popiart models routes
popiart models routes --route image.text2image
```

查看当前项目真正生效的 `route_key -> model_id` 路由表。
这个结果和 `models list` 的模型库存不是一回事。

```sh
popiart models infer <model-id> --input @input.json --wait
```

直接提交模型推理任务，不经过 skill 封装，适合做底层能力验证或路由调试。

### MCP 接入

```sh
popiart mcp serve --describe
```

打印当前 MCP server 的工具面和元数据。

```sh
popiart mcp print-config --agent codex
```

生成通用 MCP 配置片段，方便接入不同 agent。

```sh
popiart mcp doctor --agent codex
```

检查本地 discoverability 资产、认证状态和 runtime baseline 准备情况。

对于聊天附件场景，MCP 侧新增了 `upload_artifact`。宿主先把附件保存到本地路径，再调用：

```json
{
  "path": "/tmp/chat-upload.png",
  "role": "source"
}
```

返回的 `artifact_id` 可以直接填到 `run_skill.input.source_artifact_id`。

### 预算与配额

```sh
popiart budget status
```

查看当前周期的预算摘要。

```sh
popiart budget usage --group-by skill
```

按技能、日期或项目查看详细使用情况。

```sh
popiart budget limits
```

查看速率限制和配额配置。

## 官方 Runtime Baseline

当前仓库将以下三个 skill id 视为首批官方 runtime baseline：

```text
popiskill-image-text2image-basic-v1
popiskill-image-img2img-basic-v1
popiskill-video-image2video-basic-v1
```

其中 `popiskill-video-image2video-basic-v1` 同时是安装后自带的内置官方 skill：即使远端目录缺失或仍返回占位符，CLI 也会返回本地契约，并在运行时自动桥接到底层 image2video 模型。

你可以这样做一次最小验证：

```sh
popiart skills get popiskill-image-text2image-basic-v1
popiart skills schema popiskill-image-text2image-basic-v1
popiart run popiskill-image-text2image-basic-v1 --input @params.json --wait
```

任务完成后，拉取结果：

```sh
popiart artifacts pull-all <job-id>
```

## 推荐工作流

如果你是在终端里手动使用 `popiart`，推荐顺序是：

1. `popiart auth login --key pk-...`
2. `popiart project use <project-id>`
3. `popiart skills list --search "<keyword>"`
4. `popiart skills schema <skill-id>`
5. `popiart run <skill-id> --input @params.json --wait`
6. `popiart artifacts pull-all <job-id>`

如果你是在 agent 中接入 `popiart`，推荐顺序是：

1. `popiart bootstrap --agent codex --completion zsh --discoverable`
2. `popiart mcp doctor --agent codex`
3. 在 agent 里通过 MCP 工具发现 skill、提交 job、轮询状态、拉取 artifact

## 面向 Agent 的设计原则

`popiart` 的默认行为不是“终端友好优先”，而是“agent 可解析优先”：

- 默认输出 JSON，便于脚本和 agent 稳定消费
- 每次失败都带 `error.code`
- 长任务统一返回 `job_id`
- `--input` 支持内联 JSON、`@file.json` 和标准输入
- artifacts 与 jobs 分离，便于重试、恢复和二次拉取

如果你只是让人类阅读输出，可以在命令前加 `--plain`：

```sh
popiart --plain skills list --tag image
```

## 相关文档

- 安装与首次使用：[docs/install-and-usage.md](./install-and-usage.md)
- 项目边界说明：[docs/project-relationship.md](./project-relationship.md)
- MCP discoverability 设计：[docs/mcp-discoverability-v1.md](./mcp-discoverability-v1.md)
- 当前实现状态：[docs/current-status.md](./current-status.md)
- 发布维护说明：[docs/releasing.md](./releasing.md)
