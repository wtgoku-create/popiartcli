# PopiArt CLI 命令参考

## 查看版本及是否安装成功

```sh
popiart --version
```

## 下载安装更新版本

1. 下载安装最新版本

```sh
popiart update
```

2. 下载安装指定版本

```sh
popiart update --version v1.0.0
```

## 登录相关

1. 登录

```sh
popiart --endpoint https://www.popi.art auth login --key <token>
```

2. 查看当前登录用户

```sh
popiart auth whoami
```

3. 退出当前登录态

```sh
popiart auth logout
```

4. 查看当前保存的 key，脱敏显示

```sh
popiart auth key show
```

5. 直接保存 key，不做在线校验，不推荐

```sh
popiart auth key set <token>
```

## 上传相关

1. 上传本地文件，生成稳定媒体 URL

```sh
popiart media upload <path>
```

常用参数：

- `--filename` 覆盖上传后的文件名
- `--content-type` 覆盖上传内容类型
- `--metadata-json` 额外元数据
- `--project-id` 覆盖默认项目 ID
- `--visibility` 可见性，例如 `private`、`unlisted`、`public`

2. 查询单个媒体信息

```sh
popiart media get <media-id>
```

## 生图相关

1. 文生图：`popiart image generate`

示例：

```sh
popiart image generate --prompt "一只小狗" --wait
```

常用参数：

- `--prompt` 提示词
- `--model` 选择模型
- `--aspect-ratio` 图片比例
- `--size` 图片大小
- `--style` 风格提示，会透传到 `metadata.style`
- `--negative-prompt` 排除项，会透传到 `metadata.negative_prompt`
- `--seed` 随机种子，会透传到 `metadata.seed`
- `--wait` 轮询结果
- `--interval` 轮询间隔毫秒

2. 图生图：`popiart image img2img`

示例：

```sh
popiart image img2img --image ./source.png --prompt "改成黄昏电影感" --wait
```

常用参数：

- `--image` 源图
- `--source-artifact-id` 已上传源图 ID
- `--reference-image` 参考图，可重复传入
- `--reference-artifact-id` 已上传参考图 ID，可重复传入
- `--identity-reference-image` 主体一致性参考图，可重复传入
- `--identity-reference-artifact-id` 已上传主体一致性参考图 ID，可重复传入
- `--style-reference-image` 风格参考图，可重复传入
- `--style-reference-artifact-id` 已上传风格参考图 ID，可重复传入
- `--prompt` 提示词
- `--model` 选择模型
- `--aspect-ratio` 图片比例
- `--size` 图片大小
- `--style` 风格提示，会透传到 `metadata.style`
- `--negative-prompt` 排除项，会透传到 `metadata.negative_prompt`
- `--strength` 转换强度
- `--preserve-composition` 尽量保留原构图
- `--seed` 随机种子，会透传到 `metadata.seed`
- `--wait` 轮询结果

3. 显式图生图入口：`popiart image transform`

示例：

```sh
popiart image transform --image ./source.png --prompt "改成赛博朋克海报风格" --wait
```

4. 识别图片并生成描述性 prompt：`popiart image describe`

示例：

```sh
popiart image describe --image ./source.png --prompt "请写成适合文生图复用的 prompt"
```

常用参数：

- `--image` 源图
- `--from` 等同于 `--image`
- `--source-artifact-id` 已上传源图 ID
- `--prompt` 希望生成什么风格的描述
- `--model` 图像理解模型
- `--wait` 等待任务结果

## 生视频相关

1. 通用视频生成：`popiart video generate`

示例：

```sh
popiart video generate --image ./source.png --prompt "镜头缓慢推进" --wait
```

常用参数：

- `--image` 源图
- `--from` 等同于 `--image`
- `--source-artifact-id` 已上传源图 ID
- `--prompt` 视频提示词
- `--model` 选择模型
- `--prompt-enhancer-model` 先用图像理解模型增强 prompt
- `--aspect-ratio` 视频比例
- `--size` 分辨率，例如 `720P`、`1080P`、`1K`、`2K`、`4K`
- `--duration` 视频时长秒数
- `--fps` 帧率提示
- `--camera-motion` 镜头运动提示
- `--motion-intensity` 运动强度提示
- `--style` 风格提示，会透传到 `metadata.style`
- `--negative-prompt` 排除项，会透传到 `metadata.negative_prompt`
- `--seed` 随机种子，会透传到 `metadata.seed`
- `--wait` 轮询结果

