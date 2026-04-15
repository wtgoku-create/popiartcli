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
- `popiart update` 可以解析默认仓库，也可以解析 GitHub / Gitee 仓库主页、`releases` 页和 `releases/tag/vX.Y.Z` URL
- 国内镜像默认约定为 `https://gitee.com/wattx/popiartcli`
- 但它最终仍依赖对应 release 中的目标平台二进制；如果某个 tag 只有源码归档、没有 release 二进制，`popiart update` 不能直接升级

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
popiart setup --agent codex --completion zsh
```

如果你希望安装完成后，对应 agent 立刻能在原生 MCP / skill 目录中发现 `PopiArt`：

```sh
popiart setup --agent codex
```

`popiart setup --agent codex` 默认会同时写两类产物：

- `~/.popiart/agents/<agent>/` 下的 bootstrap 资产
- agent 原生配置和原生 skill 目录

如果你需要细粒度控制，仍然可以继续使用 `popiart bootstrap --discoverable`。

### 2.2 官方安装脚本

只安装 CLI：

```sh
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | sh

# 后续升级到最新 release
popiart update
```

国内镜像：

```sh
curl -fsSL https://gitee.com/wattx/popiartcli/raw/main/install.sh | sh -s -- --source gitee
popiart update --source gitee
```

安装 CLI，并继续做默认初始化：

```sh
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | sh
popiart setup --agent codex --completion zsh
```

安装指定版本：

```sh
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | \
  env VERSION=v0.3.7 sh

# 或者在已安装后更新到指定版本
popiart update --version v0.3.7
```

如果你希望显式指定 Gitee 仓库主页或 tag 页，也可以：

```sh
popiart update --repo https://gitee.com/wattx/popiartcli
popiart update --repo https://gitee.com/wattx/popiartcli/releases/tag/v0.3.7
```

脚本会优先尝试：

- Homebrew 的 `bin` 目录
- `/opt/homebrew/bin`
- `/usr/local/bin`
- `~/.local/bin`

如果安装目录不在 `PATH` 中，脚本会打印对应 shell 的追加方法。

### 2.3 GitHub Releases 手动安装

```sh
curl -fsSL https://github.com/wtgoku-create/popiartcli/releases/download/v0.3.7/popiart_0.3.7_darwin_arm64.tar.gz -o popiart.tar.gz
tar -xzf popiart.tar.gz
install -m 0755 popiart /usr/local/bin/popiart
```

国内镜像：

```sh
curl -fsSL https://gitee.com/wattx/popiartcli/releases/download/v0.3.7/popiart_0.3.7_darwin_arm64.tar.gz -o popiart.tar.gz
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

源码安装的后续更新方式：

```sh
git pull --tags
go install ./cmd/popiart
```

## 3. Linux 安装

### 3.1 官方安装脚本

只安装 CLI：

```sh
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | sh

# 后续升级到最新 release
popiart update
```

国内镜像：

```sh
curl -fsSL https://gitee.com/wattx/popiartcli/raw/main/install.sh | sh -s -- --source gitee
popiart update --source gitee
```

安装 CLI 并做默认初始化：

```sh
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | sh
popiart setup --agent claude-code --completion bash
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
curl -fsSL https://github.com/wtgoku-create/popiartcli/releases/download/v0.3.7/popiart_0.3.7_linux_amd64.tar.gz -o popiart.tar.gz
tar -xzf popiart.tar.gz
install -m 0755 popiart "$HOME/.local/bin/popiart"
```

国内镜像：

```sh
curl -fsSL https://gitee.com/wattx/popiartcli/releases/download/v0.3.7/popiart_0.3.7_linux_amd64.tar.gz -o popiart.tar.gz
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

源码安装的后续更新方式：

```sh
git pull --tags
go install ./cmd/popiart
```

## 4. Windows 安装

### 4.1 PowerShell 安装脚本

只安装 CLI：

```powershell
irm https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.ps1 | iex

