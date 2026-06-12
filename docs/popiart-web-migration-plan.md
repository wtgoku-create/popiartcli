# PopiArt CLI 迁移到 `www.popi.art` 接口方案

Date: `2026-06-10`

本文档用于记录 `popiartcli` 从当前旧后端接口体系迁移到 `https://www.popi.art` 主站接口体系的设计方案。

目标：

- 不改变现有 CLI 命令的使用方式
- 底层执行链路 100% 切换到 `popi.art`
- 用主站的统一 task API 承接图片、视频、音频、LLM 任务
- 把音乐能力并入音频能力

非目标：

- 本文档不直接修改代码
- 本文档不覆盖 MCP / bootstrap / discoverability 的最终保留策略
- 本文档不定义最终的 UI 或网站侧产品规则

## 1. 总体原则

本次迁移遵循以下原则：

1. 命令使用方式尽量不变
2. 命令语义允许调整
3. 执行链路不再调用旧后端接口
4. 图片、视频、音频、LLM 统一走 `popi.art` task API
5. 本地媒体统一先上传到 `popi.art`，再用返回 URL 创建任务
6. 登录后统一使用 `Authorization: Bearer <token>` 调用接口

补充解释：

- “命令使用方式不变” 的含义是：
  - 命令名不变
  - 子命令不变
  - 现有常用 flags 名称不变
  - 用户输入习惯不变
- 允许变化的是：
  - flags 背后的语义解释
  - 默认模型
  - 底层请求体结构
  - 底层请求目标接口

## 2. 新旧接口边界

### 2.1 迁移后保留的主站接口

核心接口：

- `GET /api_client/users/user/info`
- `POST /api_client/media/upload`
- `GET /api_client/anime/ai/model/list`
- `POST /api_client/anime/task/create`
- `GET /api_client/anime/task/list`
- `GET /api_client/anime/task/detail`
- `GET /api_client/anime/task/downloadUrls`

可选接口：

- `POST /api_client/anime/task/expandPrompt`
- `POST /api_client/anime/task/calculateTaskPrice`
- `POST /api_client/anime/task/publish`
- `GET/POST /api_client/auth/logout`

### 2.2 迁移后不再调用的旧后端接口

执行链路中不再调用：

- `/auth/login`
- `/auth/me`
- `/auth/logout`
- `/auth/token/rotate`
- `/jobs`
- `/models`
- `/models/infer`
- `/models/routes`
- `/models/routes/overrides`
- `/skills`
- `/artifacts/upload`
- `/media/upload`（旧后端版本）
- `/projects`
- `/budget`

说明：

- 这些旧接口可以继续存在于代码仓库中，作为历史实现
- 但迁移完成后的主执行链路不再依赖它们

## 3. 当前 CLI 命令清单与作用

下表先列出当前 CLI 命令

### 3.1 用户向命令

| 命令 | 作用 |
| --- | --- |
| `auth login` | 登录并保存认证信息 |
| `auth logout` | 退出登录 |
| `auth whoami` | 查看当前身份 |
| `auth key show/set/rotate` | 查看或管理本地 key/token |
| `image generate` | 文生图 |
| `image img2img` | 图生图 |
| `image transform` | 图生图别名入口 |
| `image describe` | 图片理解 / 识图描述 |
| `video generate` | 通用视频生成 |
| `video img2video` | 图生视频 |
| `video from-image` | 图生视频别名入口 |
| `video action-transfer` | 动作迁移 |
| `video seedance` | Seedance / 豆包视频 |
| `audio tts` | 文本生语音 |
| `speech synthesize` | 文本生语音别名入口 |
| `music generate` | 音乐生成 |

### 3.2 平台向命令

| 命令 | 作用 |
| --- | --- |
| `skills list/get/schema` | skill 发现与 schema 查询 |
| `skills pull/install/use-local` | 本地 skill 安装和切换 |
| `run` | 直接执行 skill |
| `jobs get/wait/list/cancel/logs` | 作业查询与跟踪 |
| `artifacts list/get/pull/pull-all/upload` | 工件查询、下载、上传 |
| `media get/upload` | 稳定媒体查询与上传 |
| `models list/routes/infer/route-override` | 模型与路由能力 |
| `project current/use/list/get/context` | 项目上下文管理 |
| `budget status/usage/limits` | 预算和用量查询 |

### 3.3 工程与集成命令

| 命令 | 作用 |
| --- | --- |
| `mcp serve/print-config/doctor` | MCP 能力 |
| `bootstrap` | discoverability 资产初始化 |
| `setup` | 一键初始化 |
| `completion` | shell 补全 |
| `export-schema` | 导出 schema |
| `update` | CLI 自更新 |

