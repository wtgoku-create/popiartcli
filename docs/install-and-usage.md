# PopiArt CLI 安装与使用说明书

这份文档面向两类读者：

- 人类用户：想在本机安装 `popiart`，登录、发现 skill、运行任务、拉取 artifacts
- Coding Agent 使用者：想让 Codex、Claude Code、OpenClaw、OpenCode 等 agent 在本机舒适地使用 `popiart`

如果你需要先理解系统边界，而不是先安装，请先看 [docs/project-relationship.md](./project-relationship.md)。

## 1. 安装方式总览

| 平台 | 推荐安装方式 | 适合场景 |
|---|---|---|
| macOS | Homebrew 或 `install.sh` | 日常本机使用、快速升级 |
| Linux | `install.sh` 或 release 压缩包 | 服务器、开发机、CI |
| Windows | `install.ps1` | PowerShell 用户、桌面环境 |
| 任意平台 | `go install ./cmd/popiart` | 本地开发、源码调试 |

安装完成后，先执行：

```sh
popiart --help
```

如果能看到命令帮助，说明 CLI 已经在 `PATH` 中。

升级说明：

- 如果是通过 `install.sh`、`install.ps1` 或手动 release 二进制安装，可以直接运行 `popiart update`
- 如果是通过 Homebrew 安装，请运行 `brew upgrade wtgoku-create/popi/popiart`
- `popiart update` 只更新 CLI 本体，不会改动现有配置，也不会自动重新执行 bootstrap

## 2. macOS 安装

### 2.1 Homebrew

```sh
brew tap wtgoku-create/popi
brew install wtgoku-create/popi/popiart

# 升级
brew upgrade wtgoku-create/popi/popiart
```

如果你想马上生成 agent 引导文件和 shell completion：

```sh
popiart bootstrap --agent codex --completion zsh --with-default-skills
```

### 2.2 官方安装脚本

只安装 CLI：

```sh
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | sh

# 后续升级到最新 release
popiart update
```

安装 CLI，并继续做 bootstrap：

```sh
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | \
  sh -s -- --bootstrap --agent codex --completion zsh --with-default-skills
```

安装指定版本：

```sh
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | \
  env VERSION=v0.3.0 sh

# 或者在已安装后更新到指定版本
popiart update --version v0.3.0
```

脚本会优先尝试：

- Homebrew 的 `bin` 目录
- `/opt/homebrew/bin`
- `/usr/local/bin`
- `~/.local/bin`

如果安装目录不在 `PATH` 中，脚本会打印对应 shell 的追加方法。

### 2.3 GitHub Releases 手动安装

```sh
curl -fsSL https://github.com/wtgoku-create/popiartcli/releases/download/v0.3.0/popiart_0.3.0_darwin_arm64.tar.gz -o popiart.tar.gz
tar -xzf popiart.tar.gz
install -m 0755 popiart /usr/local/bin/popiart
```

Intel Mac 请改成 `darwin_amd64` 对应的压缩包。

### 2.4 从源码安装

```sh
git clone https://github.com/wtgoku-create/popiartcli.git
cd popiartcli
go install ./cmd/popiart
popiart --help
```

## 3. Linux 安装

### 3.1 官方安装脚本

只安装 CLI：

```sh
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | sh

# 后续升级到最新 release
popiart update
```

安装 CLI 并 bootstrap：

```sh
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | \
  sh -s -- --bootstrap --agent claude-code --completion bash --with-default-skills
```

你也可以显式指定安装目录：

```sh
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | \
  env BINDIR="$HOME/.local/bin" sh
```

### 3.2 Homebrew

如果你的 Linux 环境已经安装 Homebrew，也可以直接使用：

```sh
brew tap wtgoku-create/popi
brew install wtgoku-create/popi/popiart
```

### 3.3 GitHub Releases 手动安装

amd64 示例：

```sh
curl -fsSL https://github.com/wtgoku-create/popiartcli/releases/download/v0.3.0/popiart_0.3.0_linux_amd64.tar.gz -o popiart.tar.gz
tar -xzf popiart.tar.gz
install -m 0755 popiart "$HOME/.local/bin/popiart"
```

arm64 机器请改用 `linux_arm64` 对应压缩包。

### 3.4 从源码安装

```sh
git clone https://github.com/wtgoku-create/popiartcli.git
cd popiartcli
go install ./cmd/popiart
popiart --help
```

## 4. Windows 安装

### 4.1 PowerShell 安装脚本

只安装 CLI：

```powershell
irm https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.ps1 | iex

# 后续升级到最新 release
popiart update
```

安装指定版本：

```powershell
$env:VERSION="v0.3.0"
irm https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.ps1 | iex
```

安装 CLI，并继续做 bootstrap：

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.ps1))) `
  -Bootstrap `
  -Agent codex `
  -Completion powershell `
  -WithDefaultSkills