# 后续升级到最新 release
popiart update
```

国内镜像：

```powershell
& ([scriptblock]::Create((irm https://gitee.com/wattx/popiartcli/raw/main/install.ps1))) -Source gitee
popiart update --source gitee
```

安装指定版本：

```powershell
$env:VERSION="v0.3.7"
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
$version = "0.3.7"
$zip = "popiart_${version}_windows_amd64.zip"
Invoke-WebRequest "https://github.com/wtgoku-create/popiartcli/releases/download/v$version/$zip" -OutFile $zip
Expand-Archive $zip -DestinationPath .
New-Item -ItemType Directory -Force "$env:LOCALAPPDATA\Programs\popiart\bin" | Out-Null
Copy-Item .\popiart.exe "$env:LOCALAPPDATA\Programs\popiart\bin\popiart.exe" -Force
```

国内镜像：

```powershell
$version = "0.3.7"
$zip = "popiart_${version}_windows_amd64.zip"
Invoke-WebRequest "https://gitee.com/wattx/popiartcli/releases/download/v$version/$zip" -OutFile $zip
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

源码安装的后续更新方式：

```powershell
git pull --tags
go install ./cmd/popiart
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
popiart auth login --key <product-key>
```

验证当前身份：

```sh
popiart auth whoami
popiart auth key show
```

说明：

- `popiart` 里保存的是 PopiArt 产品层 key
- 不要把 OpenAI、Gemini、Kling、Runway 等 provider key 直接塞进 CLI
- 如果服务端给你的产品层 key 前缀是 `sk-...`，也可以直接用于 `auth login`；CLI 不会强制要求 `pk-...`
- `auth login` 成功后，本地配置里看到 `sess_...` 这类服务端签发的会话令牌也是正常现象
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

如果要同时放进 agent 原生 skills 目录：

```sh
popiart skills install ./popiskill-audio-avatar-omnishuman-v1.zip --agent codex
popiart skills install ./popiskill-audio-avatar-omnishuman-v1.zip --agent claude-code
popiart skills install ./popiskill-audio-avatar-omnishuman-v1.zip --agent openclaw
popiart skills install ./popiskill-audio-avatar-omnishuman-v1.zip --agent opencode
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

关系说明：

- `popiart skills list/get/schema` 先看 `popiartServer` 暴露的远程 runtime skill 注册表，再合并本地 installed skill、CLI 内置 official runtime baseline 和 bundled seed。
- 当前公开定义参考仓库是 `wtgoku-create/Popiart_skillhub`，但真正可执行的 skill 集合仍以服务端 `/skills` 返回为准。
- `popiart bootstrap` 写出的 `default` skillset 只是远程发现查询 + seed 元数据，不代表这些 skill 都已经在服务端注册完成。
- 返回里的 `source` 字段会标明当前结果来自 `remote`、`installed`、`official-runtime` 或 `bundled-seed`。

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

`popiskill-video-image2video-basic-v1` 现在按安装后自带的官方 skill 处理。它应该能直接出现在 `skills list/get/schema` 里；如果远端目录里的同名条目仍是占位符或尚未注册，CLI 会自动桥接到底层 `models infer`，先试 `viduq3-turbo`，失败再回落到 `viduq2-pro-fast`。

本地图片要进入 `image2video` 时，也建议走同一条 artifact 链路：

```sh
ART=$(popiart artifacts upload ./source.png --role source | jq -r '.data.artifact_id')

popiart run popiskill-video-image2video-basic-v1 --project proj_local_dev --input "{
  \"source_artifact_id\":\"$ART\",
  \"prompt\":\"让人物衣摆和发丝在微风中轻轻摆动，镜头缓慢推进，整体保持真实电影感。\",
  \"aspect_ratio\":\"16:9\",
  \"seconds\":5
}" --wait
```

## 7. Agent 如何使用

### 7.1 先理解 agent 接入原则

- `popiart` 默认输出 JSON，这通常比 `--plain` 更适合 agent
- agent 应该拿 PopiArt 产品层 key，而不是 provider key
- agent 应该先 `skills get` / `skills schema`，再决定是否 `run`
- agent 应优先使用 `@params.json` 或 stdin，而不是把大段 JSON 内联到命令里
- bootstrap 生成的本地 bundled seed skills 现在对应官方 runtime baseline，本地 schema 可见性和远端可执行性仍然要分开判断

最重要的一条：

```text
能在 `skills get/schema` 里看到，不代表一定能直接 `run`
```

例如 `popiskill-image-text2image-basic-v1` 能在本地 `skills get/schema` 里拿到官方契约，但如果服务端还没注册对应 runtime skill，真正的 `popiart run` 仍然会失败。当前只有 `popiskill-video-image2video-basic-v1` 在远端目录缺失或仍是占位符时，CLI 会自动桥接到底层 `models infer`。

### 7.2 聊天附件如何进入 img2img / image2video

如果 agent 聊天里收到用户上传的图片，不要直接把图片二进制塞进 `run`。

当前推荐顺序是：

1. 宿主先把聊天附件保存到本地临时文件路径。
2. 调用 `popiart artifacts upload <path> --role source`。
3. 读取返回的 `artifact_id`。
4. 再调用 `popiart run popiskill-image-img2img-basic-v1` 或 `popiart run popiskill-video-image2video-basic-v1`，把它放进 `source_artifact_id`。

示例：

```sh
ART=$(popiart artifacts upload /tmp/chat-upload.png --role source | jq -r '.data.artifact_id')

popiart run popiskill-image-img2img-basic-v1 --input "{
  \"source_artifact_id\":\"$ART\",
  \"prompt\":\"保留主体身份与主要视觉特征，改成海边黄昏场景\"
}" --wait
```

```sh
popiart run popiskill-video-image2video-basic-v1 --project proj_local_dev --input "{
  \"source_artifact_id\":\"$ART\",
  \"prompt\":\"保持人物身份和构图，让头发和衣摆有自然风动，镜头轻微推进。\",
  \"aspect_ratio\":\"16:9\",
  \"seconds\":5
}" --wait
```

如果聊天附件本身已经有可访问 URL，也可以直接走 `reference_image_url` / `image_url`，不一定要先上传。对于 `popiskill-video-image2video-basic-v1`，CLI 会把 `reference_image_url` 自动归一化到 `image_url`，并把 `seconds` 归一化到 `duration_s` 后再提交。

### 7.4 当前已验证的服务端 `img2img` / `image2video` 路由

截至 `2026-03-28`，测试环境里已经验证过两条服务端图像编辑适配：

- `gemini-3-pro-image-preview`
  通过 Gemini `generateContent` 路由处理参考图编辑
- `seedream-4-5-251128`
  通过 `/v1/images/generations` + 参考图输入处理图生图

注意：

- 这两条能力属于 `popiartServer` / `PopiNewAPI` 的服务端路由适配，不是 CLI 本身直接决定的
- `seedream-4-5-251128` 对输出尺寸有最小像素限制。CLI 可以继续传递像 `1024x1536` 这样的安全预设，但服务端可能会把它抬升到满足模型要求的尺寸后再提交
- 截至 `2026-04-08`，CLI 内置 `image2video` fallback 的模型顺序是 `viduq3-turbo -> viduq2-pro-fast`
- 截至 `2026-03-28`，当前测试环境里已验证通过的服务端 `image2video` 路由是 `video.image2video -> viduq2-pro-fast`
- 如果服务端将来补齐真正 runtime skill，CLI 会优先走服务端 skill；否则继续走内置 fallback

### 7.3 让 agent 获得稳定环境

`popiart bootstrap` 现在会做四件有价值的事：

- 生成 shell completion
- 生成 agent 专用环境注入文件
- 可选生成默认 skill discovery profile
- 当传入 `--install-mcp`、`--install-skill` 或 `--discoverable` 时，同时写 agent 原生 MCP / skill 目录

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
- 如果你用了 `--discoverable`，还会同时得到这些原生落点：
  - `codex`: `~/.codex/config.toml` + `~/.codex/skills/popiart/`
  - `claude-code`: `~/.claude.json` + `~/.claude/skills/popiart/`
  - `openclaw`: `~/.openclaw/mcp.json` + `~/.openclaw/skills/popiart/`
  - `opencode`: `~/.config/opencode/mcp.json` + `~/.config/opencode/skill/popiart/`

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

当 agent 接到生图 / 生视频 / 生音频需求时，推荐按下面顺序执行。关键要求是：一旦已经拿到用户原始创作需求，后续即使经历登录、充值、失败重试，也不要要求用户重复描述需求。

1. 先检测 CLI 是否可用：

```sh
popiart --help
```

如果不可用，可并行告知安装和账号准备：

- 统一从 GitHub 仓库入口开始：`https://github.com/wtgoku-create/popiartcli`
- 如果执行者是人类用户，按该仓库当前公开文档选择安装方式
- 如果执行者是 agent，且当前环境允许执行 shell 安装，就直接自动安装，不要只停在“提示用户去装”
- 当前仓库公开的安装方式包括：

