# popiart

**面向 Coding Agent 的创作者技能 CLI。**

这是 `popiart` 的第一个版本。

为 OpenClaw、Claude Code、OpenCode 等
coding agent 提供一个统一的技能入口。

用户安装 `popiart` 后，可以围绕创作者 `skillhub.popi.art` 中的 skillhub 完成查看、发现、调用；
当这些 skill 需要使用多模态模型能力时，再由 `popiart` 统一处理授权、
鉴权、路由和计费。

---

## 项目关系

先看 [docs/project-relationship.md](./docs/project-relationship.md)。

这份文档说明 `popiartcli`、`popiartServer`、`PopiNewAPI` 三层之间的职责边界，以及为什么 CLI 只负责统一入口和本地配置，不直接持有上游模型 key。

---

## 当前版本做什么

- 发现和查看可用的创作者 skill
- 通过统一 CLI 调用这些 skill
- 当 skill 依赖图像、视频、动画等多模态模型时，统一处理授权和计费
- 用一致的作业、工件、轮询、预算模型，降低 agent 接入复杂度

## 它不做什么

- 不要求所有能力都由 CLI 自己直接实现
- 不试图替代每一个创作者 skill 的具体业务逻辑
- 不把底层模型调用细节暴露给每个 agent 单独处理

---

## 底层接入

当某个 skill 需要调用多模态模型时，`popiart` 会通过同一个 token
网关连接多种模型与平台，例如 Gemini、Veo、Sora、Vidu、Kling、
Runway 等。

后端基于 GitHub 上的 `newapi` 体系做统一接入，可按用户生成 key，
并在服务端完成授权、鉴权、路由和统一计费。

---

## 安装

完整的平台安装、首次使用和 agent 接入说明见 [docs/install-and-usage.md](./docs/install-and-usage.md)。
一页式开发者总览见 [docs/developer-docs.md](./docs/developer-docs.md)。
MCP discoverability 与 runtime baseline 设计见 [docs/mcp-discoverability-v1.md](./docs/mcp-discoverability-v1.md)。
稳定媒体 URL 的 V1 架构与分阶段计划见 [docs/stable-media-url-v1.md](./docs/stable-media-url-v1.md)。
当前仓库实际落地状态见 [docs/current-status.md](./docs/current-status.md)。

```sh
# Homebrew (macOS / Linux)
brew tap wtgoku-create/popi
brew install wtgoku-create/popi/popiart

# 后续升级
brew upgrade wtgoku-create/popi/popiart

# 安装完成后，按需执行生态引导
popiart bootstrap --agent codex --completion zsh

# 如果希望安装后直接在 agent 的 MCP / skill 目录中发现 PopiArt
popiart bootstrap --agent codex --discoverable
```

`popiart bootstrap --discoverable` 现在会同时做两件事：

- 继续在 `~/.popiart/agents/<agent>/` 下生成 bootstrap 资产
- 直接写入对应 agent 的原生 MCP 配置和原生 skill 目录

当前默认落点是：

- `codex`: `~/.codex/config.toml` 和 `~/.codex/skills/popiart/`
- `claude-code`: `~/.claude.json` 和 `~/.claude/skills/popiart/`
- `openclaw`: `~/.openclaw/mcp.json` 和 `~/.openclaw/skills/popiart/`
- `opencode`: `~/.config/opencode/mcp.json` 和 `~/.config/opencode/skill/popiart/`

```sh
# Windows PowerShell
irm https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.ps1 | iex

# 安装指定版本
$env:VERSION="v0.3.2"; irm https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.ps1 | iex
```

```sh
# 一键安装：默认只安装 CLI
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | sh

# 国内镜像：从 Gitee 拉安装脚本和 release
curl -fsSL https://gitee.com/wattx/popiartcli/raw/main/install.sh | sh -s -- --source gitee

# 安装后的自更新：从 GitHub Releases 下载最新版本，不修改本地配置
popiart update

# 国内镜像自更新：从 Gitee Releases 下载最新版本
popiart update --source gitee

# 更新到指定版本
popiart update --version v0.3.4

# 或直接给 Gitee 仓库主页 / tag 页
popiart update --repo https://gitee.com/wattx/popiartcli
popiart update --repo https://gitee.com/wattx/popiartcli/releases/tag/v0.3.4

# 安装指定版本
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | env VERSION=v0.3.4 sh

# 国内镜像安装指定版本
curl -fsSL https://gitee.com/wattx/popiartcli/raw/main/install.sh | \
  env VERSION=v0.3.2 sh -s -- --source gitee

# 显式写法：仅安装 CLI
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | sh -s -- --cli-only

# 安装 CLI，并继续执行生态引导
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | \
  sh -s -- --bootstrap --agent codex --completion zsh --with-default-skills
```