## 4. 已确定决议

### 4.1 登录与鉴权

已确定：

- `auth login --key <token>`
- `--key` 的实际语义改为 `popi.art user token`
- 登录验证接口改为：
  - `GET /api_client/users/user/info`
- 登录请求头：
  - `Authorization: Bearer <token>`
- 如果能成功拿到用户信息，则视为 token 有效
- 有效后本地保存 token

已确定：

- `auth whoami` 映射到 `GET /api_client/users/user/info`
- `auth logout` 只清本地 token

决议态：

- `auth key show` 继续保留，显示本地保存的 token
- `auth key set <token>` 保留，但语义改为直接保存 `popi.art token`
- `auth key rotate` 保留命令，但直接返回明确错误：
  - `UNSUPPORTED_IN_POPI_ART_MODE`
  - 并提示改用 `auth login --key <token>` 或 `auth key set <token>`

### 4.2 默认 endpoint

已确定：

- 默认 endpoint 改为 `https://www.popi.art`

本地配置保留：

- `endpoint`
- `token`

当前 `project` 字段先不作为主执行链路依赖。

### 4.3 模型参数

已确定：

- CLI 的 `--model` 参数后续优先解释为 `aiModelCode`
- 创建任务时必须解析成 `aiModelId`

已确定策略：

1. 优先使用用户显式传入的 `--model`
2. 通过 `GET /api_client/anime/ai/model/list` 查模型列表
3. 用 `aiModelCode` 匹配模型，得到 `aiModelId`
4. 如果用户未传 `--model`，则降级到本地预设默认模型配置

决议态：

- 默认模型配置写入 `config.go` 或独立默认模型映射模块
- 默认配置项优先保存 `aiModelCode`
- 运行时再解析为 `aiModelId`
- 解析模型时不能只取 `aiModelId`
- 还必须同时读取模型能力字段，用于决定：
  - 该模型是否支持当前任务类型
  - 该模型是否支持上传图片 / 视频 / 音频
  - 该模型是否支持当前 `subType`
  - 该模型支持哪些比例 / 分辨率 / 时长
  - 该模型允许多少张参考图
- `categories[].taskSubType` 作为模型是否可用于当前任务的硬约束

决议态：

1. 先由 CLI 命令解析出目标 `type/subType`
2. 再按 `--model` 或默认配置找到候选模型
3. 用 `categories[].taskSubType` 做硬过滤
4. 过滤通过后再校验：
   - `ratio` / `videoRatio`
   - `resolution`
   - `duration`
   - `uploadImageLimit`
5. 通过后才允许生成最终请求体

显式 `--model` 策略：

- 如果用户显式传了 `--model`
- 但该模型的 `categories[].taskSubType` 不包含当前目标 `subType`
- 直接报错：
  - `MODEL_SUBTYPE_UNSUPPORTED` 或 `VALIDATION_ERROR`
- 不自动偷偷切换到别的模型

默认模型策略：

- 如果用户未传 `--model`
- 先按当前命令查默认 `aiModelCode`
- 再用 `categories[].taskSubType` 过滤
- 若默认模型不支持当前 `subType`
  - 继续在同类型默认候选池里找支持该 `subType` 的模型
- 若仍无可用模型
  - 报 `MODEL_NOT_FOUND`

补充说明：

- `isSupportImages` / `isSupportVideos` / `isSupportAudios` 只能说明“能否接收这类输入媒资”
- 不能替代 `categories[].taskSubType`
- 真正决定“该模型能否执行这个任务”的，应以 `categories[].taskSubType` 为准

### 4.4 媒体上传

已确定：

- 一律使用 `POST /api_client/media/upload`
- 单文件上传字段名统一使用 `file`
- 多张图 / 多个文件逐个上传
- 成功后从响应中读取稳定字段 `url`

已确定重试策略：

- 上传失败最多重试 3 次
- 超过重试次数后报错

决议态：

- 仅对网络错误和 5xx 重试
- 4xx 直接失败，不重试

### 4.5 任务式执行

已确定：

- 图片、视频、音频、LLM 统一走 `POST /api_client/anime/task/create`
- `image describe` 为保持任务式体验，也走 `task/create`
- 音乐并入音频能力体系

### 4.6 图片映射

已确定：

- `image generate` -> `type=1`, `subType=102`
- `image img2img` -> `type=1`, `subType=103`
- `image transform` -> `type=1`, `subType=103`
- 多参考图最终都放进 `images[]`
- 多图顺序按用户输入顺序保持不变
- 第一张图即 `images[0]`
- 图片命令的图片输入最多允许传 5 张

### 4.7 视频映射

已确定大方向：

