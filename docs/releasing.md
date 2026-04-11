# 发布说明

`popiart` 现在只以 Go 版本作为正式 CLI 发布。
Node.js 原型代码仅保留在仓库中供迁移参考，不参与对外分发。

## 发布产物

`GoReleaser` 会在打 tag 后自动生成以下内容：

- GitHub Release
- Gitee 镜像 release
- `darwin/linux/windows` 的 `amd64/arm64` 二进制压缩包
- `checksums.txt`
- Homebrew formula 更新

归档文件名格式：

```text
popiart_<version>_<os>_<arch>.tar.gz
popiart_<version>_windows_<arch>.zip
```

例如：

```text
popiart_0.3.4_darwin_arm64.tar.gz
```

## GitHub Actions Secret

发布 workflow 默认使用：

- `GITHUB_TOKEN`
  用于创建当前仓库的 GitHub Release
- `HOMEBREW_POPI_GITHUB_TOKEN`
  用于把 formula 提交到 Homebrew tap 仓库

`HOMEBREW_POPI_GITHUB_TOKEN` 需要有目标 tap 仓库的写权限。

## Homebrew tap 方案

当前 `.goreleaser.yaml` 默认约定：

- 源码仓库：`wtgoku-create/popiartcli`
- 国内镜像仓库：`wattx/popiartcli`
- tap 仓库：`wtgoku-create/homebrew-popi`
- formula 路径：`Formula/popiart.rb`

外部用户安装方式：

```sh
brew tap wtgoku-create/popi
brew install wtgoku-create/popi/popiart
```

初始化 tap 仓库时，可直接参考：

- [docs/homebrew-popi.md](./homebrew-popi.md)
- [docs/templates/homebrew-popi/README.md](./templates/homebrew-popi/README.md)

## 发布流程

1. 确保主分支代码已经可构建，并且 `go test ./...` 通过
2. 创建并推送语义化版本 tag，例如 `v0.3.4`
3. GitHub Actions 触发 `.github/workflows/release.yml`
4. GoReleaser 生成 GitHub release、checksums 和 Homebrew formula 更新
5. 把同一个 tag、release 附件和 `checksums.txt` 同步到 Gitee `wattx/popiartcli`
6. 通过安装脚本安装的用户可以直接运行 `popiart update` 获取这个新版本；国内镜像用户可运行 `popiart update --source gitee`；Homebrew 用户使用 `brew upgrade wtgoku-create/popi/popiart`

命令示例：

```sh
git tag v0.3.4
git push origin v0.3.4
```

## 国内镜像同步

为了让国内用户可以直接使用：

- `https://gitee.com/wattx/popiartcli/raw/main/install.sh`
- `https://gitee.com/wattx/popiartcli/raw/main/install.ps1`
- `https://gitee.com/wattx/popiartcli/releases/download/<tag>/...`

每次正式发布后，需要把以下内容同步到 Gitee：

- 源码分支与 tag
- 对应 tag 的 release 说明
- 平台压缩包
- `checksums.txt`

如果 Gitee 侧只同步仓库源码、没有同步 release 二进制，`popiart update --source gitee` 和国内镜像安装脚本都无法完成升级。

## 本地预演

安装 `goreleaser` 后，可以先做一次本地快照构建：

```sh
goreleaser release --snapshot --clean
```

这会在不真正发布 GitHub Release 的情况下验证打包配置。