```powershell
# Windows PowerShell：安装 CLI 后继续执行生态引导
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.ps1))) `
  -Bootstrap `
  -Agent codex `
  -Completion powershell `
  -WithDefaultSkills
```

```sh
# 直接从 GitHub Releases 下载对应平台压缩包后解压安装
# 例如 macOS Apple Silicon
curl -fsSL https://github.com/wtgoku-create/popiartcli/releases/download/v0.3.2/popiart_0.3.2_darwin_arm64.tar.gz -o popiart.tar.gz
tar -xzf popiart.tar.gz
install -m 0755 popiart /usr/local/bin/popiart

# 国内镜像：从 Gitee Releases 下载
curl -fsSL https://gitee.com/wattx/popiartcli/releases/download/v0.3.2/popiart_0.3.2_darwin_arm64.tar.gz -o popiart.tar.gz
```

```sh
# 本地开发运行
go run ./cmd/popiart --help

# 构建本地二进制
go build -o ./dist/popiart ./cmd/popiart

# 安装到 GOPATH/bin
go install ./cmd/popiart
```

`popiart` 的正式 CLI 只保留 Go 版本。
仓库中的 `src/` 和 `bin/` 仅作为旧 Node.js 原型迁移参考，不再作为正式发布渠道。

`curl | sh` 这条安装链路现在默认只负责安装 Go CLI 二进制。
如需继续执行生态引导，可显式追加 `--bootstrap`。

`popiart update` 只会从 GitHub 或 Gitee Releases 下载并替换 CLI 本体，不会改写 `~/.popiart/config.json`，也不会自动重新执行 `bootstrap`。
如果当前安装由 Homebrew 管理，请使用 `brew upgrade wtgoku-create/popi/popiart`。

补充两个边界：

- `popiart update` 可以解析默认仓库，也可以解析 GitHub / Gitee 仓库主页、`releases` 页和 `releases/tag/vX.Y.Z` URL
- 国内镜像默认约定为 `https://gitee.com/wattx/popiartcli`
- 但它最终仍然依赖对应 release 中的目标平台二进制；如果某个 tag 只有源码归档、没有 release 二进制，`popiart update` 不能直接完成升级
- 如果你当前是 `git clone` / `go install` 的源码安装，应该使用 `git pull --tags` 后重新构建

`popiart bootstrap` 负责第二阶段的生态引导：

- 生成 shell completion
- 可选生成 agent 引导文件，并同时写出适用于 shell 的 `env.sh` 与适用于 PowerShell 的 `env.ps1`
- 可选生成默认的远程 skill discovery profile
- 在默认 profile 中写入 CLI 自带的 seed skill 元数据，例如 `popiskill-creator`
- `popiart skills list/get/schema` 会同时显示这些本地 bundled seed skills 和远程注册表技能
- 这些 bundled seed skills 是本地 authoring/helper 入口，不是远端 runtime skill；`popiart run` 只能执行服务端已注册的 runtime skill

这里的 skill 发现仍以远程注册表为主；CLI 仓库同时维护一小组内置 seed skills，作为 bootstrap 和作者引导入口，并在本地查询时一并暴露。

发布维护说明见 [docs/releasing.md](./docs/releasing.md)。

---

## 开发

```sh
# 拉取依赖
make tidy

# 格式化代码
make fmt

# 本地构建
make build

# 查看帮助
make help
```

---

## 当前项目结构

```text
cmd/popiart/main.go       CLI 入口
internal/cmd/             Cobra 命令树
internal/api/             HTTP client 与 response envelope 解包
internal/config/          ~/.popiart/config.json 配置读写
internal/input/           --input JSON / @file / stdin 解析
internal/output/          JSON/plain 输出与错误封装
internal/poll/            job 轮询
internal/seed/            bootstrap 默认 skill profile 与 seed 数据
skills/                   CLI 内置 seed skills
src/                      旧的 Node.js 原型实现（仅迁移参考，不对外发布）
bin/                      旧的 Node.js 启动入口（仅迁移参考，不对外发布）
```

---

## 设计原则

