# popiart

**面向 Coding Agent 的创作者技能 CLI。**

`popiart` 把创作者 skill、作业、工件、稳定媒体 URL、MCP discoverability 和统一鉴权/计费收敛成一个 agent 友好的本地入口。

## 3 分钟上手

先安装 CLI：

```sh
# macOS / Linux
brew tap wtgoku-create/popi
brew install wtgoku-create/popi/popiart

# 或者
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | sh
```

然后跑最短路径：

```sh
# 1. 为 agent 做一键初始化
popiart setup --agent codex --completion zsh

# 2. 登录
popiart auth login --key <product-key>

# 3. 看看有哪些官方能力
popiart skills list --search popiskill --output json --quiet --non-interactive

# 4. 直接用用户意图命令生成图片
popiart image generate \
  --prompt "A cinematic portrait of a creator at sunset" \
  --aspect-ratio 9:16 \
  --output json \
  --quiet \
  --non-interactive
```

如果你已经有一张本地图，最短的视频路径是：

```sh
popiart video generate \
  --image ./source.png \
  --prompt "Slow push-in and soft wind movement" \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

## 默认入口

推荐优先记住这 4 个入口：

- `popiart setup --agent codex`
- `popiart image generate`
- `popiart image img2img`
- `popiart video generate`
- `popiart video img2video`
- `popiart audio tts`

它们是面向新用户和 agent 的 opinionated façade，内部仍然映射到官方 runtime skill，不改变底层架构。

## Agent / CI 契约

在 agent 或 CI 环境里，推荐统一使用：

```sh
--output json --quiet --non-interactive
```

补充约定：

- `--dry-run`：预览规范化后的请求，不执行网络写操作
- `--async`：显式要求立即返回 job
- `--wait`：阻塞直到 job 结束
- `--output plain`：人类可读模式；`--plain` 仍保留兼容

完整 agent 契约见 [skill/SKILL.md](./skill/SKILL.md)。

## 用户意图命令面

目前已经提供的 façade：

```sh
popiart image generate --prompt "..."
popiart image img2img --image ./source.png --prompt "..."
popiart video generate --image ./source.png --prompt "..."
popiart video img2video --image ./source.png --prompt "..."
popiart audio tts --text "..."
```

它们当前分别映射到：

- `popiskill-image-text2image-basic-v1`
- `popiskill-image-img2img-basic-v1`
- `popiskill-video-image2video-basic-v1`
- `popiskill-video-image2video-basic-v1`
- `popiskill-audio-tts-multimodel-v1`

底层平台面仍然保留：

```sh
popiart skills ...
popiart run ...
popiart jobs ...
popiart artifacts ...
popiart media ...
popiart mcp ...
popiart bootstrap ...
```

也就是说：

- 新用户和 agent 先用意图命令面
- 平台集成、排障和精细控制再下沉到平台命令面

## 平台命令面

如果你需要更底层、更可组合的控制，下面这些命令面仍然是 `popiart` 的核心平台接口：

- `popiart skills ...`
  用来发现、查看和理解 skill 契约。常用的是 `skills list`、`skills get`、`skills schema`。
- `popiart run ...`
  直接按 `skill_id` 提交 runtime job，适合 agent 先拿 schema 再自行构造 `--input` 的场景。
- `popiart jobs ...`
  查询、等待、取消和跟踪 job。常用的是 `jobs get`、`jobs wait`、`jobs logs`。
- `popiart artifacts ...`
  上传本地文件成为可复用 artifact，或者把 job 产物拉回本地。常用的是 `artifacts upload`、`artifacts pull`、`artifacts pull-all`。
- `popiart media ...`
  把本地文件变成稳定媒体 URL，适合后续 `img2img` / `img2video` 直接消费稳定地址。
- `popiart mcp ...`
  暴露 MCP server、打印 MCP config、做 discoverability / runtime doctor 诊断。
- `popiart bootstrap ...`
  细粒度生成 bootstrap 资产，适合维护者或需要精确控制 agent 引导文件时使用。

一条常见的“平台命令面”链路是：

```sh
popiart skills schema popiskill-image-img2img-basic-v1 \
  --output json \
  --quiet \
  --non-interactive

