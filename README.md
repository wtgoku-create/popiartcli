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

如果你是从 GitHub Release 手动下载压缩包安装，解压后把里面的可执行文件 `popiart` 放到 `PATH` 里的目录即可。macOS 从浏览器下载后如果被系统拦截，可先执行：

```sh
xattr -d com.apple.quarantine popiart 2>/dev/null || true
```

然后跑最短路径：

```sh
popiart setup --agent codex --completion zsh
popiart --endpoint https://www.popi.art auth login --key <token>
```

完整命令说明、参数和示例统一见 [docs/cli-command-reference.md](./docs/cli-command-reference.md)。

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

如果要提交首尾帧视频，首帧仍用 `--image` / `--from`，尾帧用 `--last-frame`：

```sh
popiart video generate \
  --image ./first-frame.png \
  --last-frame ./last-frame.png \
  --prompt "从第一帧自然过渡到最后一帧，镜头平稳推进" \
  --model MiniMax-Hailuo-02 \
  --size 768P \
  --duration 6 \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

如果你想直接识别一张图并返回可复用的描述性 prompt，可以执行：

```sh
popiart image describe \
  --image ./source.png \
  --model gemini-2.5-flash \
  --prompt "请写成适合文生图复用的 prompt" \
  --output json \
  --quiet \
  --non-interactive
```

如果想先让带图像理解的模型把“一张图 + 一句简单描述”扩写成更完整的图生视频提示词，再提交视频模型，可以加：

```sh
popiart video generate \
  --image ./source.png \
  --prompt "让人物自然转头，镜头慢慢推进" \
  --prompt-enhancer-model gemini-2.5-flash \
  --model viduq2-pro-fast \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

如果要做即梦动作迁移，传一张身份图和一个动作参考视频：

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

行为说明：

- 默认模型是 `jimeng_dreamactor_m20_gen_video`。
- `--image` 是身份图，会提交为统一网关 `images[0]`。
- `--video` 是动作参考视频，会提交为统一网关 `videos[0]`。
- `--cut-result-first-second-switch` 会提交为 `metadata.cut_result_first_second_switch=true`。
- 本地图片 / 视频会先上传为 stable media URL，再提交给服务端。
- 如果 `--image` 是 `data:image/*;base64,...`，CLI 会自动剥离前缀，只提交即梦要求的纯 base64。

如果要走 Seedance / 豆包视频模型，可以直接用专门入口：

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

Seedance 首尾帧也可以用便捷参数：

```sh
popiart video seedance \
  --image ./first-frame.png \
  --last-frame ./last-frame.png \
  --prompt "从第一帧自然过渡到最后一帧" \
  --ratio 16:9 \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

## 默认入口

推荐优先记住这几个入口：

- `popiart setup --agent codex`
- `popiart image generate`
- `popiart image describe`
- `popiart image img2img`
- `popiart video generate`
- `popiart video img2video`
- `popiart video action-transfer`
- `popiart video seedance`
- `popiart speech synthesize`
- `popiart music generate`

它们是面向新用户和 agent 的 opinionated façade，内部仍然映射到官方 runtime skill，不改变底层架构。

## 当前保证范围

- 仓库中的权威实现是 Go CLI：`cmd/popiart`。根目录 `package.json` 只保留仓库任务入口，不再代表一个正式发布的 Node CLI。
- `popiart setup --agent ...` 会优先把 PopiArt 做到 agent 可发现，但“可发现”不等于“远端 runtime 已就绪”。
- `popiart mcp doctor` 现在会分别返回 `discoverability_status` 和 `runtime_status`。
- 当前 MCP server 重点实现的是 `tools/list` / `tools/call` 工具面；`resources`、`prompts`、`sampling` 仍未完成。
- 七个 official runtime baseline skill 已有 discoverability 契约，但 CLI 目前仍不能单独保证七个 skill 都能端到端执行成功。
- `project`、`budget`、`jobs logs/cancel`、`artifacts pull <artifact-id>` 这类旧语义命令当前仍属于兼容保留命令面，在主站迁移模式下会明确返回 `UNSUPPORTED_IN_POPI_ART_MODE`。

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

## Agent Skill 安装

最优先推荐用 CLI 自己安装 native agent 入口：

```sh
popiart setup --agent codex --completion zsh
```

这会同时生成本地 MCP 配置、agent skill wrapper、runtime baseline 与诊断清单。随后运行：

```sh
popiart mcp doctor --agent codex
```

如果你正在让 Codex、Claude Code、OpenCode、OpenClaw 等 agent 帮你接入 PopiArt，可以先直接把这句话发给 agent：

> 从 `https://github.com/wtgoku-create/popiartcli/tree/main/skill` 安装 PopiArt agent skill，安装名使用 `popiart-cli`。只安装这个 `skill/` 目录，不要安装仓库根目录、`blob/main/skill/SKILL.md` 或 `popiskills/`。如果本机还没有 `popiart` CLI，先按 README 的 Homebrew 或 `install.sh` 方式安装 CLI，再执行 `popiart setup --agent codex --completion zsh` 和 `popiart mcp doctor --agent codex`。

如果用户的 agent 支持直接从 GitHub 安装 skill，可以把本仓库的 [`skill/`](./skill/) 目录作为安装目标；这个目录只有一个权威入口 [`skill/SKILL.md`](./skill/SKILL.md)，适合被 Codex、Claude Code、OpenCode、OpenClaw 等 agent 复制到自己的 skills 目录。仓库里的 [`popiskills/`](./popiskills/) 是 PopiArt runtime skill 的 bundled seed，不是 agent skill 安装入口。

OpenClaw 用户不要发仓库根目录或 `blob/main/skill/SKILL.md` 给安装器；很多安装器会优先扫描仓库内的 `skills/` 约定目录或递归安装所有 `SKILL.md`。推荐使用目录链接：

```text
https://github.com/wtgoku-create/popiartcli/tree/main/skill
```

或手动安装 raw 文件：

```sh
mkdir -p ~/.openclaw/skills/popiart-cli
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/skill/SKILL.md \
  -o ~/.openclaw/skills/popiart-cli/SKILL.md
```

模型切换规则也写在这个 agent skill 里：单次请求优先用 intent 命令的 `--model <model-id>`，项目级长期切换才用 `popiart models route-override set`，只有用户明确要直连某个底层模型时才用 `popiart models infer`。

如果你刚完成初始化，推荐先运行：

```sh
popiart mcp doctor --agent codex
```

判读方式：

- `discoverability_status=pass`：本地 agent 原生 MCP / skill 入口大致已就位
- `runtime_status=pass`：远端登录态、baseline skill 注册与默认路由更接近可执行
- 两者都通过之前，不要把 `setup` 视为“已经可端到端跑通”

## 命令文档

README 不再内嵌命令解释。请直接查看：

- 命令参考：[docs/cli-command-reference.md](./docs/cli-command-reference.md)
- Recipes：[docs/recipes.md](./docs/recipes.md)
- 安装与使用：[docs/install-and-usage.md](./docs/install-and-usage.md)

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

如果是手动下载 Windows release zip，解压后把其中的 `popiart.exe` 放到一个已加入 `PATH` 的目录，再重新打开 PowerShell 执行 `popiart --version`。

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

- 命令参考：[docs/cli-command-reference.md](./docs/cli-command-reference.md)
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
