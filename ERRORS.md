# PopiArt CLI Error Reference

本文件定义对外可依赖的错误 envelope、常见 `error.code`、退出码和重试语义。

## 响应结构

默认 JSON 模式下：

```json
{
  "ok": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "无效的命令参数"
  }
}
```

约定：

- `error.code`：机器可读、稳定
- `error.message`：面向用户的短描述
- 其他字段：上下文、提示、HTTP status、资源 ID、路径等

## 输出模式

- `--output json`：输出标准 JSON envelope
- `--output plain`：输出人类可读错误文本
- `--plain`：`--output plain` 的兼容别名

## Exit Codes

| Code | Meaning |
|---|---|
| `0` | 成功 |
| `1` | 通用错误 / 未分类错误 |
| `2` | 参数、输入、使用方式错误 |
| `3` | 认证或鉴权错误 |
| `4` | 限流 / 配额错误 |
| `5` | 轮询超时 |
| `6` | 网络或服务端可恢复错误 |
| `7` | 远端 job / 更新流程执行失败 |
| `10` | 内容策略拦截 |

## Retry 语义

### 通常不要直接重试

- `VALIDATION_ERROR`
- `BAD_REQUEST`
- `INPUT_ERROR`
- `INPUT_NOT_FOUND`
- `INPUT_PARSE_ERROR`
- `LOCAL_SKILL_INVALID`
- `UNAUTHENTICATED`
- `FORBIDDEN`
- `NOT_FOUND`
- `NO_PROJECT`
- `LOCAL_SKILL_UNSUPPORTED`
- `LOCAL_ONLY_SKILL`
- `UNSUPPORTED_INSTALL`

这类错误通常说明参数、输入、权限或安装方式有问题，应该先修正上下文。

### 可以在回退或退避后重试

- `RATE_LIMITED`
- `NETWORK_ERROR`
- `SERVICE_UNAVAILABLE`
- `SERVER_ERROR`
- `HTTP_ERROR`
- `POLL_TIMEOUT`

建议：

- `RATE_LIMITED`：指数退避后再试
- `NETWORK_ERROR` / `SERVICE_UNAVAILABLE`：短暂重试
- `POLL_TIMEOUT`：延长 `jobs wait --interval` 或改为 `jobs get`

### 需要人工检查或业务回退

- `JOB_FAILED`
- `UPDATE_FAILED`
- `RUNTIME_SKILL_PLACEHOLDER`

这类错误通常说明远端执行链路或当前能力面还没有准备好。

## 常见错误代码

| Code | Meaning | Typical Fix |
|---|---|---|
| `VALIDATION_ERROR` | 参数不合法、缺少必填参数、flag 冲突 | 修正 CLI 参数或输入 JSON |
| `BAD_REQUEST` | API endpoint、API path 或请求体构造错误 | 检查 endpoint、URL、请求结构 |
| `INPUT_ERROR` | stdin 或本地输入读取失败 | 检查 stdin、文件权限、文件内容 |
| `INPUT_NOT_FOUND` | 输入文件不存在 | 检查路径 |
| `INPUT_PARSE_ERROR` | JSON 或 metadata 非法 | 修正 JSON |
| `UNAUTHENTICATED` | 缺少 key 或 key 无效 | `popiart auth login --key <product-key>` |
| `FORBIDDEN` | 当前 key 没有访问权限 | 更换 key 或检查项目权限 |
| `NOT_FOUND` | 资源不存在 | 检查 skill / job / artifact / media / project ID |
| `CONFLICT` | 资源冲突，例如重复安装本地 skill | 改名、清理冲突，或按提示覆盖 |
| `RATE_LIMITED` | 请求过快或额度受限 | 退避重试 |
| `NETWORK_ERROR` | 网络失败、流式写入失败、下载失败 | 检查网络，必要时重试 |
| `SERVICE_UNAVAILABLE` | 服务暂不可用 | 稍后重试 |
| `SERVER_ERROR` | 服务端内部错误 | 稍后重试或查看服务状态 |
| `POLL_TIMEOUT` | `jobs wait` 在超时内没有结束 | 延长等待、改轮询策略 |
| `JOB_FAILED` | 远端 job 执行失败 | 查看 `jobs logs`、检查输入和路由 |
| `RUNTIME_SKILL_PLACEHOLDER` | 官方 runtime skill 还是占位状态 | 改用可用 runtime，或等待服务端注册 |
| `NO_PROJECT` | 当前没有活动项目 | `popiart project use <id>` |
| `LOCAL_SKILL_INVALID` | 本地 skill 包或 manifest 非法 | 修正 skill 包结构 |
| `LOCAL_SKILL_UNSUPPORTED` | 本地 skill 当前执行模式不受支持 | 调整为 `remote-runtime` 或更换 runner |
| `LOCAL_ONLY_SKILL` | 当前 skill 只是 bundled seed helper | 改用对应远端 runtime skill |
| `UNSUPPORTED_INSTALL` | 当前安装方式不支持自更新 | 按提示改用 Homebrew 或重新安装 |
| `UPDATE_FAILED` | 更新链路执行失败 | 查看错误详情，重试或手动更新 |