| 原则 | 含义 |
|---|---|
| **默认 JSON 输出** | 每个响应都是 `{ ok: true, data: ... }` 或 `{ ok: false, error: { code, message } }` |
| **所有错误都有代码** | 每次失败都会带有机器可读的 `error.code` (参见 [错误代码](#错误代码)) |
| **长任务返回 job_id** | `run` 立即返回 `{ job_id }`；使用 `jobs wait` 或 `--wait` 来阻塞等待 |
| **所有输入都支持 JSON 文件** | 在接受 `--input` 的任何地方，都可以传递 `@file.json` 或 `-` (标准输入) |
| **所有状态可轮询** | 每个作业都可以使用 `jobs get` 或 `jobs wait` 进行轮询 |
| **所有工件可恢复** | 可以在作业完成后的任何时间执行 `artifacts pull` 或 `artifacts pull-all` |

---

## 身份验证

```sh
# 交互式输入一个 API key
popiart auth login

# 直接传入 API key
popiart auth login --key <product-key>

# 查看当前登录用户
popiart auth whoami

# 登出 (服务端撤销当前 key)
popiart auth logout

# 检查已存储的 key
popiart auth key show

# 轮换 key
popiart auth key rotate
```

已保存的 key 存储在 `~/.popiart/config.json` 中 (权限 0600)。
可以使用 `POPIART_KEY` 或 `POPIART_TOKEN` 环境变量进行覆盖。
如果服务端在登录后下发的是 `sess_...` 这类会话令牌，本地配置里看到它是正常的；CLI 不要求产品层 key 固定为某一个前缀。

---

## 发现技能

```sh
# 列出所有可用技能
popiart skills list

# 根据标签过滤
popiart skills list --tag video

# 全文搜索
popiart skills list --search "image upscale"

# 获取技能的完整模式
popiart skills get skill_abc123

# 打印输入/输出 JSON 模式
popiart skills schema skill_abc123
```

本地安装的 skill 现在也会进入这套查询链路。
`skills list/get/schema` 会按以下优先级合并显示：

- 远端 runtime skills
- 本地 installed skills
- CLI 内置 bundled seed skills

本地 skill 安装链路：

```sh
# 从显式 URL 下载 zip 到 ~/.popiart/skills/downloads/<slug>/
popiart skills pull popiskill-audio-avatar-omnishuman-v1 --url https://example.com/popi.zip

# 从本地 zip 安装到 ~/.popiart/skills/installed/<slug>
popiart skills install ./popiskill-audio-avatar-omnishuman-v1.zip

# 如果已 pull 过，也可以直接按 slug 安装最新下载包
popiart skills install popiskill-audio-avatar-omnishuman-v1

# 将该本地 skill 标记为 run 时优先使用
popiart skills use-local popiskill-audio-avatar-omnishuman-v1

# 安装时同步到 agent 原生 skills 目录
popiart skills install ./popiskill-audio-avatar-omnishuman-v1.zip --agent codex
popiart skills install ./popiskill-audio-avatar-omnishuman-v1.zip --agent claude-code
popiart skills install ./popiskill-audio-avatar-omnishuman-v1.zip --agent openclaw
popiart skills install ./popiskill-audio-avatar-omnishuman-v1.zip --agent opencode
```

当前边界：

- `skills pull/install` 只支持 zip 包
- 暂不支持 `.tar.gz`、本地目录直接安装、GitHub release 页面 URL 或仓库页面 URL
- 若下载链接不是 zip 直链，需要先转换成 zip 包再安装

最小包格式要求：

- zip 包
- 包内存在 `SKILL.md`
- 同时提供 `popiart-skill.yaml` / `popiart-skill.json`，或在 `SKILL.md` 顶部使用 YAML frontmatter
- 若要被 `popiart run` 使用，当前只支持 `execution.mode=remote-runtime`

---

## 运行技能

```sh
# 内联 JSON 输入 — 立即返回 job_id
popiart run skill_abc123 --input '{"prompt":"a sunset over Tokyo"}'

# 从文件输入
popiart run skill_abc123 --input @params.json

# 从标准输入输入
cat params.json | popiart run skill_abc123 --input -

# 阻塞直到完成
popiart run skill_abc123 --input @params.json --wait

# 幂等重试 (多次运行也是安全的)
popiart run skill_abc123 --input @params.json --idempotency-key req-20240318-001
```

成功时的输出：
```json
{ "ok": true, "data": { "job_id": "job_xyz789", "status": "pending" } }
```

使用 `--wait` 时：
```json
{ "ok": true, "data": { "job_id": "job_xyz789", "status": "done", "artifact_ids": ["art_..."] } }
```

---

## 查询作业状态

```sh
# 单个作业状态
popiart jobs get job_xyz789

# 阻塞直到作业终止
popiart jobs wait job_xyz789

# 列出近期作业
popiart jobs list
popiart jobs list --status failed
popiart jobs list --skill skill_abc123 --limit 10

# 流式获取日志
popiart jobs logs job_xyz789
popiart jobs logs job_xyz789 --follow

# 取消正在运行的作业
popiart jobs cancel job_xyz789
```

作业 `status` 的可能值： `pending` → `running` → `done` | `failed` | `cancelled`

---

## 拉取工件

```sh
# 仅上传本地文件，拿到一个稳定媒体 URL
popiart media upload ./source.png

# 读取一个稳定媒体 URL 对应的元数据
popiart media get med_abc

# 上传本地文件，生成可复用 artifact
popiart artifacts upload ./source.png --role source

# 列出作业的工件
popiart artifacts list job_xyz789

# 下载单个工件
popiart artifacts pull art_abc --out ./output.png

# 将工件输出到标准输出 (易于管道操作)
popiart artifacts pull art_abc --stdout > output.png

# 下载作业的所有工件
popiart artifacts pull-all job_xyz789 --dir ./results/
```

当服务端支持稳定媒体 URL 时，`popiart media upload` 和 `popiart artifacts upload` 都会返回 `url` 字段，供后续多模态 skill 直接复用。

当前这条链路已经在测试环境完成过一轮真实 smoke：

- `media upload`
- `media get`
- `artifacts upload`
- `artifacts get`
- `artifacts list`
- `artifacts pull`

如果要做 `img2img`，推荐先把本地图片上传成 artifact，再把返回的 `artifact_id` 填进 `source_artifact_id`：

```sh
ART=$(popiart artifacts upload ./source.png --role source | jq -r '.data.artifact_id')

popiart run popiskill-image-img2img-basic-v1 --input "{
  \"source_artifact_id\":\"$ART\",
  \"prompt\":\"保留主体，改成黄昏电影感\"
}" --wait
```

如果 agent 聊天附件已经带有可直接访问的图片 URL，也可以直接使用 `reference_image_url` / `image_url`，无需先上传。

当前在测试环境里已验证的两条 `img2img` 服务端适配是：

- `gemini-3-pro-image-preview`：通过 Gemini `generateContent` 路由执行
- `seedream-4-5-251128`：通过 `/v1/images/generations` + 参考图执行

补充一点：`seedream-4-5-251128` 对输出尺寸有最小像素要求。CLI 仍然可以提交类似 `1024x1536` 这样的安全预设，但最终是否需要上调尺寸由服务端路由适配决定。

`popiskill-video-image2video-basic-v1` 现在按安装后自带的官方 skill 处理。它应该能在 `skills list/get/schema` 里直接看到；当远端目录里的同名条目仍是占位符或尚未注册时，CLI 会自动把这条 skill 桥接到底层 `models infer`，先试 `viduq3-turbo`，失败再回落到 `viduq2-pro-fast`。

如果要做 `image2video`，推荐仍然先把本地图片上传成 artifact，再把返回的 `artifact_id` 填进 `source_artifact_id`：

```sh
ART=$(popiart artifacts upload ./source.png --role source | jq -r '.data.artifact_id')

popiart run popiskill-video-image2video-basic-v1 --project proj_local_dev --input "{
  \"source_artifact_id\":\"$ART\",
  \"prompt\":\"让人物衣摆和发丝在微风中轻轻摆动，镜头缓慢推进，整体保持真实电影感。\",
  \"aspect_ratio\":\"16:9\",
  \"seconds\":4
}" --wait
```

截至 `2026-04-08`，CLI 内置 `image2video` fallback 的模型顺序是：

- `viduq3-turbo`
- `viduq2-pro-fast`

截至 `2026-03-28`，测试环境里已验证通过的服务端 `image2video` 路由是：

- `video.image2video -> viduq2-pro-fast`

如果服务端将来补齐真正的 runtime skill 注册和路由表，`popiart run popiskill-video-image2video-basic-v1` 会优先走服务端 skill；只有在服务端条目缺失或仍是占位符时，CLI 才会自动改走底层 `models infer`。

---

## 预算与使用情况

```sh
# 当前周期摘要
popiart budget status

# 详细使用情况细分
popiart budget usage --since 2024-03-01 --group-by skill

# 速率限制与配额配置
popiart budget limits
```

---

## 项目上下文

```sh
# 显示活动项目
popiart project current

# 切换项目
popiart project use proj_abc123

# 列出所有可访问的项目
popiart project list

# 获取完整运行时上下文 (技能、预算、元数据)
popiart project context
```

---

## 模型路由

`popiart models list` 显示的是后端已注册的模型库存。
`popiart models routes` 显示的是当前项目真正生效的 `route_key -> model_id` 路由结果。

```sh
# 列出已注册的可用模型库存
popiart models list
popiart models list --type image
popiart models list --capability text2image
popiart models list --provider runway

# 查看当前生效的 route key 路由表
popiart models routes
popiart models routes --project proj_abc123
popiart models routes --project proj_abc123 --route image.text2image

# 直接提交模型推理任务
popiart models infer img-gen-xl --input @params.json
popiart models infer video-gen-v2 --input @params.json --wait

# 设置项目级 route key 覆盖
popiart models route-override set --project proj_abc123 --route image.img2img --model seedream-4-5-251128
popiart models route-override set --project proj_abc123 --route video.image2video --model viduq2-pro-fast

# 删除项目级 route key 覆盖
popiart models route-override unset --project proj_abc123 --route image.img2img

# 列出项目级 route key 覆盖
popiart models route-override list --project proj_abc123
popiart models route-override list --project proj_abc123 --route image.img2img
```

---

## 全局标志

```
--endpoint <url>    覆盖 API 端点 (将持久化到配置中)
--project <id>      覆盖活动项目 (将持久化到配置中)
--plain             人类可读的输出 (而不是 JSON)
--no-color          在纯文本输出中禁用 ANSI 颜色
-v, --version       打印版本
--help              打印帮助
```

---

## 生态引导

```sh
# 生成 Codex / OpenCode 的引导文件
popiart bootstrap --agent codex --agent opencode

# 生成 zsh completion
popiart bootstrap --completion zsh

# 保存 key，并生成默认的远程 skill discovery profile
popiart bootstrap --key pk-... --with-default-skills

# 直接输出 shell completion 到标准输出
popiart completion zsh > ~/.zsh/completions/_popiart
```

环境变量:
```
POPIART_KEY        API key (优先覆盖已存储的配置)
POPIART_TOKEN      兼容旧用法：等同于 POPIART_KEY
POPIART_ENDPOINT   API 端点 (覆盖已存储的配置)
POPIART_PROJECT    活动的项目 ID
POPIART_CONFIG_DIR 配置目录路径 (默认: ~/.popiart)
```

---

## 错误代码

| 代码 | 含义 |
|---|---|
| `UNAUTHENTICATED` | 无令牌或令牌已过期 — 请运行 `auth login` |
| `FORBIDDEN` | 令牌没有访问此资源的权限 |
| `NOT_FOUND` | 资源不存在 |
| `VALIDATION_ERROR` | 输入未通过模式验证 |
| `RATE_LIMITED` | 超出速率限制 — 请稍后重试 |
| `JOB_FAILED` | 作业达到了 `failed` 状态 (查看 `error.details`) |
| `POLL_TIMEOUT` | 使用了 `--wait` 参数，但等待超时作业仍未完成 |
| `NETWORK_ERROR` | 无法连接到 API 端点 |
| `INPUT_PARSE_ERROR` | `--input` 值不是有效的 JSON |
| `INPUT_NOT_FOUND` | `@file` 路径不存在 |
| `CLI_ERROR` | CLI 内部错误 (参数或本地处理失败等) |
| `FATAL` | 未处理的异常 |
| `BAD_REQUEST` | 服务器返回 HTTP 400 |
| `CONFLICT` | HTTP 409 (例如幂等键重复) |
| `SERVER_ERROR` | HTTP 500 |

所有错误都以代码 `1` 退出。成功执行的命令以代码 `0` 退出。

---

## 支持管道流的操作

因为输出是按换行符分隔的 JSON，所以 `popiart` 可以很自然地与 `jq` 结合使用：

```sh
# 从运行输出中提取 job_id
JOB=$(popiart run skill_abc --input @p.json | jq -r '.data.job_id')

# 等待并拉取工件
popiart jobs wait "$JOB" | jq '.data.artifact_ids[]' | \
  xargs -I {} popiart artifacts pull {}

# 通过 jq 导出 CSV 格式的使用情况报告
popiart budget usage --group-by skill | \
  jq -r '.data.rows[] | [.skill_id, .tokens_used] | @csv'
```

---

## 配置文件

位置在 `~/.popiart/config.json` (权限 0600):

```json
{
  "endpoint": "https://api.creatoragentos.io/v1",
  "token": "sess_...",
  "project": "proj_abc123"
}
```