- 按输入条件区分：
  - `202` 多图生视频
  - `203` 参考生视频
  - `204` 首尾帧视频
  - `205` 动作模仿

决议态：

- 保持原来的“按命令入口直接按子命令分实现”
- 但在每个视频入口内部，再按输入形态映射最终 `subType`
- `video seedance` 继续保留为“走 Seedance 模型语义的专属入口”

决议态：

第一层：按命令入口分执行器

- `video action-transfer`
  - 固定走动作模仿执行链
  - 固定映射 `subType=205`
  - 最多允许 2 张图片和 1 个视频
- `video seedance`
  - 固定走 Seedance 执行链
  - 优先选 Seedance 对应的默认 `aiModelCode`
  - 在该链内部再细分 `subType`
- `video generate` / `video img2video` / `video from-image`
  - 走通用视频任务执行链
  - 优先选通用视频默认模型
  - 新增 `--action`
  - `--image` 改为可重复传入
  - `--image` 最多允许传 5 张
  - 在该链内部再细分 `subType`

第二层：按 `action` 分 `subType`

- `video action-transfer`
  - 固定 `subType=205`
  - 最多允许 2 张图片和 1 个视频
- `video seedance`
  - 若显式传入 `--action`
    - `firstTailGenerate` -> `subType=204`
    - `referenceGenerate` -> `subType=203`
    - `generate` -> `subType=202`
    - `textGenerate`
      - 若迁移一期没有稳定承接模型
      - 直接返回 `CAPABILITY_UNAVAILABLE`
  - 若未显式传入 `--action`
    - 有 `--video` 或 `--audio` -> `subType=203`
    - 仅图片且图片数量 `>= 2` -> `subType=202`
    - 单张图片 -> `subType=203`
- `video generate` / `video img2video` / `video from-image`
  - `--action referenceGenerate` -> 显式传 `subType=203`
  - `--action generate` -> 显式传 `subType=202`
  - `--action firstTailGenerate` -> 显式传 `subType=204`
- 纯 prompt 视频
  - 若迁移一期没有稳定承接模型
  - 直接返回 `CAPABILITY_UNAVAILABLE`

说明：

- 也就是说，可以保留原来的“按命令入口直接按子命令分实现”
- 但不能只靠子命令名直接决定所有视频 `subType`
- 对 `video generate` / `img2video` / `from-image` 而言，`subType` 不再由图片数量判断，而是由 `--action` 直接决定
- 因为新后端的“参考生视频”和“多图生视频”都支持传多图
- `--image` 改为可重复传入后，通用视频命令即可表达参考生视频 / 多图生视频 / 首尾帧视频
- 图片数量不负责决定 `subType`，只负责做输入合法性校验
- 例如：
  - `subType=203` 至少需要 1 张参考图
  - `subType=202` 需要多张图片输入
  - `subType=204` 需要恰好 2 张图，并按输入顺序解释为首帧、尾帧
  - 通用视频命令的 `--image` 最多允许传 5 张

### 4.8 音频与音乐映射

已确定：

- `audio tts` -> `type=3`, `subType=301`
- `speech synthesize` -> `type=3`, `subType=301`
- `music generate` -> `type=3`, `subType=301`

即：

- 音乐被纳入音频能力
- 统一使用文本生音频任务路径
- 歌词、风格、乐器、节奏等附加信息写入 `metadata`

### 4.9 `--wait` 行为

已确定：

- `--wait` 后续统一轮询：
  - `GET /api_client/anime/task/detail?id=...`

当前旧逻辑说明：

- 旧实现是先拿到 `job_id`
- 然后轮询 `/jobs/{jobId}`
- 终态是 `done` / `failed` / `cancelled`
- 失败映射为 `JOB_FAILED`
- 超时映射为 `POLL_TIMEOUT`

迁移后：

- 底层改为 task 轮询
- CLI 表面仍可保持“等待任务完成”的使用方式不变

### 4.10 输出风格

已确定：

- CLI 表面输出风格尽量保持现有风格
- 对用户仍然尽量展示 `job_id`

内部语义：

- 底层主键已经切换为 `task_id`
- 输出时将 `task_id` 映射为表层 `job_id`

决议：

- 输出中同时保留：
  - `job_id`（兼容字段）
  - `task_id`（真实字段）
- 成功结果统一补充：
  - `status`
  - `download_urls`
  - `result_urls`
  - `model`
  - `type`
  - `sub_type`

## 5. `popi.art` 接口映射表

### 5.1 认证与用户