2. 显式图生视频：`popiart video img2video`

示例：

```sh
popiart video img2video --image ./source.png --prompt "让头发和衣摆自然摆动" --wait
```

常用参数：

- `--image` 源图
- `--from` 等同于 `--image`
- `--source-artifact-id` 已上传源图 ID
- `--prompt` 视频提示词
- `--model` 选择模型
- `--prompt-enhancer-model` 先增强 prompt
- `--aspect-ratio` 视频比例
- `--size` 分辨率
- `--duration` 视频时长秒数
- `--fps` 帧率提示
- `--camera-motion` 镜头运动提示
- `--motion-intensity` 运动强度提示
- `--style` 风格提示，会透传到 `metadata.style`
- `--negative-prompt` 排除项，会透传到 `metadata.negative_prompt`
- `--seed` 随机种子，会透传到 `metadata.seed`
- `--wait` 轮询结果

3. 显式 `from-image` 入口：`popiart video from-image`

示例：

```sh
popiart video from-image --image ./source.png --prompt "慢慢推近人物面部" --wait
```

4. 即梦动作迁移：`popiart video action-transfer`

示例：

```sh
popiart video action-transfer --image ./face.jpg --video ./motion.mp4 --wait
```

常用参数：

- `--image` 身份图
- `--video` 动作参考视频
- `--prompt` 可选补充提示
- `--model` 即梦动作迁移模型，默认 `jimeng_dreamactor_m20_gen_video`
- `--action` `metadata.action`，默认 `actionGenerate`
- `--cut-result-first-second-switch` 即梦动作模仿参数
- `--wait` 轮询结果

5. Seedance / 豆包视频：`popiart video seedance`

示例：

```sh
popiart video seedance --prompt "保持主体动作风格一致" --video ./ref.mp4 --wait
```

常用参数：

- `--prompt` 提示词
- `--model` Seedance / 豆包视频模型
- `--image` 参考图片，可重复传入
- `--video` 参考视频，可重复传入
- `--audio` 参考音频，可重复传入
- `--ratio` 视频比例
- `--size` 分辨率，例如 `720p`、`1080p`
- `--duration` 视频时长
- `--frames` 帧数
- `--seed` 随机种子
- `--draft` 样片模式
- `--generate-audio` 生成带声音的视频
- `--return-last-frame` 返回最后一帧
- `--action` 显式 `metadata.action`
- `--service-tier` 服务等级
- `--tools-json` Seedance 2.0 工具 JSON
- `--wait` 轮询结果

## 语音相关

1. 语音合成：`popiart speech synthesize`

示例：

```sh
popiart speech synthesize --text "你好，欢迎使用 PopiArt" --wait
```

常用参数：

- `--text` 文本
- `--text-file` 从文件读取文本
- `--model` 语音模型，默认 `speech-2.8-hd`
- `--voice` 声音 ID
- `--voice-style` 说话风格
- `--emotion` 情绪方向
- `--language` 语言标签
- `--sound-effect` 附加音效提示
- `--format` 输出格式，例如 `mp3`、`wav`
- `--bitrate` 输出码率提示
- `--channels` 声道数
- `--sample-rate-hz` 采样率提示
- `--speed` 语速
- `--volume` 音量
- `--pitch` 音高
- `--pronunciation` 自定义发音映射，可重复传入
- `--subtitles` 返回字幕时间信息
- `--seed` 随机种子
- `--wait` 轮询结果

2. 兼容 TTS 入口：`popiart audio tts`

示例：

```sh
popiart audio tts --text "你好，欢迎使用 PopiArt" --wait
```

## 音乐相关

1. 生成音乐：`popiart music generate`

示例：

```sh
popiart music generate --prompt "Warm upbeat pop" --lyrics "hello hello" --wait
```

常用参数：