```sh
git clone https://github.com/wtgoku-create/popiartcli.git
cd popiartcli
go install ./cmd/popiart
popiart --help
```

- 或按仓库 README 使用 Homebrew:

```sh
brew tap wtgoku-create/popi
brew install wtgoku-create/popi/popiart
```

- 或按仓库 README 使用安装脚本:

```sh
curl -fsSL https://raw.githubusercontent.com/wtgoku-create/popiartcli/main/install.sh | sh
```

2. 检测登录态：

```sh
popiart auth whoami
```

如果返回未认证，再引导用户去已确认的 skillhub 站点 `https://wwwskillhub.popi.art` 注册、充值并获取产品层 key，然后执行：

```sh
popiart auth login --key <product-key>
popiart auth whoami
popiart auth key show
```

说明：

- `--key` 是当前主入口
- `--token` 仅作为兼容旧用法的别名保留
- `~/.popiart/config.json` 里保存的通常是服务端签发后的会话 key，不要求和用户输入的原始 key 字面量完全一致

3. 如果账号下有多个项目，先确认当前项目：

```sh
popiart project current
popiart project list
popiart project use <project-id>
```

4. 发现和确认 skill。先按媒介类型过滤，再看 schema，不要跳过这一步：

```sh
popiart skills list --tag image
popiart skills list --tag video
popiart skills list --tag audio
popiart skills get <skill-id>
popiart skills schema <skill-id>
```