| CLI 命令 | `popi.art` 接口 | 说明 |
| --- | --- | --- |
| `auth login --key <token>` | `GET /api_client/users/user/info` | 用 Bearer token 验证有效性 |
| `auth whoami` | `GET /api_client/users/user/info` | 查询当前用户信息 |
| `auth logout` | 无远端强依赖 | 只清本地 token |

请求头：

```http
Authorization: Bearer <token>
```

### 5.2 媒体上传

| 场景 | `popi.art` 接口 | 说明 |
| --- | --- | --- |
| 本地图片上传 | `POST /api_client/media/upload` | 取响应里的 `url` 写入 `images[]` |
| 本地视频上传 | `POST /api_client/media/upload` | 取响应里的 `url` 写入 `videos[]` |
| 本地音频上传 | `POST /api_client/media/upload` | 取响应里的 `url` 写入 `voices[]` |

上传策略：

- 单文件字段名：`file`
- 多文件：逐个上传
- 上传失败最多重试 3 次

### 5.3 模型查询

| 目的 | `popi.art` 接口 | 说明 |
| --- | --- | --- |
| 查询模型列表 | `GET /api_client/anime/ai/model/list` | 用于 `aiModelCode -> aiModelId` 解析 |

用途：

- 处理用户显式 `--model`
- 解析默认模型配置
- 兼容环境间 `aiModelId` 不稳定的问题

### 5.4 `model/list` 能力字段清单

`GET /api_client/anime/ai/model/list` 不只是“模型字典”，它本身还是迁移后的能力源。

至少消费以下字段：

| 字段 | 含义 | 迁移用途 |
| --- | --- | --- |
| `id` | 模型 ID | 写入 `aiModelId` |
| `code` | 模型 code | 与 CLI `--model` 对齐 |
| `aiModelCodeAlias` | 模型别名 | 兼容旧默认模型名或外部别名 |
| `name` | 展示名 | 输出与报错提示 |
| `ratio` | 图片比例候选 | 校验 / 归一化 `--aspect-ratio` |
| `videoRatio` | 视频比例候选 | 校验视频比例 |
| `resolution` | 分辨率候选 | 校验 / 归一化 `--size` |
| `duration` | 时长候选 | 校验 `--duration` |
| `displayDimensions` | 维度展示配置 JSON | 作为 UI / CLI 帮助和候选集补充来源 |
| `billingDimensions` | 计费维度映射 JSON | 必要时把 CLI 字段翻译成站点计费维度 |
| `isSupportImages` | 是否支持图片输入 | 控制 `images[]` |
| `isSupportVideos` | 是否支持视频输入 | 控制 `videos[]` |
| `isSupportAudios` | 是否支持音频输入 | 控制 `voices[]` |
| `uploadImageLimit` | 图片上传上限 | 校验多参考图数量 |
| `categories[].taskSubType` | 模型支持的任务子类型 | 判断模型是否可用于当前命令 |
| `providers` | 供应商信息 | 迁移一期通常只读不透出 |
| `billingBindings` | 计费绑定 | 后续若要做价格预估会用到 |

补充说明：

- 文档之前只写了 `aiModelCode -> aiModelId`，这个粒度不够。
- 迁移后真正需要的是：
  - `aiModelCode -> 模型对象`
  - 再由模型对象提取 `aiModelId + 能力约束 + 可选维度`

### 5.5 迁移后的模型能力感知流程

迁移后统一执行以下流程：

1. 先根据命令确定目标 `type/subType`
2. 读取本次命令的原始 CLI flags
3. 解析 `--model` 或默认模型配置，得到候选 `aiModelCode`
4. 调 `GET /api_client/anime/ai/model/list`
5. 用 `code` 或 `aiModelCodeAlias` 匹配模型
6. 校验该模型是否支持当前 `subType`
7. 校验输入媒资类型是否被支持：
   - `images[]`
   - `videos[]`
   - `voices[]`
8. 校验维度参数是否被支持：
   - `ratio` / `videoRatio`
   - `resolution`
   - `duration`
9. 再把 CLI 通用参数翻译到 `task/create`
10. 对不支持的字段做“明确报错”或“安全降级”

规则：

- `--size` 不再盲目原样透传，应优先映射到模型支持的 `resolution`
- `--aspect-ratio` 不再盲目原样透传，应优先匹配：
  - 图片任务用 `ratio`
  - 视频任务用 `videoRatio`
- `--duration` 只在模型 `duration` 列表支持时透传
- 多参考图数量不得超过 `uploadImageLimit`
- 图片命令的图片输入上限为 5 张
- 通用视频命令的图片输入上限为 5 张
- 当前命令若要求参考视频 / 参考音频，而模型不支持 `isSupportVideos` / `isSupportAudios`，应直接报 `VALIDATION_ERROR`
- 当前命令若目标 `subType` 不在模型 `categories[].taskSubType` 内，应直接报 `MODEL_SUBTYPE_UNSUPPORTED`

