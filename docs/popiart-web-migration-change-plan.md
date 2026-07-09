# PopiArt CLI 迁移到 `www.popi.art` 修改计划

Date: `2026-06-10`

本文档基于 [docs/popiart-web-migration-plan.md](./popiart-web-migration-plan.md) 生成，目标是把方案文档落成一份可执行的代码修改计划，明确本次迁移的代码边界、实施顺序、测试范围，以及中文注释限制。

## 1. 目标

本次修改计划只服务于以下目标：

- 保持现有 CLI 命令入口基本不变
- 把主执行链路从旧后端接口切到 `https://www.popi.art`
- 统一走主站 `task` 能力完成图片、视频、音频、LLM 任务
- 把模型选择、模型能力校验、媒体上传、任务轮询前移到 CLI 内部

本次修改计划不覆盖：

- `skills` / `run` / `mcp` / `bootstrap` 的最终保留策略
- 网站 UI 规则
- 与迁移主链路无关的历史清理

## 2. 总体实施原则

### 2.1 命令层保持薄

- `internal/cmd` 只负责：
  - Cobra 命令定义
  - flags 解析
  - 输出格式兼容
  - 调用迁移后的领域服务
- `internal/cmd` 不再直接拼装大量 `popi.art` 接口细节
- 不能继续把 `/api_client/...` 路径、字段映射、模型能力判断散落在多个命令函数里

### 2.2 新主站适配逻辑集中收口

建议新增一个独立边界层，例如：

- `internal/popiart/auth.go`
- `internal/popiart/media.go`
- `internal/popiart/models.go`
- `internal/popiart/tasks.go`
- `internal/popiart/errors.go`
- `internal/popiart/defaults.go`

该层统一负责：

- 主站接口路径
- 请求/响应结构
- 显式 `aiModelId` 查找和默认 code 候选解析
- 模型能力校验
- task 创建与轮询
- 主站错误到 CLI 错误码的映射

### 2.3 旧链路先隔离，不立即硬删

- 旧 `/jobs`、`/models/infer`、`/video/generations`、`/auth/*` 代码先不大面积删除
- 第一阶段优先让用户向命令切到新链路
- 平台向命令和历史兼容命令按文档做“保留、降级、unsupported”处理

## 3. 代码边界

### 3.1 必改文件

以下文件属于迁移主链路，必须纳入本次修改：

| 边界 | 当前文件 | 本次职责 |
| --- | --- | --- |
| 配置默认值 | `internal/config/config.go` | 默认 endpoint 改为 `https://www.popi.art`，保留 `token/project` 读取逻辑 |
| 通用 client | `internal/api/client.go` | 保持 HTTP 基础能力，补足主站请求所需的 query / envelope / 上传支持 |
| 认证命令 | `internal/cmd/auth.go` | 改成 `GET /api_client/users/user/info` 校验 token，`logout` 只清本地 token，`key rotate` 返回 `unsupported` |
| 命令公共入口 | `internal/cmd/common.go` | `currentClient()` 继续作为 client 注入点，但不承担业务映射 |
| 意图命令主入口 | `internal/cmd/intent_commands.go` | `image/video/audio/speech/music` 改为调用新的主站任务执行层 |
| 图片理解 | `internal/cmd/image_describe.go` | 从 `/models/infer` 迁到 `task/create` 的 LLM 路径 |
| 媒体命令 | `internal/cmd/media.go` | `upload` 迁到 `/api_client/media/upload`，`get` 改为 `unsupported` |
| 作业命令 | `internal/cmd/jobs.go` | `get/wait/list/cancel/logs` 重新定义为 task 语义或按阶段降级 |
| 工件命令 | `internal/cmd/artifacts.go` | `list/pull-all` 改读 task 结果，`get/pull` 改为 `unsupported` |
| 模型命令 | `internal/cmd/models.go` | `list/routes` 改读 `ai/model/list` 与本地默认映射，`infer/route-override` 停止走旧中心路由 |
| 轮询能力 | `internal/poll/poll.go` | 从 job 轮询扩展为 task 轮询，或拆出新的 task poller |

### 3.2 建议新增文件

为避免把迁移逻辑继续塞进 `internal/cmd`，建议新增以下文件：

| 新文件 | 职责 |
| --- | --- |
| `internal/popiart/types.go` | 主站接口结构体、枚举、任务状态定义 |
| `internal/popiart/auth.go` | `user/info` 查询与 token 验证 |
| `internal/popiart/media.go` | 媒体上传、重试、URL 提取 |
| `internal/popiart/models.go` | 模型列表获取、默认模型选择、能力校验 |
| `internal/popiart/tasks.go` | `task/create`、`task/detail`、`downloadUrls`、状态映射 |
| `internal/popiart/mapper.go` | CLI flags 到主站请求体映射 |
| `internal/popiart/errors.go` | 主站错误文案到 CLI 错误码映射 |
| `internal/popiart/defaults.go` | 内部默认模型 code 候选配置 |