- `--prompt` 音乐提示词
- `--lyrics` 歌词
- `--lyrics-file` 从文件读取歌词
- `--model` 音乐模型，默认 `music-2.6`
- `--instrumental` 生成纯音乐
- `--lyrics-optimizer` 自动生成歌词
- `--audio-url` `music-cover` 参考音频 URL
- `--audio-base64` `music-cover` 参考音频 Base64
- `--genre` 音乐流派
- `--mood` 情绪氛围
- `--tempo` 速度描述
- `--bpm` 精确 BPM
- `--key` 调式
- `--instruments` 乐器
- `--vocals` 人声风格
- `--references` 参考曲目或歌手
- `--use-case` 使用场景
- `--structure` 歌曲结构
- `--avoid` 希望避免的元素
- `--extra` 额外细粒度要求
- `--format` 输出格式
- `--output-format` 返回格式，例如 `hex`、`url`
- `--sample-rate-hz` 采样率提示
- `--bitrate` 码率提示
- `--stream` 流式音乐生成
- `--aigc-watermark` 嵌入 AIGC 水印
- `--wait` 轮询结果

## 模型相关

1. 查看模型列表：`popiart models list`

示例：

```sh
popiart models list
popiart models list --capability text2image
```

常用参数：

- `--capability` 按能力过滤，例如 `text2image`、`img2img`、`image2video`
- `--type` 按模型类型过滤
- `--provider` 按供应商过滤

2. 查看默认模型路由结果：`popiart models routes`

示例：

```sh
popiart models routes
popiart models routes --route image.text2image
```

常用参数：

- `--route` 路由键，例如 `image.text2image`、`image.img2img`、`video.image2video`

3. 直接提交模型推理：`popiart models infer`

示例：

```sh
popiart models infer <model-id> --input @input.json --wait
```

常用参数：

- `--input` 输入 JSON、`@file.json` 或 `-`
- `--priority` 作业优先级
- `--idempotency-key` 幂等键
- `--wait` 轮询结果
- `--interval` 轮询间隔毫秒

4. 设置项目级模型覆盖：`popiart models route-override set`

示例：

```sh
popiart models route-override set --project <project-id> --route image.text2image --model <model-id>
```

5. 删除项目级模型覆盖：`popiart models route-override unset`

示例：

```sh
popiart models route-override unset --project <project-id> --route image.text2image
```

6. 查看项目级模型覆盖：`popiart models route-override list`

示例：

```sh
popiart models route-override list --project <project-id>
```

## 技能相关

1. 查看技能列表：`popiart skills list`

示例：

```sh
popiart skills list
popiart skills list --tag image
popiart skills list --search alice
```

常用参数：

- `--tag` 按标签过滤
- `--search` 全文搜索
- `--limit` 最大结果数量
- `--offset` 分页偏移量

2. 查看单个技能详情：`popiart skills get`

示例：

```sh
popiart skills get <skill-id>
```

3. 查看单个技能 schema：`popiart skills schema`

示例：

```sh
popiart skills schema <skill-id>
```

4. 下载 skill 压缩包：`popiart skills pull`

示例：

```sh
popiart skills pull <skill-id-or-url>
```

常用参数：

- `--url` 显式指定压缩包 URL

5. 安装本地 skill：`popiart skills install`

示例：

```sh
popiart skills install ./skill.zip
popiart skills install ./skill.zip --agent codex
```

常用参数：

- `--url` 显式指定压缩包 URL
- `--force` 覆盖已有安装
- `--agent` 安装到指定 agent 的 skill 目录
- `--agent-skill-dir` 显式指定 agent skill 目录

6. 切换为本地优先 skill：`popiart skills use-local`

示例：

```sh
popiart skills use-local <skill-id>
```

常用参数：

- `--agent` 同步到指定 agent skill 目录
- `--agent-skill-dir` 显式指定 agent skill 目录

## run 相关

1. 运行一个 skill：`popiart run`

示例：

```sh
popiart run <skill-id> --input @params.json
```

常用参数：

- `--input` 输入 JSON、`@file.json` 或 `-`
- `--wait` 轮询结果
- `--interval` 轮询间隔毫秒
- `--priority` 作业优先级
- `--idempotency-key` 幂等键