### 5.6 原本 CLI 是怎么处理模型差异的

现有 CLI 的主逻辑并不是“先查模型能力，再按模型裁剪字段”，而是：

1. Cobra 负责解析命令与 flags
2. 命令层把用户输入规整成一份通用 payload
3. 默认直接提交到旧后端：
   - `/jobs`
   - `/models/infer`
   - `/video/generations`
4. 由旧后端 / 具体 runtime 承担大部分模型差异吸收工作

现有代码表现为：

- 图片命令只做通用字段归一化，例如 `prompt`、`size`、`aspect_ratio`、`style`、`seed`
- 视频命令只做通用字段归一化，例如 `duration_s`、`aspect_ratio`、`camera_motion`
- `image describe` 要求用户显式传 `--model`，但并不会先查模型能力表
- 少数命令包含特定模型的定制校验，但不是统一的模型能力框架

已确认的现状示例：

- 文生图只把 `--size`、`--aspect-ratio` 直接写入通用 payload，没有基于模型列表做候选校验
- 图生视频会把 `duration` 归一化到 `duration_s`
- 旧官方 runtime 里只对少量模型差异做特殊分支，例如 image2video 时长不是 `5` 或 `10` 秒时切到 fallback 模型
- 音乐命令对 `music-cover` 与 `music-2.6` 做了手写条件校验，但这属于命令内规则，不是统一模型中心规则

结论：

- 原始 CLI 更像“通用参数归一化层”
- 原始后端更像“模型路由与模型差异吸收层”
- 迁移到 `popi.art` 后，这部分责任要前移到 CLI
- 所以迁移文档里必须把 `model/list` 能力感知写成正式前置步骤，而不能只写字段映射

## 6. CLI 命令到 `task/create` 参数映射

统一任务接口：

```http
POST /api_client/anime/task/create
Content-Type: application/json
Authorization: Bearer <token>
```

### 6.1 通用映射规则

| CLI 语义 | `popi.art` 字段 | 说明 |
| --- | --- | --- |
| 主任务类型 | `type` | 图片=1，视频=2，音频=3，LLM=5 |
| 子任务类型 | `subType` | 见具体命令映射 |
| 模型 code | `aiModelCode` | 由 `--model` 或默认配置提供 |
| 模型 ID | `aiModelId` | 通过模型列表解析得到 |
| 提示词 | `chatPrompt` | 主提示词统一映射到这里 |
| 图片输入 | `images[]` | 本地图先上传，传 `url` |
| 视频输入 | `videos[]` | 本地图先上传，传 `url` |
| 音频输入 | `voices[]` | 本地图先上传，传 `url` |
| 画面比例 | `aspectRatio` + `ratio` | 尽量两者都写 |
| 分辨率 | `resolution` | 由 `--size` 映射，但必须受模型 `resolution[]` 约束 |
| 时长 | `duration` | 主要用于视频，必须受模型 `duration[]` 约束 |
| 音色 | `voiceId` | 主要用于音频 / 语音 |
| 额外参数 | `metadata` | 统一透传高级参数 |

重要补充：

- 字段映射不是“把 CLI flags 全塞进请求体”这么简单。
- 迁移后要先看模型能力，再决定哪些字段能传。
- 也就是说，`resolution`、`ratio`、`duration` 这些字段都属于“条件透传字段”。

### 6.2 图片命令映射

#### `image generate`

| CLI 字段 | `task/create` 字段 |
| --- | --- |
| `type` | `1` |
| `subType` | `102` |
| `--model` | `aiModelCode` |
| 解析模型结果 | `aiModelId` |
| `prompt` | `chatPrompt` |
| `--aspect-ratio` | `aspectRatio` + `ratio` |
| `--size` | `resolution` |
| `--seed` | `metadata.seed` |
| `--style` | `metadata.style` |
| `--negative-prompt` | `metadata.negative_prompt` |
| `--notes` | `metadata.notes` |

#### `image img2img`

| CLI 字段 | `task/create` 字段 |
| --- | --- |
| `type` | `1` |
| `subType` | `103` |
| `--model` | `aiModelCode` |
| 解析模型结果 | `aiModelId` |
| `prompt` | `chatPrompt` |
| `--image` | 上传后写入 `images[0]` |
| `--identity-reference-image` | 继续追加到 `images[]` |
| `--style-reference-image` | 继续追加到 `images[]` |
| `--reference-image` | 继续追加到 `images[]` |
| `--aspect-ratio` | `aspectRatio` + `ratio` |
| `--size` | `resolution` |
| `--strength` | `metadata.strength` |
| `--preserve-composition` | `metadata.preserve_composition` |
| `--style` | `metadata.style` |
| `--negative-prompt` | `metadata.negative_prompt` |
| `--seed` | `metadata.seed` |
| `--notes` | `metadata.notes` |