```

默认安装目录是：

```text
%LOCALAPPDATA%\Programs\popiart\bin
```

脚本会自动尝试把这个目录加入用户级 `PATH`。

### 4.2 GitHub Releases 手动安装

```powershell
$version = "0.3.0"
$zip = "popiart_${version}_windows_amd64.zip"
Invoke-WebRequest "https://github.com/wtgoku-create/popiartcli/releases/download/v$version/$zip" -OutFile $zip
Expand-Archive $zip -DestinationPath .
New-Item -ItemType Directory -Force "$env:LOCALAPPDATA\Programs\popiart\bin" | Out-Null
Copy-Item .\popiart.exe "$env:LOCALAPPDATA\Programs\popiart\bin\popiart.exe" -Force
```

arm64 Windows 请改用 `windows_arm64` 的压缩包。

### 4.3 从源码安装

```powershell
git clone https://github.com/wtgoku-create/popiartcli.git
cd popiartcli
go install ./cmd/popiart
popiart --help
```

## 5. 安装后的检查

无论你在哪个平台安装，建议先做这几步：

```sh
popiart --help
popiart auth --help
popiart skills --help
popiart run --help
```

如果你准备给人看结果，可以加 `--plain`：

```sh
popiart --plain auth whoami
```

如果你准备让 agent 或脚本解析输出，建议保留默认 JSON 输出，不要加 `--plain`。

## 6. 人类如何使用

### 6.1 登录

交互式输入 key：

```sh
popiart auth login
```

直接传 key：

```sh
popiart auth login --key pk-...
```

验证当前身份：

```sh
popiart auth whoami
popiart auth key show
```

说明：

- `popiart` 里保存的是 PopiArt 产品层 key
- 不要把 OpenAI、Gemini、Kling、Runway 等 provider key 直接塞进 CLI
- 如果你需要切到测试环境，可以用 `--endpoint` 或 `POPIART_ENDPOINT`

### 6.2 选择项目

如果你的账号下有多个项目，先确认当前项目：

```sh
popiart project current
popiart project list
popiart project use <project-id>
```

### 6.3 安装和使用本地 skill 包

如果某个 skill 以 zip 包分发，而不是直接由远端注册表托管，可以走本地安装链路。

下载到本地缓存：

```sh
popiart skills pull popiskill-audio-avatar-omnishuman-v1 --url https://example.com/popi.zip
```

直接从 zip 安装：

```sh
popiart skills install ./popiskill-audio-avatar-omnishuman-v1.zip
```

如果已经 pull 过，也可以按 slug 从 `~/.popiart/skills/downloads/` 中安装：

```sh
popiart skills install popiskill-audio-avatar-omnishuman-v1
```

当前边界：

- `skills pull` / `skills install` 目前只支持 zip
- 暂不支持 `.tar.gz`、本地目录、GitHub release 页面 URL、GitHub 仓库页面 URL
- 如果拿到的不是 zip 直链，需要先整理成 zip 包后再安装

安装完成后：

- skill 会进入 `popiart skills list`
- `popiart skills get <slug>` / `schema <slug>` 会优先读取本地安装版本
- 如果该 skill 的 `execution.mode=remote-runtime`，则可以通过 `popiart run <slug>` 触发它映射的远端 runtime skill

当本地安装 skill 与远端同名 skill 冲突时，显式切到本地优先：

```sh
popiart skills use-local popiskill-audio-avatar-omnishuman-v1
```

如果要同时放进 agent 的 skills 目录：

```sh
popiart skills install ./popiskill-audio-avatar-omnishuman-v1.zip \
  --agent codex \
  --agent-skill-dir ~/.codex/skills
```

最小包格式：

- zip 格式
- 包内存在 `SKILL.md`
- 提供 `popiart-skill.yaml` / `popiart-skill.json`，或在 `SKILL.md` 顶部提供 YAML frontmatter
- 若要被 `popiart run` 直接使用，当前只支持：

```yaml
execution:
  mode: remote-runtime
  runtime_skill_id: popiskill-remote-runtime-v1
  runner: popiart