## 常见命令场景

### `popiart auth login`

| Scenario | Error Code | Notes |
|---|---|---|
| `--non-interactive` 下缺少 `--key` | `VALIDATION_ERROR` | agent / CI 应显式传 `--key` |
| 登录请求失败 | `NETWORK_ERROR` / `UNAUTHENTICATED` / `FORBIDDEN` | 检查 endpoint 与 key |

### `popiart image generate`

| Scenario | Error Code | Notes |
|---|---|---|
| 缺少 `--prompt` | `VALIDATION_ERROR` | façade 命令会直接拒绝 |
| 远端 job 创建失败 | `NETWORK_ERROR` / `RATE_LIMITED` / `SERVER_ERROR` | 按重试语义处理 |

### `popiart video generate`

| Scenario | Error Code | Notes |
|---|---|---|
| 同时传 `--image` 和 `--source-artifact-id` | `VALIDATION_ERROR` | 两者二选一 |
| 本地源图不存在 | `CLI_ERROR` | 检查路径 |
| 自动上传后缺少 `artifact_id` | `CLI_ERROR` | 属于异常返回，建议查看服务端 |
| 官方 runtime 仍是占位 | `RUNTIME_SKILL_PLACEHOLDER` 或 direct fallback | CLI 会尽量桥接官方 fallback |

### `popiart audio tts`

| Scenario | Error Code | Notes |
|---|---|---|
| `--text` 与 `--text-file` 同时出现 | `VALIDATION_ERROR` | 两者二选一 |
| 读取 `--text-file` 失败 | `CLI_ERROR` | 检查文件路径和权限 |

### `popiart run`

| Scenario | Error Code | Notes |
|---|---|---|
| 输入 JSON 非法 | `INPUT_PARSE_ERROR` | 修正 `--input` |
| 本地 helper skill 不能直接运行 | `LOCAL_ONLY_SKILL` | 改用远端 runtime skill |
| 本地 skill 执行模式不支持 | `LOCAL_SKILL_UNSUPPORTED` | 当前仅支持 `remote-runtime` |

### `popiart artifacts upload` / `media upload`

| Scenario | Error Code | Notes |
|---|---|---|
| 本地文件不存在 | `CLI_ERROR` | 检查路径 |
| `metadata-json` 非法 | `INPUT_PARSE_ERROR` | 修正 JSON |
| 上传失败 | `NETWORK_ERROR` / `RATE_LIMITED` / `SERVER_ERROR` | 可按重试语义处理 |

## 推荐的 agent 调用模式

在 agent / CI 里，推荐统一：

```sh
popiart ... --output json --quiet --non-interactive
```

写操作前可以先：

```sh
popiart ... --dry-run --output json --quiet --non-interactive
```

这样更容易做自动重试、错误分类和日志收敛。