popiart artifacts upload ./source.png \
  --role source \
  --output json \
  --quiet \
  --non-interactive

popiart run popiskill-image-img2img-basic-v1 \
  --input @params.json \
  --output json \
  --quiet \
  --non-interactive

popiart jobs wait <job-id> \
  --output json \
  --quiet \
  --non-interactive

popiart artifacts pull-all <job-id> \
  --output json \
  --quiet \
  --non-interactive
```

## 可组合 Recipes

标准 recipes 在 [docs/recipes.md](./docs/recipes.md)，其中包括：

- `artifact -> run -> wait -> pull`
- `media upload -> img2img`
- `video generate --image <local-file>`
- `skills schema/get -> run`
- stdout / stderr 约定
- 配置优先级

## 按平台安装

完整安装与平台说明见 [docs/install-and-usage.md](./docs/install-and-usage.md)。如果你只想快速开始，可以直接按下面的平台片段执行。

### macOS

推荐 Homebrew：

```sh
brew tap wtgoku-create/popi
brew install wtgoku-create/popi/popiart

# 给 Codex 做默认初始化
popiart setup --agent codex --completion zsh
```

如果你更喜欢脚本安装：

```sh
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | sh
popiart setup --agent codex --completion zsh
```

### Linux

推荐脚本安装：

```sh
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | sh

# 例如给 Claude Code 做默认初始化
popiart setup --agent claude-code --completion bash
```

如果你的 Linux 环境已经装了 Homebrew，也可以：

```sh
brew tap wtgoku-create/popi
brew install wtgoku-create/popi/popiart
```

### Windows

推荐 PowerShell 安装脚本：

```powershell
irm https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.ps1 | iex

# 给 Codex 做默认初始化
popiart setup --agent codex --completion powershell
```

### 安装后建议做什么

无论在哪个平台，安装完成后建议按这个顺序做：

```sh
popiart setup --agent codex
popiart auth login --key <product-key>
popiart image generate --prompt "hello" --output json --quiet --non-interactive
```

如果你需要更细的安装方式，比如 release 压缩包、国内镜像、源码安装、Windows 参数化安装，直接看 [docs/install-and-usage.md](./docs/install-and-usage.md)。

## 错误与退出码

公开错误参考见 [ERRORS.md](./ERRORS.md)。

你可以依赖：

- 稳定的 JSON error envelope
- 公开的 `error.code`
- 明确的 exit code 语义
- 每类错误的重试建议

## 相关文档

- 安装与使用：[docs/install-and-usage.md](./docs/install-and-usage.md)
- Recipes：[docs/recipes.md](./docs/recipes.md)
- 错误参考：[ERRORS.md](./ERRORS.md)
- 开发者总览：[docs/developer-docs.md](./docs/developer-docs.md)
- 当前仓库实际状态：[docs/current-status.md](./docs/current-status.md)
- MCP discoverability 设计：[docs/mcp-discoverability-v1.md](./docs/mcp-discoverability-v1.md)
- 稳定媒体 URL 设计：[docs/stable-media-url-v1.md](./docs/stable-media-url-v1.md)
- 项目边界：[docs/project-relationship.md](./docs/project-relationship.md)

## 项目边界

如果你是首次接触 `popiart`，这一节可以后读。

`popiartcli` 的职责是：

- 给 Coding Agent 提供统一本地入口
- 暴露 discoverability、MCP、jobs、artifacts、media、runtime baseline
- 统一处理本地配置、项目上下文和 agent 接入资产

它不负责：

- 替代每个创作者 skill 的业务逻辑
- 在 CLI 内部复制所有服务端 runtime
- 直接持有所有上游模型 key

三层关系见 [docs/project-relationship.md](./docs/project-relationship.md)。

## 开发

```sh
make tidy
make fmt
make build
make help
```

正式发布渠道只保留 Go CLI。仓库里的 `src/` 和 `bin/` 仅作为旧 Node.js 原型迁移参考，不再作为正式发布入口。
