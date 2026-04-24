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

## 默认入口

推荐优先记住这几个入口：

- `popiart setup --agent codex`
- `popiart image generate`
- `popiart image describe`
- `popiart image img2img`
- `popiart video generate`
- `popiart video img2video`
- `popiart video action-transfer`
- `popiart speech synthesize`
- `popiart music generate`

它们是面向新用户和 agent 的 opinionated façade，内部仍然映射到官方 runtime skill，不改变底层架构。

## 当前保证范围

- 仓库中的权威实现是 Go CLI：`cmd/popiart`。根目录 `package.json` 只保留仓库任务入口，不再代表一个正式发布的 Node CLI。
- `popiart setup --agent ...` 会优先把 PopiArt 做到 agent 可发现，但“可发现”不等于“远端 runtime 已就绪”。
- `popiart mcp doctor` 现在会分别返回 `discoverability_status` 和 `runtime_status`。
- 当前 MCP server 重点实现的是 `tools/list` / `tools/call` 工具面；`resources`、`prompts`、`sampling` 仍未完成。
- 七个 official runtime baseline skill 已有 discoverability 契约，但 CLI 目前仍不能单独保证七个 skill 都能端到端执行成功。

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

如果你刚完成初始化，推荐先运行：

```sh
popiart mcp doctor --agent codex
```

判读方式：

- `discoverability_status=pass`：本地 agent 原生 MCP / skill 入口大致已就位
- `runtime_status=pass`：远端登录态、baseline skill 注册与默认路由更接近可执行
- 两者都通过之前，不要把 `setup` 视为“已经可端到端跑通”

## 用户意图命令面

目前已经提供的 façade：

```sh
popiart image generate --prompt "..."
popiart image img2img --image ./source.png --prompt "..."
popiart video generate --image ./source.png --prompt "..."
popiart video img2video --image ./source.png --prompt "..."
popiart speech synthesize --text "..."
popiart music generate --prompt "..." --lyrics "..."
```

它们当前分别映射到：

- `popiskill-image-text2image-basic-v1`
- `popiskill-image-img2img-basic-v1`
- `popiskill-video-image2video-basic-v1`
- `popiskill-video-image2video-basic-v1`
- `speech-2.8-hd` (MiniMax direct infer by default)
- `music-2.6-free` (MiniMax direct infer by default)

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

## 图片输入

`popiart` 在图片命令里支持 4 类常见输入：

| 输入类型 | 示例 | 适用场景 |
|---|---|---|
| 本地文件 | `./source.png` | 图片已经在本机磁盘上 |
| 稳定媒体 URL | `https://server.popi.art/v1/media/med_xxx/content` | 多轮任务复用或跨步骤传递 |
| artifact_id | `art_xxx` | 已经在 PopiArt runtime 内部存在的输入或产物 |
| data URL | `data:image/png;base64,...` | 图片只以内联形式存在时的兜底输入 |

输入决策规则：

- 已有本地文件时，优先直接传本地文件，façade 命令会在需要时自动上传。
- 已有稳定 URL 时，优先直接传 URL，适合 `img2img`、`img2video` 和多轮 agent 工作流。
- 已在 PopiArt 内部存在的图片，优先传 `artifact_id`，最适合 runtime 链路复用。
- `data URL` 适合作为内联兜底输入；能用本地文件或稳定 URL 时，优先不用 `data URL`。

概念边界：

- `artifact`：PopiArt runtime 内部的稳定引用，适合同一任务链路内复用。
- `media URL`：可直接被后续命令消费的稳定地址，适合跨步骤、跨会话传递。

多图任务可以混用本地文件、URL 和 artifact，但为了降低不可见差异，建议同一组参考图尽量使用同一类输入。

## Role-Aware Img2Img

复杂编辑时，推荐把多张图的角色明确表达出来：

- `--image`：源场景图
- `--identity-reference-image`：角色一致性参考图
- `--style-reference-image`：风格参考图
- `--preserve-composition`：尽量保留源场景机位、动作和构图

标准三图示例：