#### `image transform`

- 与 `image img2img` 相同

#### `image describe`

| CLI 字段 | `task/create` 字段 |
| --- | --- |
| `type` | `5` |
| `subType` | `501` |
| `--model` | `aiModelCode` |
| 解析模型结果 | `aiModelId` |
| 图片输入 | `images[]` |
| 描述提示 | `chatPrompt` |
| `--notes` | `metadata.notes` |

说明：

- 为保持任务式体验，`image describe` 走 `task/create`
- `--model` 的使用方式不变，但语义从“旧多模态模型 ID”切到“`popi.art` 的 `aiModelCode`”

### 6.3 视频命令映射

#### `video generate`

| CLI 字段 | `task/create` 字段 |
| --- | --- |
| `type` | `2` |
| `subType` | 按输入条件确定 |
| `--model` | `aiModelCode` |
| 解析模型结果 | `aiModelId` |
| `prompt` | `chatPrompt` |
| `--image` / `--from` | 上传后写入 `images[]` |
| `--aspect-ratio` | `aspectRatio` + `ratio` |
| `--duration` | `duration` |
| `--camera-motion` | `metadata.camera_motion` |
| `--motion-intensity` | `metadata.movement_amplitude` |
| `--negative-prompt` | `metadata.negative_prompt` |
| `--style` | `metadata.style` |
| `--seed` | `metadata.seed` |
| `--notes` | `metadata.notes` |

#### `video img2video`

- 与 `video generate` 相同

#### `video from-image`

- 与 `video generate` 相同

#### `video action-transfer`

| CLI 字段 | `task/create` 字段 |
| --- | --- |
| `type` | `2` |
| `subType` | `205` |
| `--model` | `aiModelCode` |
| 解析模型结果 | `aiModelId` |
| `--image` | 上传后写入 `images[]` |
| `--video` | 上传后写入 `videos[]` |
| `prompt` | `chatPrompt` |
| `--action` | `metadata.action` |
| `--cut-result-first-second-switch` | `metadata.cut_result_first_second_switch` |
| `--notes` | `metadata.notes` |

约束：

- 最多允许 2 张图片
- 最多允许 1 个视频

#### `video seedance`

| CLI 字段 | `task/create` 字段 |
| --- | --- |
| `type` | `2` |
| `subType` | 按输入条件确定 |
| `--model` | `aiModelCode` |
| 解析模型结果 | `aiModelId` |
| `prompt` | `chatPrompt` |
| `--image` | 上传后写入 `images[]` |
| `--video` | 上传后写入 `videos[]` |
| `--audio` | 上传后写入 `voices[]` |
| `--duration` | `duration` |
| `--ratio` | `aspectRatio` + `ratio` |
| `--size` | `resolution` |
| `--seed` | `metadata.seed` |
| `--return-last-frame` | `metadata.return_last_frame` |
| `--generate-audio` | `metadata.generate_audio` |
| `--service-tier` | `metadata.service_tier` |
| `--execution-expires-after` | `metadata.execution_expires_after` |
| `--draft` | `metadata.draft` |
| `--tools-json` | `metadata.tools` |
| `--safety-identifier` | `metadata.safety_identifier` |
| `--notes` | `metadata.notes` |

### 6.4 音频与音乐命令映射

#### `audio tts`

| CLI 字段 | `task/create` 字段 |
| --- | --- |
| `type` | `3` |
| `subType` | `301` |
| `--model` | `aiModelCode` |
| 解析模型结果 | `aiModelId` |
| `text` | `chatPrompt` |
| `--voice` | `voiceId` |
| `--language` | `metadata.language` |
| `--voice-style` | `metadata.voice_style` |
| `--speed` | `metadata.speed` |
| `--volume` | `metadata.volume` |
| `--pitch` | `metadata.pitch` |
| `--emotion` | `metadata.emotion` |
| `--format` | `metadata.format` |
| `--sample-rate-hz` | `metadata.sample_rate_hz` |
| `--bitrate` | `metadata.bitrate` |
| `--channels` | `metadata.channels` |
| `--subtitles` | `metadata.subtitles` |
| `--pronunciation` | `metadata.pronunciation` |
| `--sound-effect` | `metadata.sound_effect` |
| `--seed` | `metadata.seed` |
| `--notes` | `metadata.notes` |

#### `speech synthesize`

- 与 `audio tts` 相同

#### `music generate`

