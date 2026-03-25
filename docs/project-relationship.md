# PopiArt 项目关系

这份文档用于说明 `popiartcli`、`popiartServer`、`PopiNewAPI` 三个项目之间的职责边界、调用链路和联调顺序。

同一份文档会分别放在三个仓库中，目的不是重复维护，而是让开发者在任何一个仓库里开始工作时，都能先看到完整上下文，避免把职责放错层。

## 三个项目分别做什么

| 项目 | 角色 | 应该负责 | 不应该负责 |
|---|---|---|---|
| `popiartcli` | 面向 coding agent 和创作者的统一 CLI 入口 | 登录、发现 skill、调用 skill、查看 jobs、拉取 artifacts、本地配置 | 不直接持有上游 provider key，不直接做模型路由，不直接做供应商计费 |
| `popiartServer` | PopiArt 产品后端 | 用户鉴权、项目权限、skill 注册表聚合、skill 执行、job 生命周期、artifact 管理、路由决策、计费归因 | 不把供应商细节暴露给 CLI，不把 skillhub 直接耦合到 CLI |
| `PopiNewAPI` | 模型网关和通道管理层 | 管理上游渠道和 key、代理模型请求、记录原始用量、提供模型层能力 | 不承载 PopiArt 的 skill 业务语义，不负责 CLI 交互，不负责产品级项目上下文 |

## 标准调用链路

```text
coding agent / creator
        ->
    popiartcli
        ->
   popiartServer
        ->
    PopiNewAPI
        ->
model providers
```

这条链路里，`popiartcli` 是入口层，`popiartServer` 是产品编排层，`PopiNewAPI` 是模型网关层。

## skillhub 放在哪一层

`skillhub` 是公开 skill 注册表，不是模型执行层。

推荐关系是：

```text
GitHub skillhub / skillhub.popi.art
              ->
        popiartServer 同步或聚合
              ->
          /skills API
              ->
           popiartcli
```

这样做有三个好处：

1. CLI 不需要直接依赖 GitHub 或站点结构。
2. skill 搜索、过滤、版本兼容可以统一放在后端处理。
3. 以后从 GitHub 仓库切到 `skillhub.popi.art` 时，不需要改 CLI。

## 授权、路由、计费分别在哪一层

- 用户登录和项目权限：`popiartServer`
- CLI 本地 token 持久化：`popiartcli`
- 模型路由选择：`popiartServer`
- 上游 provider key 管理：`PopiNewAPI`
- 原始模型调用计量：`PopiNewAPI`
- 面向 skill / project / user 的计费归因：`popiartServer`

一个重要原则是：

**真实 provider key 不进入 `popiartcli`。**

CLI 只拿产品层 token；后端再用自己的方式调用 `PopiNewAPI`。

## 当前测试应该怎么分阶段

### 第一阶段：只验证产品协议

先验证：

- `auth`
- `skills`
- `run`
- `jobs`
- `artifacts`

这一阶段可以完全不接真实模型，也不需要真实 provider key。

### 第二阶段：接一个真实模型

先只打通一个最小 skill，例如：

- `popiskill-image-text2image-basic-v1`

并让 `popiartServer` 将它映射到一个明确的 `route_key` 和一个真实可用模型。

这一阶段才需要在 `PopiNewAPI` 中放可用渠道和 key。

### 第三阶段：扩展 skill 与路由

在文生图跑通后，再逐步加入：

- 图生图
- 图生视频
- 更多供应商和项目级路由覆盖

## 什么时候改哪个仓库

- 你在改命令体验、输出格式、配置逻辑：改 `popiartcli`
- 你在改 skill 调度、项目权限、artifact 规则、计费归因：改 `popiartServer`
- 你在改渠道、模型映射、供应商 key、原始模型代理：改 `PopiNewAPI`

如果一个改动同时需要触达三层，先从边界文档开始，再改代码。