```

也可以临时覆盖：

```sh
popiart --project <project-id> skills list
```

### 6.3 发现技能

```sh
popiart skills list --plain
popiart skills list --search "image"
popiart skills list --tag video
popiart skills get <skill-id> --plain
popiart skills schema <skill-id> --plain
```

建议在第一次运行一个 skill 之前，至少先执行一次：

```sh
popiart skills get <skill-id>
popiart skills schema <skill-id>
```

### 6.4 运行技能

内联 JSON：

```sh
popiart run <skill-id> --input '{"prompt":"a sunset over Tokyo"}'
```

从文件读取输入：

```sh
popiart run <skill-id> --input @params.json
```

从标准输入读取：

```sh
cat params.json | popiart run <skill-id> --input -
```

等待任务完成：

```sh
popiart run <skill-id> --input @params.json --wait
```

安全重试：

```sh
popiart run <skill-id> --input @params.json --idempotency-key req-001
```

### 6.5 查看 jobs 和拉取 artifacts

```sh
popiart jobs get <job-id>
popiart jobs wait <job-id>
popiart jobs logs <job-id>
popiart artifacts list <job-id>
popiart artifacts upload ./source.png --role source
popiart artifacts pull <artifact-id>
popiart artifacts pull-all <job-id>
```

把单个 artifact 直接写到 stdout：

```sh
popiart artifacts pull <artifact-id> --stdout
```

本地图片要进入 `img2img` 时，优先先上传成 artifact：

```sh
ART=$(popiart artifacts upload ./source.png --role source | jq -r '.data.artifact_id')

popiart run popiskill-image-img2img-basic-v1 --input "{
  \"source_artifact_id\":\"$ART\",
  \"prompt\":\"保留主体，改成黄昏电影感\"
}" --wait
```

## 7. Agent 如何使用

### 7.1 先理解 agent 接入原则

- `popiart` 默认输出 JSON，这通常比 `--plain` 更适合 agent
- agent 应该拿 PopiArt 产品层 key，而不是 provider key
- agent 应该先 `skills get` / `skills schema`，再决定是否 `run`
- agent 应优先使用 `@params.json` 或 stdin，而不是把大段 JSON 内联到命令里
- bootstrap 生成的本地 bundled seed skills 只用于 authoring 和引导，不等于远端 runtime skill

最重要的一条：

```text
能在 `skills get/schema` 里看到，不代表一定能直接 `run`
```

例如 `popiskill-creator` 是 CLI 内置 helper skill；如果服务端没有注册对应 runtime skill，`popiart run popiskill-creator` 会返回本地提示，而不是假装执行成功。

### 7.2 聊天附件如何进入 img2img

如果 agent 聊天里收到用户上传的图片，不要直接把图片二进制塞进 `run`。

当前推荐顺序是：

1. 宿主先把聊天附件保存到本地临时文件路径。
2. 调用 `popiart artifacts upload <path> --role source`。
3. 读取返回的 `artifact_id`。
4. 再调用 `popiart run popiskill-image-img2img-basic-v1`，把它放进 `source_artifact_id`。

示例：

```sh
ART=$(popiart artifacts upload /tmp/chat-upload.png --role source | jq -r '.data.artifact_id')

