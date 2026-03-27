---
name: popiskill-image-generate-edit-workflow-v1
description: 统一处理 PopiArt 中的图片生成与图片编辑工作流。当用户要生成图片、海报、视觉草图、信息图、封面图，或要修改、重绘、替换现有图片时使用此技能。默认将需求映射到 PopiArt 的 text2image 与 img2img runtime skills，并优先使用 artifact-based 编辑链路。
---

# 图片生成与编辑工作流

使用这个技能，把用户的图片需求落到 `popiart` 的真实命令面，而不是假设某个 provider 专属脚本一定存在。

这是一个 `popiartcli` 内置的 local bundled seed skill。
真正执行时，优先复用这两个 runtime skills：

- `popiskill-image-text2image-basic-v1`
- `popiskill-image-img2img-basic-v1`

如果只是生成一张新图，走 `text2image`。
如果已经有一张图，想做重绘、换风格、换背景、保留主体再编辑，走 `img2img`。

## 什么时候使用

当用户提出这些需求时，优先使用这个技能：

- “帮我生成一张图”
- “做一个封面/海报/缩略图/信息图”
- “把这张图改成另一个风格”
- “保留主体，但换场景/光线/镜头语言”
- “基于上一张图继续改”

## 核心原则

### 1. 保留用户原始提示词

默认直接使用用户原始完整输入作为 `prompt` 主体，不要先擅自改写成你自己的版本。

只有在信息明显不足时，才先补问关键缺失项，例如：

- 主体是谁
- 想生成新图，还是编辑已有图片
- 横图、竖图，还是方图
- 更偏写实、插画、海报，还是产品图

用户确认后的补充信息，应追加到原始提示词后，而不是替换原始提示词。

### 2. 优先用 `source_artifact_id` 做图生图

如果用户要编辑的是上一轮 PopiArt 已生成的图，优先使用：

```json
{
  "source_artifact_id": "art_xxx"
}
```

不要默认走远程图片 URL。

原因很简单：

- artifact 链路更稳定
- 不依赖第三方图床是否可访问
- 更适合 agent 连续工作流

只有在没有 PopiArt artifact 的情况下，再退回到：

- `reference_image_url`
- `image_url`

### 3. 以 `size` 作为稳定执行参数

当前 PopiArt 图片 runtime 的稳定执行参数是 `size`，例如：

- `1024x1024`
- `1536x1024`
- `1024x1536`
- `1792x1024`
- `1024x1792`

如果用户说的是“手机壁纸”“封面图”“头像”这类自然语言意图，先在工作流里推断出合适的 `size`，再提交到 `popiart run`。

推荐安全预设：

| 场景 | 推荐 `size` |
|---|---|
| 头像 / 方图 / 图标 | `1024x1024` |
| 横版封面 / 网页头图 / 桌面视觉 | `1536x1024` |
| 竖版海报 / 社媒单图 / 人像视觉 | `1024x1536` |
| 电影感宽画幅 / 视频封面 | `1792x1024` |
| 手机壁纸 / 竖版长构图 | `1024x1792` |

如果用户只给了宽高比概念，也可以这样理解：

- `1:1` -> `1024x1024`
- `16:9` -> `1792x1024`
- `9:16` -> `1024x1792`
- `3:2` / `4:3` 横构图 -> `1536x1024`
- `2:3` / `3:4` 竖构图 -> `1024x1536`

如果需求是极端长图、横幅或特殊比例，不要假装平台已经有稳定标准。
更稳妥的做法是：

1. 先生成最接近的安全预设尺寸
2. 再在后处理或二次编辑环节裁切

## 工作流

### A. 文生图

当没有 `source_artifact_id`、`reference_image_url`、`image_url` 时，默认走文生图：

```sh
popiart run popiskill-image-text2image-basic-v1 --input @params.json --wait
```

最小 payload：

```json
{
  "prompt": "一张高质感电影海报，主角在新西兰雪山上跳伞，强风、阳光、速度感、纪实摄影风格。",
  "size": "1024x1536"
}
```

### B. 图生图

当用户要编辑已有图片时，优先走 artifact-based `img2img`：

```sh
popiart run popiskill-image-img2img-basic-v1 --input @edit.json --wait
```

推荐 payload：

```json
{
  "source_artifact_id": "art_previous_result",
  "prompt": "保留主体身份、服装和核心配色，改成黄昏逆光、电影感、更强的速度感和更戏剧化的云层。",
  "size": "1024x1536"
}
```

如果只能使用远程图片 URL，再退回到：

```json
{
  "reference_image_url": "https://example.com/reference.png",
  "prompt": "保留主体身份与主要视觉特征，改成海边黄昏场景。",
  "size": "1024x1536"
}
```

## 长任务预期

在执行前，应明确告诉用户图片任务不是瞬时完成的。

可直接使用这种表述：

- “图片生成已启动，通常需要 20 秒到 2 分钟。”
- “图像编辑已启动，正在等待模型返回结果。”

如果是串联工作流：

- 第一步先生成基础图
- 第二步再基于 artifact 做编辑

则应明确告诉用户这是两段任务，不是单次调用。

## 输出处理

成功后，优先看：

- `job_id`
- `artifact_ids`

拉取结果：

```sh
popiart artifacts pull-all <job-id>
```

如果用户只要主图，也可以拉单个 artifact：

```sh
popiart artifacts pull <artifact-id>
```

## 推荐执行顺序

1. 判断是【生成新图】还是【编辑已有图】
2. 保留用户原始提示词，只在确认后追加补充
3. 根据用途推断 `size`
4. 文生图时使用 `popiskill-image-text2image-basic-v1`
5. 图生图时优先使用 `source_artifact_id`
6. 等待 job 完成并拉取 artifact

## 实战样例

先生成一张基础图：

```sh
popiart run popiskill-image-text2image-basic-v1 --input '{
  "prompt":"一只戴着护目镜的小熊在新西兰高空跳伞，阳光明亮，纪实冒险摄影风格",
  "size":"1024x1024"
}' --wait
```

再基于上一张图继续编辑：

```sh
popiart run popiskill-image-img2img-basic-v1 --input '{
  "source_artifact_id":"art_previous_result",
  "prompt":"保留小熊主体与护目镜，改成电影感傍晚金色光线，背景仍然是新西兰高空跳伞场景，更强的速度感和风压",
  "size":"1024x1024"
}' --wait
```

## 什么时候换别的技能

- 要做角色三视图，换 `popiskill-image-character-three-view-v1`
- 要做固定 Alice 主角展示图，换 `popiskill-image-img2img-popistudio-alice-showcase-v1`
- 要把图继续做成视频，换 `popiskill-video-image2video-basic-v1`
