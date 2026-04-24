# popiartServer 正式服发布交接

这份文档给发布同学使用。
目标是把已经在测试环境验证通过的 `popiartserver` 二进制上传到正式服务器，并完成替换、重启、验收与回滚准备。

## 本次交付内容

- 服务二进制部署路径：
  `/opt/popiartserver/bin/popiartserver`
- 建议工作目录：
  `/opt/popiartserver`
- 建议数据目录：
  `/opt/popiartserver/data`
- 建议源码快照归档目录：
  `/opt/popiartserver/src`

本次测试环境已验证通过的二进制特征：

- 默认图片模型：`gemini-3.1-flash-image-preview`
- 图片任务超时：`420` 秒
- `POPIART_NEWAPI_TOKEN` 不是必填前提
  说明：当前服务默认通过用户登录后的 session 绑定 upstream key 去调用 `PopiNewAPI`

## 二进制校验

当前已验证版本的 Linux `amd64` 二进制 SHA256：

```text
a13bed60a91badebae9b3af3ab9be9be9d675f70fc906e5f2c932d2986a1270c
```

发布同学在正式服落盘后，必须先校验：

```sh
sha256sum /opt/popiartserver/bin/popiartserver
```

结果应与上面的 SHA256 一致。

## 前置条件

正式服需要满足：

- `new-api` 已可用
- `popiartserver` 能访问 `PopiNewAPI`，默认文档按 `http://127.0.0.1:3000`
- 正式服上已有 `systemd`
- 具备写入 `/opt/popiartserver` 和 `/etc/systemd/system` 的权限

## systemd 配置

建议正式服 `systemd` 文件路径：

```text
/etc/systemd/system/popiartserver.service
```

参考内容：

```ini
[Unit]
Description=popiartServer
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/popiartserver
Environment=POPIART_SERVER_ADDR=0.0.0.0:18080
Environment=POPIART_NEWAPI_BASE_URL=http://127.0.0.1:3000
Environment=POPIART_DEFAULT_IMAGE_MODEL=gemini-3.1-flash-image-preview
Environment=POPIART_DEFAULT_VIDEO_MODEL=viduq2
Environment=POPIART_IMAGE_JOB_TIMEOUT_SECONDS=420
Environment=POPIART_SESSION_SECRET=<replace-with-prod-secret>
Environment=POPIART_DATA_DIR=/opt/popiartserver/data
Environment=POPIART_SQLITE_PATH=/opt/popiartserver/data/popiart.db
ExecStart=/opt/popiartserver/bin/popiartserver
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

注意：

- `POPIART_SESSION_SECRET` 不要复用测试环境值，正式服请重新生成
- 如果正式服确实需要服务级 fallback key，可额外配置：
  `POPIART_NEWAPI_TOKEN=<service-level-key>`
- 如果不配置 `POPIART_NEWAPI_TOKEN`，服务仍然可以通过用户 session 绑定的 upstream key 调用 `PopiNewAPI`

## 发布步骤

### 1. 准备目录

```sh
mkdir -p /opt/popiartserver/bin
mkdir -p /opt/popiartserver/data
mkdir -p /opt/popiartserver/src
```

### 2. 上传二进制

把交付的 `popiartserver` 二进制上传到正式服临时路径，例如：

```sh
scp ./popiartserver root@<prod-host>:/tmp/popiartserver.new
```

### 3. 校验 SHA256

```sh
sha256sum /tmp/popiartserver.new
```

必须匹配：

```text
a13bed60a91badebae9b3af3ab9be9be9d675f70fc906e5f2c932d2986a1270c
```

### 4. 备份旧版本并替换

```sh
TS=$(date +%Y%m%d%H%M%S)
cp /opt/popiartserver/bin/popiartserver /opt/popiartserver/bin/popiartserver.bak.$TS
install -m 755 /tmp/popiartserver.new /opt/popiartserver/bin/popiartserver
```

如果同时要存一份源码归档：

```sh
mv /tmp/popiartserver-src-snapshot.tgz /opt/popiartserver/src/popiartserver-src-snapshot.$TS.tgz
```

### 5. 刷新并重启服务

```sh
systemctl daemon-reload
systemctl restart popiartserver
systemctl status popiartserver --no-pager
```

## 发布后验收

### 1. 健康检查

```sh
curl -fsS http://127.0.0.1:18080/health
```

期望返回：

```json
{"ok":true,"service":"popiart-server-dev", ...}
```

### 2. 查看启动日志

```sh
journalctl -u popiartserver -n 20 --no-pager
```

至少应看到：

- `popiartServer listening on http://0.0.0.0:18080`
- `PopiNewAPI relay enabled via session-bound user keys: http://127.0.0.1:3000 ...`

### 3. 核对正式服配置项

```sh
egrep -n "POPIART_DEFAULT_IMAGE_MODEL|POPIART_DEFAULT_VIDEO_MODEL|POPIART_IMAGE_JOB_TIMEOUT_SECONDS|POPIART_NEWAPI_BASE_URL" /etc/systemd/system/popiartserver.service
```

重点确认：

- `POPIART_DEFAULT_IMAGE_MODEL=gemini-3.1-flash-image-preview`
- `POPIART_IMAGE_JOB_TIMEOUT_SECONDS=420`

### 4. CLI 冒烟

从一台能访问正式服的机器执行：

```sh
popiart --endpoint http://<prod-host>:18080/v1 auth whoami
popiart --endpoint http://<prod-host>:18080/v1 models routes
popiart --endpoint http://<prod-host>:18080/v1 skills list --search image --limit 5
```

`models routes` 里应至少看到：

```json
{
  "global": {
    "image": "gemini-3.1-flash-image-preview"
  }
}
```

### 5. 真实文生图冒烟

建议执行一次最小任务：

```sh
popiart --endpoint http://<prod-host>:18080/v1 run popiskill-image-text2image-basic-v1 \
  --input '{"prompt":"prod smoke test: a single red apple on a clean white background","size":"1024x1024"}' \
  --wait
```

期望：

- `status = done`
- 返回 `artifact_ids`
- `artifacts pull <artifact-id>` 可以成功下载

## 已验证修复点

这次交付中，已经确认修复了以下问题：

- `popiartserver` 之前图片任务超时只有 `3m`
- `Gemini` 文生图在当前链路下实际可能超过 `3m`
- 现在已把图片任务超时提升为 `420s`

如果正式服仍然出现 `context deadline exceeded`，优先排查：

- `POPIART_IMAGE_JOB_TIMEOUT_SECONDS` 是否遗漏
- `PopiNewAPI` 到 Gemini 上游的真实耗时
- `new-api` 容器或宿主机网络是否限流

## 回滚步骤

如果新版本上线后异常，直接回滚：

```sh
systemctl stop popiartserver
cp /opt/popiartserver/bin/popiartserver.bak.<timestamp> /opt/popiartserver/bin/popiartserver
systemctl start popiartserver
systemctl status popiartserver --no-pager
```

回滚后再做一次健康检查：

```sh
curl -fsS http://127.0.0.1:18080/health
```

## 备注

- 旧 artifact 如果依赖上游临时签名 URL，可能出现“元数据可读、内容下载过期”的现象
- 这个问题与本次二进制发布不是同一个问题
- 本次发布的重点是：让 `gemini-3.1-flash-image-preview` 作为默认图片模型时，`popiartserver` 不会因为本地超时过短而提前失败