| CLI 字段 | `task/create` 字段 |
| --- | --- |
| `type` | `3` |
| `subType` | `301` |
| `--model` | `aiModelCode` |
| 解析模型结果 | `aiModelId` |
| `prompt` + `lyrics` | `chatPrompt` |
| `--lyrics-optimizer` | `metadata.lyrics_optimizer` |
| `--instrumental` | `metadata.instrumental` |
| `--vocals` | `metadata.vocals` |
| `--genre` | `metadata.genre` |
| `--mood` | `metadata.mood` |
| `--instruments` | `metadata.instruments` |
| `--tempo` | `metadata.tempo` |
| `--bpm` | `metadata.bpm` |
| `--key` | `metadata.key` |
| `--avoid` | `metadata.avoid` |
| `--use-case` | `metadata.use_case` |
| `--structure` | `metadata.structure` |
| `--references` | `metadata.references` |
| `--extra` | `metadata.extra` |
| `--aigc-watermark` | `metadata.aigc_watermark` |
| `--format` | `metadata.format` |
| `--sample-rate-hz` | `metadata.sample_rate_hz` |
| `--bitrate` | `metadata.bitrate` |
| `--audio-url` | 先上传，再写入 `voices[]` |
| `--audio-base64` | 先上传，再写入 `voices[]` |

## 7. 结果查询与 `--wait`

### 7.1 当前旧逻辑

当前 CLI 的 `--wait` 逻辑：

1. 提交任务
2. 返回 `job_id`
3. 如果设置 `--wait`
4. 轮询旧接口 `/jobs/{jobId}`
5. 直到进入终态：
   - `done`
   - `failed`
   - `cancelled`

失败时：

- 映射为 `JOB_FAILED`

超时时：

- 映射为 `POLL_TIMEOUT`

### 7.2 迁移后新逻辑

迁移后统一为：

1. 调 `POST /api_client/anime/task/create`
2. 取返回任务主键
3. 对外输出仍可保留 `job_id`
4. `--wait` 时轮询：
   - `GET /api_client/anime/task/detail?id=...`

状态映射：

| `popi.art` task 状态 | 含义 | CLI 行为 |
| --- | --- | --- |
| `0` | 排队中 | 继续轮询 |
| `1` | 生成中 | 继续轮询 |
| `2` | 成功 | 返回结果 |
| `-1` | 已取消 | 视为终态 |
| `-2` | 失败 | 返回 `JOB_FAILED` |

结论：

- `detail` 成功后，再补一次 `downloadUrls`
- 用于返回最终下载地址集合

## 8. `batchSize` 行为差异

这是迁移中的重点差异之一。

### 8.1 `popi.art` 当前行为

图片任务：

- `batchSize > 1`
- 仍只创建 1 条 task
- 该 task 内部生成多张结果

非图片任务（视频 / 音频 / LLM）：

- `batchSize > 1`
- 后端会拆成多条 task
- 每条 task 的 `batchSize = 1`
- 但 `task/create` 接口只返回第一条 task

### 8.2 对 CLI 的影响

影响包括：

- `video generate --batch-size 3`
  - 可能实际创建 3 条 task
  - 但 CLI 只拿到第一条 task
- `--wait`
  - 如果只跟踪第一条 task，不能代表整批都完成
- 输出结果数量
  - 也可能与用户直觉不完全一致

### 8.3 拍板结论

拍板为：

- 第一阶段不新增用户侧 `--batch-size`
- 内部统一按 `batchSize=1`
- 不做非图片任务的批量拆单兼容

原因：

- 当前现有 CLI 并没有给用户暴露统一的 `--batch-size` 主命令参数
- 迁移目标要求“命令使用方式不变”
- `popi.art` 对非图片任务的拆单返回不适合直接暴露给当前 CLI

## 9. 错误映射规则

迁移后需要补一层 `popi.art` 到 CLI 错误体系的映射。

如下映射：

| `popi.art` 场景 | CLI 错误码 | 说明 |
| --- | --- | --- |
| `Please login first` | `UNAUTHENTICATED` | token 无效或未登录 |
| `Your account has been logged out.` | `UNAUTHENTICATED` 或 `SESSION_EXPIRED` | 被踢下线 |
| `invalid user` | `UNAUTHENTICATED` | 用户态无效 |
| 参数错误，例如“请上传图片” | `VALIDATION_ERROR` | 用户输入不满足接口要求 |
| 媒体上传失败 | `UPLOAD_FAILED` 或 `NETWORK_ERROR` | 区分上传链路 |
| 模型匹配不到 | `MODEL_NOT_FOUND` | `aiModelCode` 无法解析 |
| task 状态为 `-2` | `JOB_FAILED` | 任务执行失败 |
| 轮询超时 | `POLL_TIMEOUT` | 与旧 CLI 保持一致 |

