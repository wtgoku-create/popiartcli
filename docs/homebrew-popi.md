# homebrew-popi 初始化

本文档用于初始化 `wtgoku-create/homebrew-popi`，作为 `popiart` 的 Homebrew tap 仓库。

## 目标结构

建议新仓库初始化后至少包含：

```text
README.md
Formula/.gitkeep
```

其中：

- `README.md` 用于说明这个 tap 的用途和安装方式
- `Formula/` 用于存放 GoReleaser 生成的 `popiart.rb`

## 推荐仓库设置

建议新仓库使用：

- 仓库名：`homebrew-popi`
- 可见性：`public`
- 默认分支：`main`

原因：

- `brew tap` 默认面向公开仓库使用
- 当前 [`.goreleaser.yaml`](/Users/jiajia/popiartcli/.goreleaser.yaml#L54) 已按 `wtgoku-create/homebrew-popi` 和 `main` 分支配置

## 初始化步骤

1. 创建 GitHub 仓库 `wtgoku-create/homebrew-popi`
2. 确保默认分支为 `main`
3. 把 [README 模板](/Users/jiajia/popiartcli/docs/templates/homebrew-popi/README.md) 放到新仓库根目录
4. 创建空目录占位文件 `Formula/.gitkeep`
5. 提交第一次初始化 commit

如果你直接在本地初始化，新仓库最小内容可以是：

```sh
mkdir -p homebrew-popi/Formula
cp docs/templates/homebrew-popi/README.md homebrew-popi/README.md
touch homebrew-popi/Formula/.gitkeep
```

上面的 `README.md` 指的是本仓库中的模板文件：
[docs/templates/homebrew-popi/README.md](/Users/jiajia/popiartcli/docs/templates/homebrew-popi/README.md)

## 第一次接收 GoReleaser 提交的最小设置

GoReleaser 不会在 tap 仓库里跑 workflow。
它是在 `popiartcli` 的 release workflow 中，直接用 token 往 tap 仓库提交 `Formula/popiart.rb`。

所以最小要求是：

- tap 仓库已经存在
- tap 仓库默认分支是 `main`
- `popiartcli` 仓库的 GitHub Actions secret 中已经配置 `HOMEBREW_POPI_GITHUB_TOKEN`
- 这个 token 对 `wtgoku-create/homebrew-popi` 具备写权限

## Token 权限建议

如果使用经典 PAT，最少确保有：

- `repo`

如果使用 fine-grained PAT，最少确保：

- Repository access: `wtgoku-create/homebrew-popi`
- Repository permissions:
  - Contents: `Read and write`
  - Metadata: `Read`

然后把这个 token 配到源码仓库 `wtgoku-create/popiartcli` 的 Actions secret：

- `HOMEBREW_POPI_GITHUB_TOKEN`

## 分支保护注意事项

如果 `homebrew-popi` 开了 branch protection，GoReleaser 的 token 必须还能直接推送到 `main`。

否则第一次 release 时通常会失败在：

- 无法 push 到 `main`
- 无法创建或更新 `Formula/popiart.rb`

所以最小可用方案建议：

- 先不要给 `main` 开严格保护
- 或者给用于 release 的 bot / token 保留 bypass push 权限

## 首次发布后的结果

当你在源码仓库推送 tag，例如：

```sh
git tag v0.3.0
git push origin v0.3.0
```

GoReleaser 会做两件事：

1. 在 `wtgoku-create/popiartcli` 创建 GitHub Release 和二进制附件
2. 在 `wtgoku-create/homebrew-popi` 提交 `Formula/popiart.rb`

之后用户就可以执行：

```sh
brew tap wtgoku-create/popi
brew install wtgoku-create/popi/popiart
```

## 验收清单

第一次接通前，检查这几项：

- `wtgoku-create/homebrew-popi` 已创建
- 默认分支是 `main`
- 仓库是公开的
- `Formula/` 目录已存在
- `HOMEBREW_POPI_GITHUB_TOKEN` 已配置到 `wtgoku-create/popiartcli`
- token 对 tap 仓库有写权限
- [`.goreleaser.yaml`](/Users/jiajia/popiartcli/.goreleaser.yaml#L54) 中的 owner/name/branch 与真实仓库一致

## 可选增强

不是首次接通所必须，但后续可以补：

- 在 tap 仓库 README 里加版本徽章
- 增加更多 formula 的安装说明
- 把 `brew install` 例子改成同时展示 `brew upgrade`
- 在源码仓库 release 文档里加“首次发版前检查”章节