5. 余额预检要按“有站点、无 CLI 命令”的现实能力处理：

- 当前 `popiartcli` 没有名为 `balance` / `credits` / `quota` 的独立命令，但已经提供：

```sh
popiart budget status
popiart budget usage --group-by skill
popiart budget limits
```

- 如果要在提交前确认余额，或服务端已经返回积分不足，直接打开 `https://wwwskillhub.popi.art` 引导用户充值
- 如果当前 agent 具备浏览器能力，应直接打开该站点；否则至少明确给出该链接
- 充值完成后继续使用已保留的原始需求直接重试，不要让用户重写需求

6. 根据 `skills schema` 构造 `params.json`，优先使用文件输入：

```sh
popiart run <skill-id> --input @params.json --wait
```

如果需要安全重试，追加幂等键，避免 agent 重放导致重复扣费：

```sh
popiart run <skill-id> --input @params.json --idempotency-key req-001 --wait
```

7. 如果不使用 `--wait`，就显式等待：

```sh
popiart jobs wait <job-id>
```

8. 如果任务失败，至少回传这些信息给用户：

- `job_id`
- `status`
- `error.code`
- `error.message`

必要时继续查看日志：

```sh
popiart jobs logs <job-id>
```

9. 如果任务完成，拉取全部产物：

```sh
popiart artifacts pull-all <job-id> --dir ./output/
```

10. 展示产物时补充当前已知限制：

- 当前 CLI 的 `job` / `artifact` 结构没有专门的“本次消耗金额”和“剩余余额”字段
- 如果服务端未来在 job 响应里补充计费信息，agent 可以直接展示
- 在此之前，在线查看积分与充值应回到 `https://wwwskillhub.popi.art`

只要 `popiart auth whoami` 仍然成功，后续再次使用通常可以直接从 skill 发现步骤开始，而不需要重新安装 CLI。

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
| `config.json` | endpoint、已保存的 key / session token、project 等本地配置 |
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