task 失败时的错误信息优先级：

1. `user_error_tip_msg`
2. `error_msg`
3. 默认兜底文案

## 10. `project` 字段说明

当前 CLI 中的 `project` 字段来自旧后端体系，作用是：

- 记录活动项目 ID
- 自动在旧请求里附带 `project_id`
- 涉及：
  - 任务
  - 模型推理
  - 媒体上传
  - 预算查询
  - 项目上下文

迁移到 `popi.art` task 体系后：

- 当前执行链路先不依赖 `project`
- 本地配置字段可以暂时保留
- 但第一阶段不作为主功能输入

## 11. 命令兼容性目标

本次迁移的核心兼容目标是：

- 不改变现有 CLI 命令的使用方式

即：

- 命令名不变
- 子命令不变
- 常见 flags 尽量不变
- 用户调用习惯尽量不变

允许变化的部分：

- 登录语义变化
  - `--key` 现在表示 `popi.art token`
- 执行语义变化
  - 由 job / infer 模式改为 task 模式
- 输出底层主键变化
  - 内部是真正的 `task_id`
  - 对外可兼容展示为 `job_id`

## 12. 兼容保留但重做语义的命令

以下命令可以考虑“命令名继续保留”，文档结尾有决定

### 12.1 兼容命令映射表

| 命令 | 旧语义 | 迁移后语义 | 底层实现 |
| --- | --- | --- | --- |
| `artifacts list <task-id>` | 列出 job 生成的 artifact 列表 | 列出 task 结果文件列表 | `task/detail` + `task/downloadUrls` |
| `artifacts get <artifact-id>` | 查询单个 artifact 元数据 | 不保留 | `unsupported` |
| `artifacts pull <artifact-id>` | 下载单个 artifact 内容 | 不保留 | `unsupported` |
| `artifacts pull-all <task-id>` | 下载 job 的全部 artifact | 下载 task 的全部结果文件 | 基于 `downloadUrls` 批量下载 |
| `media get <media-id>` | 查询稳定 media 对象 | 不保留 | `unsupported` |
| `models routes` | 查看 route key -> model id 生效路由表 | 查看当前 CLI 默认模型选择结果 | 本地默认模型映射 + `ai/model/list` 解析结果 |

### 12.2 设计原则

- 保留命令名是为了尽量不破坏现有 CLI 使用习惯
- 但不应伪造不存在的旧后端对象语义
- 若 `popi.art` 无法提供足够接近的能力，应明确 `unsupported`

### 12.3 `artifacts` 兼容策略说明

迁移后最适合兼容保留的是：

- `artifacts list <task-id>`
- `artifacts pull-all <task-id>`

因为它们本质上都是“围绕任务结果文件集合工作”。

相对地，以下命令直接不保留：

- `artifacts get <artifact-id>`
- `artifacts pull <artifact-id>`

原因：

- 它们原本依赖“单个 artifact 对象 ID”这一旧后端概念
- 迁移到 `popi.art` 后，任务结果更适合表达为 URL 集合
- 不应继续伪造稳定 `artifact_id`

### 12.4 `models routes` 兼容策略说明

旧后端里的 `models routes` 查询的是中心化路由系统：

- `route_key -> model_id`

迁移后这套系统不再存在，因此该命令若保留，只能改成查看：

- 当前 CLI 命令对应的默认 `aiModelCode`
- 通过 `ai/model/list` 解析得到的 `aiModelId`
- 当前模型是否支持目标 `subType`

也就是说，它保留的是“看当前模型选择结果”的用户意图，而不是旧路由系统本身。

输出字段：

- `command`
- `default_ai_model_code`
- `resolved_ai_model_id`
- `supported_sub_types`
- `selected_by`

## 13. 当前状态

截至本文档当前版本，迁移方案的主设计决策已基本拍板完成。

当前已明确：

- `media get` 迁移后不保留
- 直接返回 `unsupported`

剩余事项主要属于实现细节，而非方案级待定问题。
   - 或直接降级为仅本地语义

## 13. 建议实现顺序

建议按以下顺序实施：

1. 改默认 endpoint 与 token 语义
2. 改 `auth login` / `auth whoami` / `auth logout`
3. 接入 `media/upload`
4. 接入 `ai/model/list`
5. 实现 `aiModelCode -> aiModelId`
6. 改 `image` / `video` / `audio` / `speech` / `music` / `image describe`
7. 改 `--wait` 到 task 轮询
8. 补错误映射
9. 再处理平台向命令的保留与降级策略