### 3.3 第一阶段明确不改的文件

以下文件第一阶段不应该承载主迁移逻辑，只允许做最小兼容修补：

- `internal/cmd/skills.go`
- `internal/cmd/run.go`
- `internal/cmd/mcp.go`
- `internal/cmd/mcp_server.go`
- `internal/cmd/bootstrap.go`
- `internal/cmd/local_skills_cmd.go`
- `internal/cmd/project.go`
- `internal/cmd/budget.go`
- `internal/cmd/official_runtime.go`

限制：

- 不在这些文件里新增主站 task 映射规则
- 不在这些文件里复制一份模型能力校验逻辑
- 若平台向命令暂时还依赖旧接口，应明确标注“兼容态”或“unsupported”

## 4. 实施拆分
注意：修改/新增的代码加上标准格式的中文注释

### 4.1 阶段一：配置与认证切换

目标：

- 切换默认 endpoint
- 切换 token 语义
- 确保 `auth` 子命令符合迁移文档

改动边界：

- `internal/config/config.go`
- `internal/cmd/auth.go`
- `internal/cmd/common.go`
- `internal/api/client.go`

完成标准：

- `auth login --key <token>` 调 `GET /api_client/users/user/info`
- `auth whoami` 调 `GET /api_client/users/user/info`
- `auth logout` 不再依赖远端登出成功
- `auth key rotate` 返回 `UNSUPPORTED_IN_POPI_ART_MODE`

### 4.2 阶段二：主站基础能力层落地

目标：

- 新增主站适配层
- 收口接口路径、请求结构、错误映射

改动边界：

- 新增 `internal/popiart/*`
- `internal/api/client.go`

完成标准：

- 命令层不直接关心 `aiModelId` 解析细节
- 命令层不直接关心 `/api_client/anime/task/*` 路径
- 上传、模型查询、任务创建、任务查询、下载地址查询都可由领域层单测覆盖

### 4.3 阶段三：媒体上传迁移

目标：

- 统一使用 `/api_client/media/upload`
- 统一返回上传后的稳定 `url`

改动边界：

- `internal/popiart/media.go`
- `internal/cmd/media.go`
- 被图片、视频、音频命令复用的本地文件解析逻辑

完成标准：

- 单文件字段名固定为 `file`
- 仅对网络错误和 5xx 做最多 3 次重试
- 4xx 直接失败
- `media get` 改为 `unsupported`

### 4.4 阶段四：模型中心能力层落地

目标：

- 从“命令内硬编码默认模型”转成“内部默认模型 code 候选 + 运行时解析”
- 把模型能力判断写成统一前置步骤

改动边界：

- `internal/popiart/models.go`
- `internal/popiart/defaults.go`
- `internal/cmd/models.go`

完成标准：

- 显式用户 `--model` 按主站 `aiModelId` 匹配
- 内部默认候选支持 `code` / `aiModelCodeAlias` 匹配
- 统一检查：
  - `categories[].taskSubType`
  - `isSupportImages`
  - `isSupportVideos`
  - `isSupportAudios`
  - `ratio` / `videoRatio`
  - `resolution`
  - `duration`
  - `uploadImageLimit`
- 用户显式 `--model` 不允许静默切换模型

### 4.5 阶段五：用户向命令切到 task 主链路

目标：

- 把 `image/video/audio/speech/music/image describe` 全部切到 `task/create`

改动边界：

- `internal/cmd/intent_commands.go`
- `internal/cmd/image_describe.go`
- `internal/popiart/mapper.go`
- `internal/popiart/tasks.go`

完成标准：

- `image generate` -> `type=1/subType=103`
- `image img2img` / `transform` -> `type=1/subType=103`
- `image describe` -> `type=5/subType=501`
- `audio tts` / `speech synthesize` / `music generate` -> `type=3/subType=301`
- `video` 系列按迁移文档中的 `action/subType` 规则映射
- 本地图片、视频、音频一律先上传，再写入 `images[]/videos[]/voices[]`

### 4.6 阶段六：等待、结果和输出兼容

目标：

- `--wait` 不再轮询 job，改为轮询 task
- 兼容输出 `job_id`

改动边界：

- `internal/poll/poll.go`
- `internal/popiart/tasks.go`
- `internal/cmd/intent_commands.go`
- `internal/cmd/image_describe.go`