popiart run popiskill-image-img2img-basic-v1 --input "{
  \"source_artifact_id\":\"$ART\",
  \"prompt\":\"保留主体身份与主要视觉特征，改成海边黄昏场景\"
}" --wait
```

如果聊天附件本身已经有可访问 URL，也可以直接走 `reference_image_url` / `image_url`，不一定要先上传。

### 7.4 当前已验证的服务端 `img2img` 路由

截至 `2026-03-28`，测试环境里已经验证过两条服务端图像编辑适配：

- `gemini-3-pro-image-preview`
  通过 Gemini `generateContent` 路由处理参考图编辑
- `seedream-4-5-251128`
  通过 `/v1/images/generations` + 参考图输入处理图生图

注意：

- 这两条能力属于 `popiartServer` / `PopiNewAPI` 的服务端路由适配，不是 CLI 本身直接决定的
- `seedream-4-5-251128` 对输出尺寸有最小像素限制。CLI 可以继续传递像 `1024x1536` 这样的安全预设，但服务端可能会把它抬升到满足模型要求的尺寸后再提交

### 7.3 让 agent 获得稳定环境

`popiart bootstrap` 会做三件有价值的事：

- 生成 shell completion
- 生成 agent 专用环境注入文件
- 可选生成默认 skill discovery profile

支持的 agent 名称：

- `codex`
- `claude-code`
- `openclaw`
- `opencode`

支持的 completion shell：

- `bash`
- `zsh`
- `fish`
- `powershell`

### 7.4 macOS / Linux 上给 agent 注入环境

Codex 示例：

```sh
popiart bootstrap --agent codex --completion zsh --with-default-skills
source ~/.popiart/agents/codex/env.sh
source ~/.popiart/completions/_popiart
```

Claude Code 示例：

```sh
popiart bootstrap --agent claude-code --completion bash --with-default-skills
source ~/.popiart/agents/claude-code/env.sh
source ~/.popiart/completions/popiart.bash
```

OpenClaw 示例：

```sh
popiart bootstrap --agent openclaw --completion fish --with-default-skills
source ~/.popiart/agents/openclaw/env.sh
source ~/.popiart/completions/popiart.fish
```

OpenCode 示例：

```sh
popiart bootstrap --agent opencode --completion zsh --with-default-skills
source ~/.popiart/agents/opencode/env.sh
source ~/.popiart/completions/_popiart
```

推荐做法：

- 先在一个干净 shell 中 `source` 对应的 `env.sh`
- 再从这个 shell 启动 agent，或把其中的环境变量写到 agent 的环境注入配置里
- 让 agent 在同一个 shell 会话里调用 `popiart`

### 7.5 Windows 上给 agent 注入环境

Codex 示例：

```powershell
popiart bootstrap --agent codex --completion powershell --with-default-skills
$agentEnv = Join-Path $HOME ".popiart\agents\codex\env.ps1"
$completion = Join-Path $HOME ".popiart\completions\popiart.ps1"
. $agentEnv
. $completion
```

Claude Code / OpenClaw / OpenCode 也是同样模式，只需要把 agent 名称替换掉：

```powershell
popiart bootstrap --agent claude-code --completion powershell --with-default-skills
$agentEnv = Join-Path $HOME ".popiart\agents\claude-code\env.ps1"
$completion = Join-Path $HOME ".popiart\completions\popiart.ps1"
. $agentEnv
. $completion
```

推荐做法：

- 先在 PowerShell 里执行 `. <env.ps1>`
- 再从这个 PowerShell 会话中启动 agent
- 如果 agent 有自己的环境变量配置面板，也可以直接复制 `env.ps1` 中的变量

### 7.6 agent 的推荐调用模式

发现：

```sh
popiart skills list --search "image"
popiart skills get <skill-id>
popiart skills schema <skill-id>
```

提交任务：

```sh
popiart run <skill-id> --input @params.json
```

等待结果：

```sh
popiart jobs wait <job-id>
```

下载结果：

```sh
popiart artifacts pull-all <job-id>
```

如果 agent 只想拿结构化结果，不想轮询多次，可以直接：

```sh
popiart run <skill-id> --input @params.json --wait
```

### 7.7 agent 使用中的几个约束

- 人读为主的命令可以加 `--plain`，但机器读为主时建议不要加
- `--interval` 必须是一个大于 `0` 的整数毫秒值，例如 `2000`
- 大输入优先走 `@params.json` 或 stdin
- 如果你切换了 endpoint、project 或 key，最好重新生成或重新 `source` 一次 agent env 文件

## 8. 配置路径与环境变量

默认配置目录：

| 平台 | 默认配置目录 |
|---|---|
| macOS / Linux | `~/.popiart` |
| Windows | `%USERPROFILE%\.popiart` |

常见文件：

| 文件 | 作用 |
|---|---|
| `config.json` | endpoint、key、project 等本地配置 |
| `bootstrap.json` | 最近一次 bootstrap 产物清单 |
| `agents/<agent>/env.sh` | shell agent 环境文件 |
| `agents/<agent>/env.ps1` | PowerShell agent 环境文件 |
| `completions/...` | shell completion 脚本 |
| `skillsets/default.json` | 默认远程 skill discovery profile |

可用环境变量：

| 变量 | 作用 |
|---|---|
| `POPIART_CONFIG_DIR` | 覆盖配置目录 |
| `POPIART_ENDPOINT` | 覆盖 API endpoint |
| `POPIART_KEY` | 覆盖本地保存的 key |
| `POPIART_TOKEN` | `POPIART_KEY` 的兼容别名 |
| `POPIART_PROJECT` | 覆盖当前活动项目 |

## 9. 常见问题

### 9.1 `popiart: command not found`

说明二进制目录还没进 `PATH`。先确认实际安装位置，然后把它加入当前 shell 的 `PATH`。

macOS / Linux 常见位置：

- `/opt/homebrew/bin`
- `/usr/local/bin`
- `~/.local/bin`

Windows 常见位置：

- `%LOCALAPPDATA%\Programs\popiart\bin`

### 9.2 `auth login` 成功了，但 agent 里还是不可用

通常是因为 agent 进程没有拿到同一个 shell / PowerShell 会话里的环境变量。重新执行：

- macOS / Linux：`source ~/.popiart/agents/<agent>/env.sh`
- Windows：`. (Join-Path $HOME ".popiart\agents\<agent>\env.ps1")`

然后从这个会话里启动 agent。

### 9.3 能 `skills get`，但 `run` 提示不能执行

这通常说明你看到的是 CLI bundled helper skill，而不是服务端注册的 runtime skill。先确认：

```sh
popiart skills get <skill-id>
```

如果这是 authoring/helper 入口，请改为选择真正的远端 runtime skill。

### 9.4 我想让结果更适合人看

可以给查看类命令加 `--plain`：

```sh
popiart --plain skills list
popiart --plain auth whoami
```

但如果你要给 agent、脚本或 MCP 层解析，建议保留默认 JSON。