```sh
popiart image img2img \
  --image https://server.popi.art/v1/media/med_scene/content \
  --identity-reference-image https://example.com/identity.jpg \
  --style-reference-image https://example.com/style.png \
  --prompt "Replace the person in the source scene with the main character from the identity reference. Keep the exact action and framing from the source scene. Apply only the style from the style reference." \
  --preserve-composition \
  --wait \
  --output json \
  --quiet \
  --non-interactive
```

复杂任务建议优先两步法：

1. 先做 `source + identity`，把角色和动作锁住。
2. 再用第一步结果做单独风格迁移。

建议先用 `--dry-run` 查看最终请求形状，再执行真实任务。

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
- `popiart export-schema ...`
  导出 CLI 自身命令的 tool JSON schema，适合动态注册到 Anthropic / OpenAI 等 agent 框架。
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

## CLI Tool Schema 导出

如果你要把 `popiart` 的 CLI 命令动态注册为 agent tools，而不是手写 schema，可以直接导出 CLI 自身的命令结构：

```sh
# 导出所有可执行 leaf 命令的 Anthropic-compatible tool schema
popiart export-schema --format anthropic

# 导出所有命令的 OpenAI-compatible function tool schema
popiart export-schema --format openai

# 只导出一个命令
popiart export-schema --command "video generate" --format openai
popiart export-schema --command "models route-override set" --format generic
```

这里导出的不是远程 `skills schema`，而是 **CLI 自身命令** 的结构。

当前支持：

- `anthropic`
- `openai`
- `generic`

这个命令会直接输出原始 JSON schema，不包在 `{ ok, data }` envelope 里，方便直接喂给工具注册逻辑。

## MiniMax Support

当前能力面里，MiniMax 相关支持分成 3 组：

- `music` / `speech`
  - 通过 `models infer` 直连 MiniMax 音乐与语音模型
- `image-01` / `image-01-live`
  - 已在测试环境验证 `popiart image generate` / `popiart image img2img` 可以通过服务端链路成功出图
- Hailuo / T2V / I2V / S2V 视频模型
  - 已在测试环境验证 `MiniMax-Hailuo-2.3` 文生视频、图生视频，以及 `MiniMax-Hailuo-02` 首尾帧视频可以成功提交并产出视频
  - `S2V-01` 已验证能通过 `popiart` 正确提交主体参考视频任务
- 即梦动作迁移
  - `popiart video action-transfer` 默认使用 `jimeng_dreamactor_m20_gen_video`
  - CLI 会提交 `images[0]`、`videos[0]`、`metadata.action=actionGenerate`
  - 即梦图片 data URL 会自动剥离 `data:image/*;base64,` 前缀，避免上游 base64 解码失败
  - 已在测试服 `http://101.42.99.35:18080/v1` 验证 5 秒动作迁移预览可完成并返回 MP4 artifact

```sh
popiart music generate \
  --prompt "Upbeat pop" \
  --lyrics "La la la" \
  --output-format url \
  --format mp3 \
  --output json \
  --quiet \
  --non-interactive

popiart speech synthesize \
  --text "Hello world" \
  --output json \
  --quiet \
  --non-interactive
```

约定：

- `music` 默认模型：`music-2.6-free`
- `speech` / `audio tts` 默认模型：`speech-2.8-hd`
- 显式传 `--model` 时，会改为本次请求 direct model override
- `music --instrumental` 会映射为网关 `is_instrumental`；`--output-format` 映射为 `output_format`；`--format`、`--sample-rate-hz`、`--bitrate` 会写入 `audio_setting`
- 这两条命令当前走 `models infer`，不是远程 `skills schema`

MiniMax 图片 / 视频更推荐显式指定模型：

```sh
popiart image generate \
  --model image-01 \
  --prompt "A watercolor cat portrait" \
  --aspect-ratio 4:3 \
  --wait \
  --output json \
  --quiet \
  --non-interactive

popiart image img2img \
  --model image-01 \
  --image https://example.com/reference.jpg \
  --prompt "Turn this into a poster-style portrait" \
  --aspect-ratio 3:4 \
  --wait \
  --output json \
  --quiet \
  --non-interactive

popiart models infer MiniMax-Hailuo-2.3 \
  --input '{"prompt":"A person picks up a book and reads it quietly.","duration":6,"size":"1080P"}' \
  --wait
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