完成标准：

- `GET /api_client/anime/task/detail?id=...` 成为统一轮询入口
- 成功后补一次 `GET /api_client/anime/task/downloadUrls`
- 输出同时保留：
  - `job_id`
  - `task_id`
- 失败映射为 `JOB_FAILED`
- 超时映射为 `POLL_TIMEOUT`

### 4.7 阶段七：平台向命令兼容与降级

目标：

- 把需要保留的命令改成新语义
- 把不该伪造旧对象的命令明确改为 `unsupported`

改动边界：

- `internal/cmd/artifacts.go`
- `internal/cmd/jobs.go`
- `internal/cmd/models.go`
- 必要时补 `internal/popiart/tasks.go`

完成标准：

- `artifacts list <task-id>` 改为 task 结果列表
- `artifacts pull-all <task-id>` 改为批量下载 `downloadUrls`
- `artifacts get/pull` 直接 `unsupported`
- `models routes` 改为展示默认模型选择结果
- `jobs` 命令若无法自然映射，应在阶段内明确保留策略，不允许半旧半新

## 5. 测试边界

### 5.1 必补单测

- `internal/config/config_test.go`
- `internal/api/client_test.go`
- `internal/poll/poll_test.go`
- `internal/cmd/auth_test.go` 或现有顶层命令测试
- `internal/cmd/media_test.go`
- `internal/cmd/artifacts_test.go`
- `internal/cmd/models_test.go`
- `internal/cmd/intent_commands_test.go`
- `internal/cmd/top_level_command_execute_test.go`

### 5.2 测试重点

- endpoint 默认值变化
- `Authorization: Bearer <token>` 是否正确发送
- 主站 envelope / 错误结构是否能正确解码
- 本地文件上传是否只用字段 `file`
- 显式 `--model` 不支持 `subType` 时是否直接失败
- `--wait` 是否改为 task 轮询
- `download_urls/result_urls` 是否进入最终输出
- `unsupported` 命令是否给出稳定错误码与提示

## 6. 风险控制

### 6.1 不能边迁移边继续扩散旧抽象

禁止做法：

- 在 `intent_commands.go` 中继续直接拼 `/models/infer` 或 `/video/generations`
- 在多个命令函数中重复写一套 `subType` 判断
- 在命令层手写多个版本的上传重试逻辑

### 6.2 不能把平台命令伪装成完全兼容

禁止做法：

- 为 `artifacts get/pull` 伪造不存在的稳定 `artifact_id`
- 为 `models routes` 伪造旧中心路由概念
- 为 `jobs cancel/logs` 提供没有后端承接的假语义

## 7. 中文注释限制

本次迁移必须显式限制中文注释数量，避免把代码改成“注释驱动实现”。

规则如下：

- 默认不新增解释性中文注释，优先用清晰命名表达逻辑
- 只有在以下场景允许新增中文注释：
  - 兼容层适配存在非直觉行为
  - 主站字段语义与 CLI 语义明显不同
  - 某段校验顺序必须严格遵守迁移文档
- 每个新增文件最多添加 3 处中文注释块
- 每处中文注释最多 2 行
- 禁止添加“翻译代码本身”的中文注释
  - 例如“给变量赋值”“调用接口”“循环任务列表”这类注释一律不允许
- 禁止在 Cobra flag 定义附近堆叠大段中文实现说明
- 用户可见帮助文案、错误信息、命令描述不属于这里说的“中文注释”，可正常保留中文

建议执行方式：

- 先写实现，再回看是否真的需要注释
- 如果一段逻辑需要超过 2 行中文注释才能解释清楚，应优先继续拆函数

## 8. 建议提交顺序

建议按以下提交顺序拆分变更，避免一个超大补丁同时改配置、命令、模型、轮询：

1. 配置与 `auth` 切换
2. 新增 `internal/popiart` 基础适配层
3. 媒体上传迁移
4. 模型查询与能力校验
5. `image` / `video` / `audio` / `speech` / `music` / `image describe`
6. task 轮询与输出兼容
7. `artifacts` / `models routes` / `media get`
8. `jobs` 兼容策略收口

## 9. 验收标准

达到以下条件后，可认为迁移主链路完成：

- 用户向生成命令不再调用 `/jobs`、`/models/infer`、`/video/generations`
- `auth`、上传、模型选择、task 创建、task 轮询全部走 `www.popi.art`
- 输出保留 `job_id` 兼容字段，但内部主键已切到 `task_id`
- 不支持保留的命令明确返回 `unsupported`
- 新增中文注释数量符合本计划限制