说明：

当前不是任意 skill 都能执行，主要支持已桥接官方 skill，以及映射到这些已桥接官方 skill 的 installed local skill。

## jobs 相关

1. 查看任务状态：`popiart jobs get`

示例：

```sh
popiart jobs get <task-id>
```

2. 等待任务结束：`popiart jobs wait`

示例：

```sh
popiart jobs wait <task-id> --interval 2000
```

常用参数：

- `--interval` 轮询间隔毫秒

3. 查看任务列表：`popiart jobs list`

示例：

```sh
popiart jobs list
popiart jobs list --status running --limit 20 --offset 0
```

常用参数：

- `--status` 按状态过滤
- `--skill` 按技能过滤
- `--project` 按项目过滤
- `--limit` 最大结果数量
- `--offset` 分页偏移量

4. 取消任务

```sh
popiart jobs cancel <task-id>
```

5. 查看任务日志

```sh
popiart jobs logs <task-id>
```

说明：

当前 `jobs cancel` 和 `jobs logs` 在主站模式下会返回 `UNSUPPORTED_IN_POPI_ART_MODE`。

## artifacts 相关

1. 上传本地文件，返回 artifact 外观对象

```sh
popiart artifacts upload <path>
```

示例：

```sh
popiart artifacts upload ./source.png --role source
```

常用参数：

- `--filename` 覆盖上传后的文件名
- `--content-type` 覆盖上传内容类型
- `--role` 上传角色，例如 `source`、`mask`、`reference`
- `--metadata-json` 额外元数据
- `--visibility` 可见性

2. 查看某个任务的结果列表

```sh
popiart artifacts list <task-id>
```

3. 查看单个 artifact 元数据

```sh
popiart artifacts get <artifact-id>
```

4. 下载任务全部结果

```sh
popiart artifacts pull-all <task-id>
```

示例：

```sh
popiart artifacts pull-all <task-id> --dir ./output
```

常用参数：

- `--dir` 输出目录

5. 按单个 artifact 下载

```sh
popiart artifacts pull <artifact-id>
```

说明：

当前 `artifacts pull <artifact-id>` 在主站模式下会返回 `UNSUPPORTED_IN_POPI_ART_MODE`。

## MCP / 初始化相关

1. 一键初始化

```sh
popiart setup --agent codex --completion zsh
```

常用参数：

- `--agent` 目标 agent，可重复传入
- `--completion` 生成 shell completion，可重复传入
- `--key` 直接保存 key
- `--no-agent-config` 跳过 agent env 文件生成

2. 细粒度初始化

```sh
popiart bootstrap --agent codex --discoverable
```

常用参数：

- `--agent` 目标 agent
- `--completion` 生成 completion
- `--discoverable` 一次性生成 discoverability 资产
- `--install-mcp` 生成 MCP 配置
- `--install-skill` 生成 skill wrapper
- `--with-default-skills` 生成默认 skill profile
- `--with-runtime-baseline` 生成 runtime baseline
- `--key` 直接保存 key
- `--no-agent-config` 跳过 agent env 文件生成

3. 启动 MCP server

```sh
popiart mcp serve
```

4. 打印 MCP 配置

```sh
popiart mcp print-config --agent codex
```

5. 诊断 MCP / runtime 状态

```sh
popiart mcp doctor --agent codex
```

常用参数：

- `--agent` 指定 agent 一起检查

## Schema / completion 相关

1. 导出 CLI 命令 schema

```sh
popiart export-schema --format generic
```

示例：

```sh
popiart export-schema --command "video generate" --format openai
```

常用参数：

- `--format` `anthropic`、`openai`、`generic`
- `--command` 只导出指定命令

2. 生成 shell completion

```sh
popiart completion zsh
```

## 当前明确不支持的命令

预算相关：

```sh
popiart budget status
popiart budget usage
popiart budget limits
```

项目详情相关：

```sh
popiart project list
popiart project get <project-id>
popiart project context
```

说明：

这些命令当前在主站迁移模式下会明确返回 `UNSUPPORTED_IN_POPI_ART_MODE`。
