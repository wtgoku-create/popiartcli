package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	neturl "net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/types"
)

const (
	defaultMiniMaxMusicModelID         = "music-2.6-free"
	defaultMiniMaxSpeechModelID        = "speech-2.8-hd"
	defaultJimengActionTransferModelID = "jimeng_dreamactor_m20_gen_video"
)

func newImageCmd() *cobra.Command {
	imageCmd := &cobra.Command{
		Use:   "image [prompt]",
		Short: "围绕官方 image runtime 的意图化命令面",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && strings.TrimSpace(flagString(cmd, "prompt")) == "" {
				return cmd.Help()
			}
			payload, err := resolveText2ImageInput(cmd, args)
			if err != nil {
				return err
			}
			return executeSkillRun(cmd, officialText2ImageSkillID, payload, "image", nil)
		},
	}
	addText2ImageFlags(imageCmd)
	addCommonExecutionFlags(imageCmd)

	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "通过官方 text2image runtime 生成图片",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := resolveText2ImageInput(cmd, nil)
			if err != nil {
				return err
			}
			return executeSkillRun(cmd, officialText2ImageSkillID, payload, "image.generate", nil)
		},
	}
	addText2ImageFlags(generateCmd)
	addCommonExecutionFlags(generateCmd)

	img2imgCmd := &cobra.Command{
		Use:   "img2img",
		Short: "基于一张源图生成新图片",
		Long:  "默认映射到官方 img2img runtime。`--image` 接受本地文件路径、稳定媒体 URL 或支持的 data URL；本地文件会在需要时自动上传为 source artifact。复杂多图编辑时，推荐用 `--image` 传源场景、`--identity-reference-image` 传角色一致性参考、`--style-reference-image` 传风格参考；如果图片已经在 PopiArt 内部存在，优先改用对应的 `--*-artifact-id` 参数复用。",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, preview, err := resolveImageTransformInput(cmd)
			if err != nil {
				return err
			}
			return executeSkillRun(cmd, officialImage2ImageSkillID, payload, "image.img2img", preview)
		},
	}
	addCommonExecutionFlags(img2imgCmd)
	addImageTransformFlags(img2imgCmd)

	transformCmd := &cobra.Command{
		Use:   "transform",
		Short: "显式的 img2img 入口",
		Long:  "与 `popiart image img2img` 等价，但用更自然的 transform 命名暴露官方 img2img runtime。",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, preview, err := resolveImageTransformInput(cmd)
			if err != nil {
				return err
			}
			return executeSkillRun(cmd, officialImage2ImageSkillID, payload, "image.transform", preview)
		},
	}
	addCommonExecutionFlags(transformCmd)
	addImageTransformFlags(transformCmd)

	describeCmd := newImageDescribeCmd()

	imageCmd.AddCommand(generateCmd, img2imgCmd, transformCmd, describeCmd)
	return imageCmd
}

func newVideoCmd() *cobra.Command {
	videoCmd := &cobra.Command{
		Use:   "video [prompt]",
		Short: "围绕官方 video runtime 的意图化命令面",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && strings.TrimSpace(flagString(cmd, "prompt")) == "" && strings.TrimSpace(flagString(cmd, "from")) == "" && strings.TrimSpace(flagString(cmd, "image")) == "" {
				return cmd.Help()
			}
			payload, preview, err := resolveVideoGenerateInput(cmd, args)
			if err != nil {
				return err
			}
			return executeVideoGenerateCommand(cmd, payload, "video", preview)
		},
	}
	addVideoGenerateFlags(videoCmd)
	addCommonExecutionFlags(videoCmd)

	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "通用视频生成入口",
		Long:  "当前优先映射到官方 image2video runtime。传入 --image / --from 时可直接使用稳定 URL 或本地文件路径；本地文件会先自动上传为 source artifact。纯 prompt 的 text2video 路径会在 runtime baseline ready 后接入。",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, preview, err := resolveVideoGenerateInput(cmd, nil)
			if err != nil {
				return err
			}
			return executeVideoGenerateCommand(cmd, payload, "video.generate", preview)
		},
	}
	addCommonExecutionFlags(generateCmd)
	addVideoGenerateFlags(generateCmd)

	img2videoCmd := &cobra.Command{
		Use:   "img2video",
		Short: "显式的 image-to-video 入口",
		Long:  "与 `popiart video generate` 等价，但用更直接的 img2video 命名暴露官方 image2video runtime。",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, preview, err := resolveVideoGenerateInput(cmd, nil)
			if err != nil {
				return err
			}
			return executeVideoGenerateCommand(cmd, payload, "video.img2video", preview)
		},
	}
	addCommonExecutionFlags(img2videoCmd)
	addVideoGenerateFlags(img2videoCmd)

	fromImageCmd := &cobra.Command{
		Use:   "from-image",
		Short: "显式的 from-image 入口",
		Long:  "与 `popiart video generate` 等价，但用 from-image 命名强调当前是 image2video 路径。",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, preview, err := resolveVideoGenerateInput(cmd, nil)
			if err != nil {
				return err
			}
			return executeVideoGenerateCommand(cmd, payload, "video.from-image", preview)
		},
	}
	addCommonExecutionFlags(fromImageCmd)
	addVideoGenerateFlags(fromImageCmd)

	actionTransferCmd := &cobra.Command{
		Use:     "action-transfer",
		Aliases: []string{"motion-transfer", "dreamactor"},
		Short:   "通过即梦 DreamActor 做动作迁移",
		Long:    "提交一张身份图和一个动作参考视频到即梦动作迁移模型。默认模型为 jimeng_dreamactor_m20_gen_video，payload 会按统一网关规整为 images[0]、videos[0] 和 metadata.action=actionGenerate。",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, preview, err := resolveVideoActionTransferInput(cmd)
			if err != nil {
				return err
			}
			return executeDirectModelCommand(cmd, defaultJimengActionTransferModelID, payload, "video.action-transfer", preview)
		},
	}
	addCommonExecutionFlags(actionTransferCmd)
	addVideoActionTransferFlags(actionTransferCmd)

	videoCmd.AddCommand(generateCmd, img2videoCmd, fromImageCmd, actionTransferCmd)
	return videoCmd
}

func newAudioCmd() *cobra.Command {
	audioCmd := &cobra.Command{
		Use:   "audio",
		Short: "围绕官方 audio runtime 的意图化命令面",
	}

	ttsCmd := &cobra.Command{
		Use:   "tts",
		Short: "通过官方 TTS runtime 合成语音",
		RunE: func(cmd *cobra.Command, args []string) error {
			text, err := resolveTextInput(cmd)
			if err != nil {
				return err
			}

			payload := map[string]any{
				"text": text,
			}
			putString(payload, "voice", flagString(cmd, "voice"))
			putString(payload, "language", flagString(cmd, "language"))
			putString(payload, "voice_style", flagString(cmd, "voice-style"))
			putString(payload, "emotion", flagString(cmd, "emotion"))
			putString(payload, "format", flagString(cmd, "format"))
			putString(payload, "sound_effect", flagString(cmd, "sound-effect"))
			putString(payload, "notes", flagString(cmd, "notes"))
			putFloat(payload, "speed", flagFloat64(cmd, "speed"))
			putFloat(payload, "volume", flagFloat64(cmd, "volume"))
			putFloat(payload, "pitch", flagFloat64(cmd, "pitch"))
			putFloat(payload, "sample_rate_hz", flagFloat64(cmd, "sample-rate-hz"))
			putFloat(payload, "seed", flagFloat64(cmd, "seed"))
			putInt(payload, "bitrate", flagInt(cmd, "bitrate"))
			putInt(payload, "channels", flagInt(cmd, "channels"))
			putBool(payload, "subtitles", flagBool(cmd, "subtitles"))
			putStringSlice(payload, "pronunciation", flagStringArray(cmd, "pronunciation"))

			return executeDirectModelCommand(cmd, defaultMiniMaxSpeechModelID, payload, "audio.tts", nil)
		},
	}
	addCommonExecutionFlags(ttsCmd)
	addSpeechSynthesizeFlags(ttsCmd)

	audioCmd.AddCommand(ttsCmd)
	return audioCmd
}

func newSpeechCmd() *cobra.Command {
	speechCmd := &cobra.Command{
		Use:   "speech",
		Short: "围绕官方 speech runtime 的意图化命令面",
	}

	synthesizeCmd := &cobra.Command{
		Use:   "synthesize",
		Short: "通过官方 TTS runtime 合成语音",
		RunE: func(cmd *cobra.Command, args []string) error {
			text, err := resolveTextInput(cmd)
			if err != nil {
				return err
			}

			payload := map[string]any{
				"text": text,
			}
			putString(payload, "voice", flagString(cmd, "voice"))
			putString(payload, "language", flagString(cmd, "language"))
			putString(payload, "voice_style", flagString(cmd, "voice-style"))
			putString(payload, "emotion", flagString(cmd, "emotion"))
			putString(payload, "format", flagString(cmd, "format"))
			putString(payload, "sound_effect", flagString(cmd, "sound-effect"))
			putString(payload, "notes", flagString(cmd, "notes"))
			putFloat(payload, "speed", flagFloat64(cmd, "speed"))
			putFloat(payload, "volume", flagFloat64(cmd, "volume"))
			putFloat(payload, "pitch", flagFloat64(cmd, "pitch"))
			putFloat(payload, "sample_rate_hz", flagFloat64(cmd, "sample-rate-hz"))
			putFloat(payload, "seed", flagFloat64(cmd, "seed"))
			putInt(payload, "bitrate", flagInt(cmd, "bitrate"))
			putInt(payload, "channels", flagInt(cmd, "channels"))
			putBool(payload, "subtitles", flagBool(cmd, "subtitles"))
			putStringSlice(payload, "pronunciation", flagStringArray(cmd, "pronunciation"))

			return executeDirectModelCommand(cmd, defaultMiniMaxSpeechModelID, payload, "speech.synthesize", nil)
		},
	}
	addCommonExecutionFlags(synthesizeCmd)
	addSpeechSynthesizeFlags(synthesizeCmd)

	speechCmd.AddCommand(synthesizeCmd)
	return speechCmd
}

func newMusicCmd() *cobra.Command {
	musicCmd := &cobra.Command{
		Use:   "music [prompt]",
		Short: "围绕 MiniMax music 的意图化命令面",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && strings.TrimSpace(flagString(cmd, "prompt")) == "" && strings.TrimSpace(flagString(cmd, "lyrics")) == "" && strings.TrimSpace(flagString(cmd, "lyrics-file")) == "" {
				return cmd.Help()
			}
			payload, err := resolveMusicGenerateInput(cmd, args)
			if err != nil {
				return err
			}
			return executeDirectModelCommand(cmd, defaultMiniMaxMusicModelID, payload, "music", nil)
		},
	}
	addCommonExecutionFlags(musicCmd)
	addMusicGenerateFlags(musicCmd)

	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "通过 MiniMax music 模型生成音乐",
		Long:  "当前 `music generate` 直接走 MiniMax music 模型，默认使用 music-2.6-free。命令面参考 MiniMax CLI 的 music generate 设计。",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := resolveMusicGenerateInput(cmd, nil)
			if err != nil {
				return err
			}
			return executeDirectModelCommand(cmd, defaultMiniMaxMusicModelID, payload, "music.generate", nil)
		},
	}
	addCommonExecutionFlags(generateCmd)
	addMusicGenerateFlags(generateCmd)

	musicCmd.AddCommand(generateCmd)
	return musicCmd
}

func addCommonExecutionFlags(cmd *cobra.Command) {
	cmd.Flags().BoolP("wait", "w", false, "阻塞进程直到作业完成")
	cmd.Flags().String("interval", "2000", "轮询间隔（毫秒，默认：2000）")
	cmd.Flags().String("priority", "normal", "作业优先级: low | normal | high")
	cmd.Flags().String("idempotency-key", "", "用于安全重试的幂等键")
}

func addText2ImageFlags(cmd *cobra.Command) {
	cmd.Flags().String("model", "", "显式指定本次请求使用的模型；传入后会直接走 models infer")
	cmd.Flags().String("prompt", "", "图片提示词")
	cmd.Flags().String("negative-prompt", "", "排除项或不希望出现的元素")
	cmd.Flags().String("style", "", "风格提示，例如 anime、product render、cinematic realism")
	cmd.Flags().String("size", "", "精确尺寸，例如 1024x1024")
	cmd.Flags().String("aspect-ratio", "", "画幅比例，例如 1:1、16:9、9:16")
	cmd.Flags().Float64("seed", 0, "可选复现种子")
	cmd.Flags().String("notes", "", "额外约束说明")
}

func addImageTransformFlags(cmd *cobra.Command) {
	cmd.Flags().String("model", "", "显式指定本次请求使用的模型；传入后会直接走 models infer")
	cmd.Flags().String("image", "", "源图 URL 或本地文件路径")
	cmd.Flags().String("source-artifact-id", "", "已上传源图的 artifact_id")
	cmd.Flags().StringArray("identity-reference-image", nil, "主体一致性参考图 URL 或本地文件路径，可重复传入")
	cmd.Flags().StringArray("identity-reference-artifact-id", nil, "已上传主体一致性参考图的 artifact_id，可重复传入")
	cmd.Flags().StringArray("style-reference-image", nil, "风格参考图 URL 或本地文件路径，可重复传入")
	cmd.Flags().StringArray("style-reference-artifact-id", nil, "已上传风格参考图的 artifact_id，可重复传入")
	cmd.Flags().StringArray("reference-image", nil, "参考图 URL 或本地文件路径，可重复传入")
	cmd.Flags().StringArray("reference-artifact-id", nil, "已上传参考图的 artifact_id，可重复传入")
	cmd.Flags().String("prompt", "", "转换提示词")
	cmd.Flags().String("negative-prompt", "", "排除项或不希望出现的元素")
	cmd.Flags().Float64("strength", 0, "转换强度")
	cmd.Flags().Bool("preserve-composition", false, "尽量保留原始场景构图与机位关系")
	cmd.Flags().String("style", "", "视觉风格提示")
	cmd.Flags().String("size", "", "精确尺寸，例如 1024x1024")
	cmd.Flags().String("aspect-ratio", "", "画幅比例，例如 1:1、16:9、9:16")
	cmd.Flags().Float64("seed", 0, "可选复现种子")
	cmd.Flags().String("notes", "", "额外约束说明")
}

func addVideoGenerateFlags(cmd *cobra.Command) {
	cmd.Flags().String("model", "", "显式指定本次请求使用的模型；传入后会直接走 models infer")
	cmd.Flags().String("prompt-enhancer-model", "", "显式指定前置图像理解/提示词增强模型；传入后会先生成增强后的图生视频 prompt 再提交视频任务")
	cmd.Flags().String("from", "", "源图路径或 URL（等同于 --image）")
	cmd.Flags().String("image", "", "源图 URL 或本地文件路径")
	cmd.Flags().String("source-artifact-id", "", "已上传源图的 artifact_id")
	cmd.Flags().String("prompt", "", "动作或镜头提示词")
	cmd.Flags().String("negative-prompt", "", "排除项或不希望出现的运动/风格")
	cmd.Flags().Float64("duration", 0, "视频时长（秒）")
	cmd.Flags().Float64("fps", 0, "帧率提示")
	cmd.Flags().String("camera-motion", "", "镜头运动提示")
	cmd.Flags().String("motion-intensity", "", "运动强度提示")
	cmd.Flags().String("style", "", "视觉风格提示")
	cmd.Flags().String("aspect-ratio", "", "画幅比例，例如 16:9、9:16")
	cmd.Flags().Float64("seed", 0, "可选复现种子")
	cmd.Flags().String("notes", "", "额外约束说明")
}

func addVideoActionTransferFlags(cmd *cobra.Command) {
	cmd.Flags().String("model", defaultJimengActionTransferModelID, "即梦动作迁移模型；默认 jimeng_dreamactor_m20_gen_video")
	cmd.Flags().String("image", "", "身份图 URL、本地文件路径、纯 base64，或 data:image/*;base64 URL（会自动剥离前缀）")
	cmd.Flags().String("video", "", "动作参考视频 URL 或本地文件路径；会作为 videos[0] 提交")
	cmd.Flags().String("prompt", "", "可选动作迁移补充提示；即梦动作模仿可不传")
	cmd.Flags().String("action", "actionGenerate", "显式 metadata.action；默认 actionGenerate")
	cmd.Flags().Bool("cut-result-first-second-switch", false, "即梦动作模仿参数 cut_result_first_second_switch")
	cmd.Flags().String("notes", "", "额外约束说明")
}

func addSpeechSynthesizeFlags(cmd *cobra.Command) {
	cmd.Flags().String("model", defaultMiniMaxSpeechModelID, "显式指定本次请求使用的语音模型；默认使用 MiniMax speech-2.8-hd")
	cmd.Flags().String("text", "", "要合成的文本")
	cmd.Flags().String("text-file", "", "从文件读取文本；传 - 表示标准输入")
	cmd.Flags().String("voice", "", "语音 ID 或预设名")
	cmd.Flags().String("language", "", "语言标签，例如 zh-CN、en-US")
	cmd.Flags().String("voice-style", "", "语气、说话风格或表演方向")
	cmd.Flags().Float64("speed", 0, "语速倍率")
	cmd.Flags().Float64("volume", 0, "音量倍率")
	cmd.Flags().Float64("pitch", 0, "音高调整")
	cmd.Flags().String("emotion", "", "情感方向")
	cmd.Flags().String("format", "", "输出格式，例如 mp3、wav")
	cmd.Flags().Float64("sample-rate-hz", 0, "输出采样率提示")
	cmd.Flags().Int("bitrate", 0, "输出码率提示")
	cmd.Flags().Int("channels", 0, "输出声道数")
	cmd.Flags().Bool("subtitles", false, "返回字幕时间信息")
	cmd.Flags().StringArray("pronunciation", nil, "自定义发音映射，可重复传入")
	cmd.Flags().String("sound-effect", "", "附加音效提示")
	cmd.Flags().Float64("seed", 0, "可选复现种子")
	cmd.Flags().String("notes", "", "额外约束说明")
}

func addMusicGenerateFlags(cmd *cobra.Command) {
	cmd.Flags().String("model", defaultMiniMaxMusicModelID, "显式指定本次请求使用的音乐模型；默认使用 MiniMax music-2.6-free")
	cmd.Flags().String("prompt", "", "音乐风格或生成提示词")
	cmd.Flags().String("lyrics", "", "歌词文本")
	cmd.Flags().String("lyrics-file", "", "从文件读取歌词；传 - 表示标准输入")
	cmd.Flags().Bool("lyrics-optimizer", false, "根据 prompt 自动生成歌词")
	cmd.Flags().Bool("instrumental", false, "生成纯音乐（无歌词）")
	cmd.Flags().String("vocals", "", "人声风格，例如 warm male baritone")
	cmd.Flags().String("genre", "", "音乐流派，例如 folk、pop、jazz")
	cmd.Flags().String("mood", "", "情绪氛围，例如 warm、uplifting、melancholic")
	cmd.Flags().String("instruments", "", "主打乐器，例如 acoustic guitar, piano")
	cmd.Flags().String("tempo", "", "速度描述，例如 fast、slow、moderate")
	cmd.Flags().Int("bpm", 0, "精确 BPM")
	cmd.Flags().String("key", "", "调式，例如 C major、A minor")
	cmd.Flags().String("avoid", "", "希望避免的元素")
	cmd.Flags().String("use-case", "", "使用场景，例如 background music for video")
	cmd.Flags().String("structure", "", "歌曲结构，例如 verse-chorus-bridge")
	cmd.Flags().String("references", "", "参考曲目或歌手")
	cmd.Flags().String("extra", "", "额外细粒度要求")
	cmd.Flags().Bool("aigc-watermark", false, "嵌入 AI 生成内容水印")
	cmd.Flags().String("output-format", "", "网关响应格式: hex | url")
	cmd.Flags().Bool("stream", false, "请求流式音乐生成；仅允许 output-format=hex")
	cmd.Flags().String("format", "", "输出格式，例如 mp3、wav")
	cmd.Flags().Int("sample-rate-hz", 0, "输出采样率提示")
	cmd.Flags().Int("bitrate", 0, "输出码率提示")
	cmd.Flags().String("audio-url", "", "music-cover 参考音频 URL")
	cmd.Flags().String("audio-base64", "", "music-cover 参考音频 Base64")
}

func executeSkillRun(cmd *cobra.Command, skillID string, payload map[string]any, action string, extras map[string]any) error {
	if err := validateJobExecutionFlags(cmd); err != nil {
		return err
	}

	resolvedSkillID, err := resolveRunnableSkillID(context.Background(), skillID)
	if err != nil {
		return err
	}
	modelOverride := strings.TrimSpace(flagString(cmd, "model"))

	body := buildSkillJobBody(resolvedSkillID, payload, flagString(cmd, "priority"), flagString(cmd, "idempotency-key"))
	if dryRunMode(cmd) {
		if modelOverride != "" {
			directInput, err := normalizeOfficialRuntimeDirectInput(resolvedSkillID, payload)
			if err != nil {
				return err
			}
			preview := map[string]any{
				"skill_id":       resolvedSkillID,
				"model_id":       modelOverride,
				"execution_mode": "direct-model-override",
				"request": map[string]any{
					"method": "POST",
					"path":   "/models/infer",
					"body":   buildModelInferBody(modelOverride, directInput, flagString(cmd, "priority"), flagString(cmd, "idempotency-key")),
				},
			}
			for key, value := range extras {
				preview[key] = value
			}
			return writeDryRunPreview(cmd, action, preview)
		}
		preview := map[string]any{
			"skill_id": resolvedSkillID,
			"request": map[string]any{
				"method": "POST",
				"path":   "/jobs",
				"body":   body,
			},
		}
		for key, value := range extras {
			preview[key] = value
		}
		return writeDryRunPreview(cmd, action, preview)
	}

	if modelOverride != "" {
		directInput, err := normalizeOfficialRuntimeDirectInput(resolvedSkillID, payload)
		if err != nil {
			return err
		}
		job, err := submitModelInferJob(context.Background(), modelOverride, directInput, flagString(cmd, "priority"), "", flagString(cmd, "idempotency-key"))
		if err != nil {
			return err
		}
		job["requested_skill_id"] = resolvedSkillID
		job["model_id"] = modelOverride
		job["execution_mode"] = "direct-model-override"
		for key, value := range extras {
			job[key] = value
		}
		return writeJobResultOrWait(cmd, job)
	}

	if job, handled, err := maybeRunOfficialRuntimeDirectFallbackJob(context.Background(), resolvedSkillID, payload, flagString(cmd, "priority"), "", flagString(cmd, "idempotency-key")); handled {
		if err != nil {
			return err
		}
		for key, value := range extras {
			job[key] = value
		}
		return writeJobResultOrWait(cmd, job)
	}

	var job types.Job
	if err := currentClient().PostJSON(context.Background(), "/jobs", body, &job); err != nil {
		return err
	}
	return writeTypedJobResultOrWait(cmd, job)
}

func executeDirectModelCommand(cmd *cobra.Command, defaultModelID string, payload map[string]any, action string, extras map[string]any) error {
	if err := validateJobExecutionFlags(cmd); err != nil {
		return err
	}
	modelID := strings.TrimSpace(flagString(cmd, "model"))
	if modelID == "" {
		modelID = defaultModelID
	}
	if modelID == "" {
		return output.NewError("VALIDATION_ERROR", "缺少可用模型", map[string]any{
			"flag": "model",
			"hint": "请显式传入 --model，或为该命令配置默认模型",
		})
	}

	if dryRunMode(cmd) {
		preview := map[string]any{
			"model_id":       modelID,
			"execution_mode": directModelExecutionMode(modelID, defaultModelID),
			"request": map[string]any{
				"method": "POST",
				"path":   "/models/infer",
				"body":   buildModelInferBodyWithModelType(modelID, payload, flagString(cmd, "priority"), flagString(cmd, "idempotency-key"), directModelTypeForAction(action)),
			},
		}
		for key, value := range extras {
			preview[key] = value
		}
		return writeDryRunPreview(cmd, action, preview)
	}

	job, err := submitModelInferJobWithModelType(context.Background(), modelID, payload, flagString(cmd, "priority"), "", flagString(cmd, "idempotency-key"), directModelTypeForAction(action))
	if err != nil {
		return err
	}
	job["model_id"] = modelID
	job["execution_mode"] = directModelExecutionMode(modelID, defaultModelID)
	for key, value := range extras {
		job[key] = value
	}
	return writeJobResultOrWait(cmd, job)
}

func directModelExecutionMode(modelID, defaultModelID string) string {
	if strings.TrimSpace(modelID) != "" && strings.TrimSpace(defaultModelID) != "" && strings.TrimSpace(modelID) == strings.TrimSpace(defaultModelID) {
		return "direct-model-default"
	}
	return "direct-model-override"
}

func directModelTypeForAction(action string) string {
	switch strings.TrimSpace(action) {
	case "music", "music.generate":
		return "music"
	case "video.action-transfer":
		return "video"
	default:
		return ""
	}
}

func buildSkillJobBody(skillID string, payload map[string]any, priority, idempotencyKey string) map[string]any {
	return buildSkillJobBodyAny(skillID, payload, priority, idempotencyKey)
}

func buildSkillJobBodyAny(skillID string, payload any, priority, idempotencyKey string) map[string]any {
	cfg := config.Load()
	body := map[string]any{
		"skill_id": skillID,
		"input":    payload,
		"priority": priority,
	}
	if cfg.Project != "" {
		body["project_id"] = cfg.Project
	}
	if strings.TrimSpace(idempotencyKey) != "" {
		body["idempotency_key"] = strings.TrimSpace(idempotencyKey)
	}
	return body
}

func buildModelInferBody(modelID string, payload any, priority, idempotencyKey string) map[string]any {
	return buildModelInferBodyWithModelType(modelID, payload, priority, idempotencyKey, "")
}

func buildModelInferBodyWithModelType(modelID string, payload any, priority, idempotencyKey, modelType string) map[string]any {
	cfg := config.Load()
	body := map[string]any{
		"model_id": modelID,
		"input":    payload,
		"priority": priority,
	}
	if strings.TrimSpace(modelType) != "" {
		body["model_type"] = strings.TrimSpace(modelType)
	}
	if cfg.Project != "" {
		body["project_id"] = cfg.Project
	}
	if strings.TrimSpace(idempotencyKey) != "" {
		body["idempotency_key"] = strings.TrimSpace(idempotencyKey)
	}
	return body
}

func resolveText2ImageInput(cmd *cobra.Command, args []string) (map[string]any, error) {
	prompt := strings.TrimSpace(flagString(cmd, "prompt"))
	if prompt == "" && len(args) > 0 {
		prompt = strings.TrimSpace(args[0])
	}
	if prompt == "" {
		return nil, invalidFlagValueError("--prompt", "", "请传入图片生成提示词")
	}

	payload := map[string]any{
		"prompt": prompt,
	}
	putString(payload, "negative_prompt", flagString(cmd, "negative-prompt"))
	putString(payload, "style", flagString(cmd, "style"))
	putString(payload, "size", flagString(cmd, "size"))
	putString(payload, "aspect_ratio", normalizePortableAspectRatio(flagString(cmd, "aspect-ratio")))
	putString(payload, "notes", flagString(cmd, "notes"))
	putFloat(payload, "seed", flagFloat64(cmd, "seed"))
	return payload, nil
}

func resolveMusicGenerateInput(cmd *cobra.Command, args []string) (map[string]any, error) {
	prompt := strings.TrimSpace(flagString(cmd, "prompt"))
	if prompt == "" && len(args) > 0 {
		prompt = strings.TrimSpace(args[0])
	}

	lyrics, err := resolveOptionalTextInput(cmd, "lyrics", "lyrics-file")
	if err != nil {
		return nil, err
	}
	lyricsOptimizer := flagBool(cmd, "lyrics-optimizer")
	instrumental := flagBool(cmd, "instrumental")
	modelID := strings.TrimSpace(flagString(cmd, "model"))
	audioURL := strings.TrimSpace(flagString(cmd, "audio-url"))
	audioBase64 := strings.TrimSpace(flagString(cmd, "audio-base64"))
	outputFormat := strings.TrimSpace(flagString(cmd, "output-format"))
	stream := flagBool(cmd, "stream")
	isCoverModel := strings.HasPrefix(modelID, "music-cover")

	switch {
	case lyrics != "" && lyricsOptimizer:
		return nil, output.NewError("VALIDATION_ERROR", "lyrics-optimizer 不能与 lyrics 同时使用", map[string]any{
			"flags": []string{"lyrics", "lyrics-file", "lyrics-optimizer"},
		})
	case lyrics != "" && instrumental:
		return nil, output.NewError("VALIDATION_ERROR", "instrumental 不能与歌词同时使用", map[string]any{
			"flags": []string{"lyrics", "lyrics-file", "instrumental"},
		})
	case lyricsOptimizer && instrumental:
		return nil, output.NewError("VALIDATION_ERROR", "lyrics-optimizer 不能与 instrumental 同时使用", map[string]any{
			"flags": []string{"lyrics-optimizer", "instrumental"},
		})
	case outputFormat != "" && outputFormat != "hex" && outputFormat != "url":
		return nil, invalidFlagValueError("--output-format", outputFormat, "请传入 hex 或 url")
	case stream && outputFormat != "" && outputFormat != "hex":
		return nil, output.NewError("VALIDATION_ERROR", "stream=true 时只能使用 output-format=hex", map[string]any{
			"flags": []string{"stream", "output-format"},
		})
	case audioURL != "" && audioBase64 != "":
		return nil, output.NewError("VALIDATION_ERROR", "audio-url 和 audio-base64 只能提供一个", map[string]any{
			"flags": []string{"audio-url", "audio-base64"},
		})
	case isCoverModel && prompt == "":
		return nil, invalidFlagValueError("--prompt", "", "music-cover 模型必须传入 prompt")
	case isCoverModel && audioURL == "" && audioBase64 == "":
		return nil, output.NewError("VALIDATION_ERROR", "music-cover 模型必须传入 audio-url 或 audio-base64", map[string]any{
			"flags": []string{"audio-url", "audio-base64"},
		})
	case isCoverModel && (lyricsOptimizer || instrumental):
		return nil, output.NewError("VALIDATION_ERROR", "music-cover 模型不支持 lyrics-optimizer 或 instrumental", map[string]any{
			"flags": []string{"lyrics-optimizer", "instrumental"},
		})
	case !isCoverModel && (audioURL != "" || audioBase64 != ""):
		return nil, output.NewError("VALIDATION_ERROR", "music-2.6 模型不支持 audio-url 或 audio-base64", map[string]any{
			"flags": []string{"audio-url", "audio-base64"},
		})
	case !isCoverModel && instrumental && prompt == "":
		return nil, invalidFlagValueError("--prompt", "", "instrumental=true 时必须传入 prompt")
	case !isCoverModel && !instrumental && lyrics == "" && !lyricsOptimizer:
		return nil, invalidFlagValueError("--lyrics", "", "非纯音乐必须传入 lyrics，或使用 --lyrics-optimizer")
	case prompt == "" && lyrics == "":
		return nil, invalidFlagValueError("--prompt", "", "请传入 --prompt、--lyrics，或通过 --lyrics-file 提供歌词")
	}

	payload := map[string]any{}
	putString(payload, "prompt", prompt)
	putString(payload, "lyrics", lyrics)
	putString(payload, "output_format", outputFormat)
	putString(payload, "audio_url", audioURL)
	putString(payload, "audio_base64", audioBase64)
	putBool(payload, "lyrics_optimizer", lyricsOptimizer)
	putBool(payload, "is_instrumental", instrumental)
	putBool(payload, "stream", stream)
	putBool(payload, "aigc_watermark", flagBool(cmd, "aigc-watermark"))
	if audioSetting := musicAudioSetting(cmd); len(audioSetting) > 0 {
		payload["audio_setting"] = audioSetting
	}
	if promptAddendum := musicPromptAddendum(cmd); promptAddendum != "" {
		if payload["prompt"] == nil {
			payload["prompt"] = promptAddendum
		} else {
			payload["prompt"] = strings.TrimSpace(payload["prompt"].(string) + "\n\n" + promptAddendum)
		}
	}
	return payload, nil
}

func musicAudioSetting(cmd *cobra.Command) map[string]any {
	audioSetting := map[string]any{}
	putString(audioSetting, "format", flagString(cmd, "format"))
	putInt(audioSetting, "sample_rate", flagInt(cmd, "sample-rate-hz"))
	putInt(audioSetting, "bitrate", flagInt(cmd, "bitrate"))
	return audioSetting
}

func musicPromptAddendum(cmd *cobra.Command) string {
	parts := []string{}
	appendPart := func(label, flagName string) {
		if value := strings.TrimSpace(flagString(cmd, flagName)); value != "" {
			parts = append(parts, label+": "+value)
		}
	}
	appendPart("vocals", "vocals")
	appendPart("genre", "genre")
	appendPart("mood", "mood")
	appendPart("instruments", "instruments")
	appendPart("tempo", "tempo")
	appendPart("key", "key")
	appendPart("avoid", "avoid")
	appendPart("use case", "use-case")
	appendPart("structure", "structure")
	appendPart("references", "references")
	appendPart("extra", "extra")
	if bpm := flagInt(cmd, "bpm"); bpm > 0 {
		parts = append(parts, fmt.Sprintf("bpm: %d", bpm))
	}
	return strings.Join(parts, "\n")
}

func resolveVideoGenerateInput(cmd *cobra.Command, args []string) (map[string]any, map[string]any, error) {
	prompt := strings.TrimSpace(flagString(cmd, "prompt"))
	if prompt == "" && len(args) > 0 {
		prompt = strings.TrimSpace(args[0])
	}
	modelOverride := strings.TrimSpace(flagString(cmd, "model"))

	if !hasImageSourceInput(cmd) {
		if prompt == "" {
			return nil, nil, invalidFlagValueError("--prompt", "", "请传入视频提示词，或通过 --image / --from / --source-artifact-id 提供源图")
		}
		if modelOverride != "" {
			payload := map[string]any{}
			putString(payload, "prompt", prompt)
			putString(payload, "negative_prompt", flagString(cmd, "negative-prompt"))
			putString(payload, "camera_motion", flagString(cmd, "camera-motion"))
			putString(payload, "motion_intensity", flagString(cmd, "motion-intensity"))
			putString(payload, "style", flagString(cmd, "style"))
			putString(payload, "aspect_ratio", normalizePortableAspectRatio(flagString(cmd, "aspect-ratio")))
			putString(payload, "notes", flagString(cmd, "notes"))
			putFloat(payload, "duration_s", flagFloat64(cmd, "duration"))
			putFloat(payload, "fps", flagFloat64(cmd, "fps"))
			putFloat(payload, "seed", flagFloat64(cmd, "seed"))
			return payload, map[string]any{
				"mode": "prompt-only",
			}, nil
		}
		return nil, nil, output.NewError("CAPABILITY_UNAVAILABLE", "当前 video.generate 还未开放 text2video runtime", map[string]any{
			"command": "video generate",
			"hint":    "先通过 --image / --from / --source-artifact-id 走 image2video；等 runtime baseline ready 后再开放纯 prompt 视频生成",
		})
	}

	payload, preview, err := resolveImageSourceInput(cmd)
	if err != nil {
		return nil, nil, err
	}
	putString(payload, "prompt", prompt)
	putString(payload, "negative_prompt", flagString(cmd, "negative-prompt"))
	putString(payload, "camera_motion", flagString(cmd, "camera-motion"))
	putString(payload, "motion_intensity", flagString(cmd, "motion-intensity"))
	putString(payload, "style", flagString(cmd, "style"))
	putString(payload, "aspect_ratio", normalizePortableAspectRatio(flagString(cmd, "aspect-ratio")))
	putString(payload, "notes", flagString(cmd, "notes"))
	putFloat(payload, "duration_s", flagFloat64(cmd, "duration"))
	putFloat(payload, "fps", flagFloat64(cmd, "fps"))
	putFloat(payload, "seed", flagFloat64(cmd, "seed"))

	return payload, preview, nil
}

func resolveVideoActionTransferInput(cmd *cobra.Command) (map[string]any, map[string]any, error) {
	image := strings.TrimSpace(flagString(cmd, "image"))
	if image == "" {
		return nil, nil, invalidFlagValueError("--image", "", "动作迁移需要一张身份图")
	}
	video := strings.TrimSpace(flagString(cmd, "video"))
	if video == "" {
		return nil, nil, invalidFlagValueError("--video", "", "动作迁移需要一个动作参考视频")
	}

	imageValue, imageSource, imagePreflight, uploadedImage, err := resolveJimengImageGatewaySource(cmd, image)
	if err != nil {
		return nil, nil, err
	}
	videoValue, videoSource, videoPreflight, uploadedVideo, err := resolveGatewayMediaURLSource(cmd, video, "source_video")
	if err != nil {
		return nil, nil, err
	}

	metadata := map[string]any{}
	action := strings.TrimSpace(flagString(cmd, "action"))
	if action == "" {
		action = "actionGenerate"
	}
	metadata["action"] = action
	putBool(metadata, "cut_result_first_second_switch", flagBool(cmd, "cut-result-first-second-switch"))

	payload := map[string]any{
		"images":   []string{imageValue},
		"videos":   []string{videoValue},
		"metadata": metadata,
	}
	putString(payload, "prompt", flagString(cmd, "prompt"))
	putString(payload, "notes", flagString(cmd, "notes"))

	preview := map[string]any{
		"image_source": imageSource,
		"video_source": videoSource,
	}
	preflights := []map[string]any{}
	if imagePreflight != nil {
		preflights = append(preflights, imagePreflight)
	}
	if videoPreflight != nil {
		preflights = append(preflights, videoPreflight)
	}
	if len(preflights) > 0 {
		preview["preflight_uploads"] = preflights
	}
	uploads := []map[string]any{}
	if uploadedImage != nil {
		uploads = append(uploads, uploadedImage)
	}
	if uploadedVideo != nil {
		uploads = append(uploads, uploadedVideo)
	}
	if len(uploads) > 0 {
		preview["uploaded_media"] = uploads
	}
	return payload, preview, nil
}

func resolveJimengImageGatewaySource(cmd *cobra.Command, value string) (string, map[string]any, map[string]any, map[string]any, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil, nil, nil, invalidFlagValueError("--image", "", "动作迁移身份图不能为空")
	}
	if pureBase64, ok, err := stripImageDataURLPrefix(value); ok || err != nil {
		if err != nil {
			return "", nil, nil, nil, err
		}
		return pureBase64, map[string]any{
			"kind": "base64",
			"note": "data URL prefix stripped for Jimeng binary_data_base64 compatibility",
		}, nil, nil, nil
	}
	if looksLikeURL(value) {
		return value, map[string]any{
			"kind":  "url",
			"value": value,
		}, nil, nil, nil
	}
	if looksLikePureBase64Payload(value) {
		return value, map[string]any{
			"kind": "base64",
		}, nil, nil, nil
	}
	return resolveGatewayMediaURLSource(cmd, value, "source_image")
}

func resolveGatewayMediaURLSource(cmd *cobra.Command, value, role string) (string, map[string]any, map[string]any, map[string]any, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil, nil, nil, output.NewError("VALIDATION_ERROR", "媒体输入不能为空", map[string]any{
			"role": role,
		})
	}
	if looksLikeURL(value) {
		return value, map[string]any{
			"kind":  "url",
			"value": value,
			"role":  role,
		}, nil, nil, nil
	}
	if _, err := os.Stat(value); err != nil {
		return "", nil, nil, nil, output.NewError("CLI_ERROR", "读取媒体输入失败", map[string]any{
			"path":    value,
			"role":    role,
			"details": err.Error(),
		})
	}
	if dryRunMode(cmd) {
		placeholder := "(from artifacts.upload.url)"
		return placeholder, map[string]any{
				"kind":  "upload_url",
				"value": placeholder,
				"from":  value,
				"role":  role,
			}, map[string]any{
				"method": "POST",
				"path":   "/artifacts/upload",
				"body": map[string]any{
					"path":       value,
					"role":       role,
					"visibility": "unlisted",
				},
			}, nil, nil
	}

	uploaded, err := uploadArtifact(context.Background(), value, artifactUploadOptions{
		Role:       role,
		Visibility: "unlisted",
	})
	if err != nil {
		return "", nil, nil, nil, err
	}
	mediaURL := stringValue(uploaded["url"])
	if mediaURL == "" {
		return "", nil, nil, nil, output.NewError("CLI_ERROR", "上传媒体后缺少可提交给网关的 URL", map[string]any{
			"path":        value,
			"role":        role,
			"artifact_id": stringValue(uploaded["artifact_id"]),
			"hint":        "即梦动作迁移需要 images[0] / videos[0] 使用 URL 或纯 base64；请确认 artifact 服务返回稳定 URL，或直接传入 URL",
		})
	}
	return mediaURL, map[string]any{
		"kind":  "upload_url",
		"value": mediaURL,
		"from":  value,
		"role":  role,
	}, nil, uploaded, nil
}

func stripImageDataURLPrefix(value string) (string, bool, error) {
	if !looksLikeDataURL(value) {
		return "", false, nil
	}
	parts := strings.SplitN(value, ",", 2)
	if len(parts) != 2 {
		return "", true, output.NewError("INPUT_PARSE_ERROR", "不合法的图片 data URL", nil)
	}
	header := strings.ToLower(strings.TrimSpace(parts[0]))
	if !strings.Contains(header, ";base64") {
		return "", true, output.NewError("INPUT_PARSE_ERROR", "即梦图片 data URL 必须使用 base64 编码", map[string]any{
			"hint": "请传入 data:image/*;base64,xxxx，或直接传纯 base64 / URL / 本地文件",
		})
	}
	body := strings.TrimSpace(parts[1])
	if !looksLikePureBase64Payload(body) {
		return "", true, output.NewError("INPUT_PARSE_ERROR", "图片 data URL 的 base64 内容不合法", nil)
	}
	return body, true, nil
}

func looksLikePureBase64Payload(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 32 {
		return false
	}
	if strings.ContainsAny(value, " \t\r\n") {
		return false
	}
	if _, err := base64.StdEncoding.DecodeString(value); err == nil {
		return true
	}
	if _, err := base64.RawStdEncoding.DecodeString(value); err == nil {
		return true
	}
	if _, err := base64.URLEncoding.DecodeString(value); err == nil {
		return true
	}
	if _, err := base64.RawURLEncoding.DecodeString(value); err == nil {
		return true
	}
	return false
}

func hasImageSourceInput(cmd *cobra.Command) bool {
	image := strings.TrimSpace(flagString(cmd, "image"))
	from := strings.TrimSpace(flagString(cmd, "from"))
	sourceArtifactID := strings.TrimSpace(flagString(cmd, "source-artifact-id"))
	return image != "" || from != "" || sourceArtifactID != ""
}

func resolveImageTransformInput(cmd *cobra.Command) (map[string]any, map[string]any, error) {
	requiresArtifactSource := img2imgHasReferenceInputs(cmd)

	payload, preview, err := resolveImageTransformSourceInput(cmd, requiresArtifactSource)
	if err != nil {
		return nil, nil, err
	}

	referencePayload, referencePreview, err := resolveImageTransformReferenceInput(cmd)
	if err != nil {
		return nil, nil, err
	}
	mergeStringAnyMaps(payload, referencePayload)
	mergeStringAnyMaps(preview, referencePreview)

	prompt := strings.TrimSpace(flagString(cmd, "prompt"))
	if prompt == "" {
		return nil, nil, invalidFlagValueError("--prompt", "", "请传入 img2img 转换提示词")
	}
	payload["prompt"] = prompt
	putString(payload, "negative_prompt", flagString(cmd, "negative-prompt"))
	putString(payload, "style", flagString(cmd, "style"))
	putString(payload, "size", flagString(cmd, "size"))
	putString(payload, "aspect_ratio", normalizePortableAspectRatio(flagString(cmd, "aspect-ratio")))
	putString(payload, "notes", flagString(cmd, "notes"))
	putFloat(payload, "strength", flagFloat64(cmd, "strength"))
	putFloat(payload, "seed", flagFloat64(cmd, "seed"))
	putBool(payload, "preserve_composition", flagBool(cmd, "preserve-composition"))
	return payload, preview, nil
}

func resolveImageTransformSourceInput(cmd *cobra.Command, forceArtifactSource bool) (map[string]any, map[string]any, error) {
	sourceArtifactID := strings.TrimSpace(flagString(cmd, "source-artifact-id"))
	image := strings.TrimSpace(flagString(cmd, "image"))

	switch {
	case sourceArtifactID == "" && image == "":
		return nil, nil, invalidFlagValueError("--image", "", "请传入 --image 或 --source-artifact-id")
	case sourceArtifactID != "" && image != "":
		return nil, nil, conflictingAgentFlagsError("image", "source-artifact-id")
	}

	payload := map[string]any{}
	preview := map[string]any{}
	if sourceArtifactID != "" {
		payload["source_artifact_id"] = sourceArtifactID
		preview["source"] = map[string]any{
			"kind":  "artifact",
			"value": sourceArtifactID,
		}
		return payload, preview, nil
	}

	if forceArtifactSource {
		artifactID, uploaded, uploadPreview, err := resolveImageInputArtifact(cmd, image, "source")
		if err != nil {
			return nil, nil, err
		}
		payload["source_artifact_id"] = artifactID
		preview["source"] = map[string]any{
			"kind":  "artifact",
			"value": artifactID,
		}
		if uploadPreview != nil {
			preview["preflight"] = uploadPreview
		}
		if uploaded != nil {
			preview["uploaded_source_artifact"] = uploaded
		}
		return payload, preview, nil
	}

	if looksLikeURL(image) || looksLikeDataURL(image) {
		payload["image"] = image
		// Keep the portable `image` field while also sending compatibility
		// aliases so older server-side adapters can still resolve the source.
		payload["image_url"] = image
		payload["reference_image_url"] = image
		preview["source"] = map[string]any{
			"kind":  "image",
			"value": image,
		}
		return payload, preview, nil
	}

	if _, err := os.Stat(image); err != nil {
		return nil, nil, output.NewError("CLI_ERROR", "读取源图失败", map[string]any{
			"path":    image,
			"details": err.Error(),
		})
	}
	if dryRunMode(cmd) {
		payload["source_artifact_id"] = "(from artifacts.upload)"
		preview["preflight"] = map[string]any{
			"method": "POST",
			"path":   "/artifacts/upload",
			"body": map[string]any{
				"path":       image,
				"role":       "source",
				"visibility": "unlisted",
			},
		}
		return payload, preview, nil
	}

	uploaded, err := uploadArtifact(context.Background(), image, artifactUploadOptions{
		Role:       "source",
		Visibility: "unlisted",
	})
	if err != nil {
		return nil, nil, err
	}
	artifactID := stringValue(uploaded["artifact_id"])
	if artifactID == "" {
		return nil, nil, output.NewError("CLI_ERROR", "上传源图后缺少 artifact_id", map[string]any{
			"path": image,
		})
	}
	payload["source_artifact_id"] = artifactID
	preview["uploaded_source_artifact"] = uploaded
	return payload, preview, nil
}

func resolveImageTransformReferenceInput(cmd *cobra.Command) (map[string]any, map[string]any, error) {
	payload := map[string]any{}
	preview := map[string]any{}

	referenceArtifactIDs := cleanedStringSlice(flagStringArray(cmd, "reference-artifact-id"))
	referenceImages := cleanedStringSlice(flagStringArray(cmd, "reference-image"))
	identityReferenceArtifactIDs := cleanedStringSlice(flagStringArray(cmd, "identity-reference-artifact-id"))
	identityReferenceImages := cleanedStringSlice(flagStringArray(cmd, "identity-reference-image"))
	styleReferenceArtifactIDs := cleanedStringSlice(flagStringArray(cmd, "style-reference-artifact-id"))
	styleReferenceImages := cleanedStringSlice(flagStringArray(cmd, "style-reference-image"))
	if len(referenceArtifactIDs) == 0 && len(referenceImages) == 0 &&
		len(identityReferenceArtifactIDs) == 0 && len(identityReferenceImages) == 0 &&
		len(styleReferenceArtifactIDs) == 0 && len(styleReferenceImages) == 0 {
		return payload, preview, nil
	}

	generalReferenceArtifactIDs := append([]string(nil), referenceArtifactIDs...)
	previewSources := make([]map[string]any, 0, len(referenceArtifactIDs)+len(referenceImages)+len(identityReferenceArtifactIDs)+len(identityReferenceImages)+len(styleReferenceArtifactIDs)+len(styleReferenceImages))
	identityPreviewSources, identityReferenceArtifactIDs, preflightUploads, uploadedArtifacts, err := appendReferenceArtifacts(cmd, previewSources, nil, nil, nil, identityReferenceArtifactIDs, identityReferenceImages, "identity")
	if err != nil {
		return nil, nil, err
	}
	stylePreviewSources, styleReferenceArtifactIDs, preflightUploads, uploadedArtifacts, err := appendReferenceArtifacts(cmd, identityPreviewSources, nil, preflightUploads, uploadedArtifacts, styleReferenceArtifactIDs, styleReferenceImages, "style")
	if err != nil {
		return nil, nil, err
	}
	generalPreviewSources, generalReferenceArtifactIDs, preflightUploads, uploadedArtifacts, err := appendReferenceArtifacts(cmd, stylePreviewSources, nil, preflightUploads, uploadedArtifacts, generalReferenceArtifactIDs, referenceImages, "reference")
	if err != nil {
		return nil, nil, err
	}
	previewSources = generalPreviewSources

	combinedReferenceArtifactIDs := make([]string, 0, len(identityReferenceArtifactIDs)+len(styleReferenceArtifactIDs)+len(generalReferenceArtifactIDs))
	combinedReferenceArtifactIDs = append(combinedReferenceArtifactIDs, identityReferenceArtifactIDs...)
	combinedReferenceArtifactIDs = append(combinedReferenceArtifactIDs, styleReferenceArtifactIDs...)
	combinedReferenceArtifactIDs = append(combinedReferenceArtifactIDs, generalReferenceArtifactIDs...)
	putStringSlice(payload, "reference_artifact_ids", combinedReferenceArtifactIDs)
	putStringSlice(payload, "identity_reference_artifact_ids", identityReferenceArtifactIDs)
	putStringSlice(payload, "style_reference_artifact_ids", styleReferenceArtifactIDs)
	if len(previewSources) > 0 {
		preview["reference_sources"] = previewSources
	}
	if len(preflightUploads) > 0 {
		preview["reference_preflight_uploads"] = preflightUploads
	}
	if len(uploadedArtifacts) > 0 {
		preview["uploaded_reference_artifacts"] = uploadedArtifacts
	}
	return payload, preview, nil
}

func appendReferenceArtifacts(cmd *cobra.Command, previewSources []map[string]any, referenceArtifactIDs []string, preflightUploads []map[string]any, uploadedArtifacts []any, artifactIDs []string, images []string, role string) ([]map[string]any, []string, []map[string]any, []any, error) {
	for _, artifactID := range artifactIDs {
		referenceArtifactIDs = append(referenceArtifactIDs, artifactID)
		previewSources = append(previewSources, map[string]any{
			"kind":  "artifact",
			"value": artifactID,
			"role":  role,
		})
	}
	for _, value := range images {
		artifactID, uploaded, uploadPreview, err := resolveImageInputArtifact(cmd, value, "reference")
		if err != nil {
			return nil, nil, nil, nil, err
		}
		referenceArtifactIDs = append(referenceArtifactIDs, artifactID)
		if uploadPreview != nil {
			preflightUploads = append(preflightUploads, uploadPreview)
		}
		if uploaded != nil {
			uploadedArtifacts = append(uploadedArtifacts, uploaded)
		}
		previewSources = append(previewSources, map[string]any{
			"kind":  "artifact",
			"value": artifactID,
			"from":  value,
			"role":  role,
		})
	}
	return previewSources, referenceArtifactIDs, preflightUploads, uploadedArtifacts, nil
}

func img2imgHasReferenceInputs(cmd *cobra.Command) bool {
	return len(cleanedStringSlice(flagStringArray(cmd, "reference-artifact-id"))) > 0 ||
		len(cleanedStringSlice(flagStringArray(cmd, "reference-image"))) > 0 ||
		len(cleanedStringSlice(flagStringArray(cmd, "identity-reference-artifact-id"))) > 0 ||
		len(cleanedStringSlice(flagStringArray(cmd, "identity-reference-image"))) > 0 ||
		len(cleanedStringSlice(flagStringArray(cmd, "style-reference-artifact-id"))) > 0 ||
		len(cleanedStringSlice(flagStringArray(cmd, "style-reference-image"))) > 0
}

func resolveImageInputArtifact(cmd *cobra.Command, value, role string) (string, map[string]any, map[string]any, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil, nil, output.NewError("VALIDATION_ERROR", "图片输入不能为空", map[string]any{
			"role": role,
		})
	}

	if !looksLikeURL(value) && !looksLikeDataURL(value) {
		if _, err := os.Stat(value); err != nil {
			return "", nil, nil, output.NewError("CLI_ERROR", "读取图片输入失败", map[string]any{
				"path":    value,
				"role":    role,
				"details": err.Error(),
			})
		}
		if dryRunMode(cmd) {
			return "(from artifacts.upload)", nil, map[string]any{
				"method": "POST",
				"path":   "/artifacts/upload",
				"body": map[string]any{
					"path":       value,
					"role":       role,
					"visibility": "unlisted",
				},
			}, nil
		}

		uploaded, err := uploadArtifact(context.Background(), value, artifactUploadOptions{
			Role:       role,
			Visibility: "unlisted",
		})
		if err != nil {
			return "", nil, nil, err
		}
		artifactID := stringValue(uploaded["artifact_id"])
		if artifactID == "" {
			return "", nil, nil, output.NewError("CLI_ERROR", "上传图片后缺少 artifact_id", map[string]any{
				"path": value,
				"role": role,
			})
		}
		return artifactID, uploaded, nil, nil
	}

	if dryRunMode(cmd) {
		return "(from artifacts.upload)", nil, map[string]any{
			"method": "POST",
			"path":   "/artifacts/upload",
			"body": map[string]any{
				"source":     value,
				"role":       role,
				"visibility": "unlisted",
			},
		}, nil
	}

	filename, contentType, body, err := loadRemoteImageBytes(context.Background(), value)
	if err != nil {
		return "", nil, nil, err
	}

	tempFile, err := os.CreateTemp("", "popiart-image-*")
	if err != nil {
		return "", nil, nil, output.NewError("CLI_ERROR", "创建临时图片文件失败", map[string]any{
			"details": err.Error(),
			"role":    role,
		})
	}
	tempPath := tempFile.Name()
	if _, err := tempFile.Write(body); err != nil {
		tempFile.Close()
		_ = os.Remove(tempPath)
		return "", nil, nil, output.NewError("CLI_ERROR", "写入临时图片文件失败", map[string]any{
			"details": err.Error(),
			"role":    role,
		})
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", nil, nil, output.NewError("CLI_ERROR", "关闭临时图片文件失败", map[string]any{
			"details": err.Error(),
			"role":    role,
		})
	}
	defer os.Remove(tempPath)

	uploaded, err := uploadArtifact(context.Background(), tempPath, artifactUploadOptions{
		Filename:    filename,
		ContentType: contentType,
		Role:        role,
		Visibility:  "unlisted",
	})
	if err != nil {
		return "", nil, nil, err
	}
	artifactID := stringValue(uploaded["artifact_id"])
	if artifactID == "" {
		return "", nil, nil, output.NewError("CLI_ERROR", "上传远程图片后缺少 artifact_id", map[string]any{
			"source": value,
			"role":   role,
		})
	}
	return artifactID, uploaded, nil, nil
}

func loadRemoteImageBytes(ctx context.Context, value string) (string, string, []byte, error) {
	if looksLikeDataURL(value) {
		return decodeImageDataURL(value)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, value, nil)
	if err != nil {
		return "", "", nil, output.NewError("BAD_REQUEST", "构建远程图片请求失败", map[string]any{
			"source":  value,
			"details": err.Error(),
		})
	}
	req.Header.Set("User-Agent", "popiart-cli/0.1.0")

	client := &http.Client{Timeout: 60 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return "", "", nil, output.NewError("NETWORK_ERROR", "下载远程图片失败", map[string]any{
			"source":  value,
			"details": err.Error(),
		})
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", "", nil, output.NewError("NETWORK_ERROR", "下载远程图片失败", map[string]any{
			"source": value,
			"status": res.StatusCode,
		})
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", "", nil, output.NewError("NETWORK_ERROR", "读取远程图片失败", map[string]any{
			"source":  value,
			"details": err.Error(),
		})
	}

	contentType := strings.TrimSpace(res.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}
	filename := filenameFromImageURL(value, contentType)
	return filename, contentType, body, nil
}

func decodeImageDataURL(value string) (string, string, []byte, error) {
	parts := strings.SplitN(value, ",", 2)
	if len(parts) != 2 {
		return "", "", nil, output.NewError("INPUT_PARSE_ERROR", "不合法的 data URL 图片输入", nil)
	}

	header := strings.TrimSpace(parts[0])
	bodyPart := parts[1]
	contentType := strings.TrimPrefix(header, "data:")
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = contentType[:idx]
	}
	if contentType == "" {
		contentType = "image/png"
	}

	var body []byte
	var err error
	if strings.Contains(header, ";base64") {
		body, err = base64.StdEncoding.DecodeString(bodyPart)
	} else {
		decoded, decodeErr := neturl.QueryUnescape(bodyPart)
		err = decodeErr
		body = []byte(decoded)
	}
	if err != nil {
		return "", "", nil, output.NewError("INPUT_PARSE_ERROR", "解析 data URL 图片失败", map[string]any{
			"details": err.Error(),
		})
	}

	filename := "image" + extensionForContentType(contentType)
	return filename, contentType, body, nil
}

func filenameFromImageURL(value, contentType string) string {
	parsed, err := neturl.Parse(value)
	if err == nil {
		base := path.Base(parsed.Path)
		base = strings.TrimSpace(base)
		if base != "" && base != "." && base != "/" {
			return base
		}
	}
	return "image" + extensionForContentType(contentType)
}

func extensionForContentType(contentType string) string {
	if exts, _ := mime.ExtensionsByType(contentType); len(exts) > 0 {
		return exts[0]
	}
	return ".img"
}

func cleanedStringSlice(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func mergeStringAnyMaps(dst, src map[string]any) {
	if dst == nil || src == nil {
		return
	}
	for key, value := range src {
		dst[key] = value
	}
}

func resolveImageSourceInput(cmd *cobra.Command) (map[string]any, map[string]any, error) {
	sourceArtifactID := strings.TrimSpace(flagString(cmd, "source-artifact-id"))
	image := strings.TrimSpace(flagString(cmd, "image"))
	from := strings.TrimSpace(flagString(cmd, "from"))
	if image == "" && from != "" {
		image = from
	}

	switch {
	case sourceArtifactID == "" && image == "":
		return nil, nil, invalidFlagValueError("--image", "", "请传入 --image / --from 或 --source-artifact-id")
	case sourceArtifactID != "" && image != "":
		return nil, nil, conflictingAgentFlagsError("image", "source-artifact-id")
	}

	payload := map[string]any{}
	preview := map[string]any{}
	if sourceArtifactID != "" {
		payload["source_artifact_id"] = sourceArtifactID
		preview["source"] = map[string]any{
			"kind":  "artifact",
			"value": sourceArtifactID,
		}
		return payload, preview, nil
	}

	if looksLikeURL(image) {
		payload["image_url"] = image
		preview["source"] = map[string]any{
			"kind":  "url",
			"value": image,
		}
		return payload, preview, nil
	}

	if _, err := os.Stat(image); err != nil {
		return nil, nil, output.NewError("CLI_ERROR", "读取源图失败", map[string]any{
			"path":    image,
			"details": err.Error(),
		})
	}
	if dryRunMode(cmd) {
		payload["source_artifact_id"] = "(from artifacts.upload)"
		preview["preflight"] = map[string]any{
			"method": "POST",
			"path":   "/artifacts/upload",
			"body": map[string]any{
				"path":       image,
				"role":       "source",
				"visibility": "unlisted",
			},
		}
		return payload, preview, nil
	}

	uploaded, err := uploadArtifact(context.Background(), image, artifactUploadOptions{
		Role:       "source",
		Visibility: "unlisted",
	})
	if err != nil {
		return nil, nil, err
	}
	artifactID := stringValue(uploaded["artifact_id"])
	if artifactID == "" {
		return nil, nil, output.NewError("CLI_ERROR", "上传源图后缺少 artifact_id", map[string]any{
			"path": image,
		})
	}
	payload["source_artifact_id"] = artifactID
	preview["uploaded_source_artifact"] = uploaded
	return payload, preview, nil
}

func resolveTextInput(cmd *cobra.Command) (string, error) {
	text, err := resolveOptionalTextInput(cmd, "text", "text-file")
	if err != nil {
		return "", err
	}
	if text == "" {
		return "", invalidFlagValueError("--text", "", "请传入 --text 或 --text-file")
	}
	return text, nil
}

func resolveOptionalTextInput(cmd *cobra.Command, valueFlag, fileFlag string) (string, error) {
	value := flagString(cmd, valueFlag)
	filePath := strings.TrimSpace(flagString(cmd, fileFlag))

	switch {
	case strings.TrimSpace(value) != "" && filePath != "":
		return "", conflictingAgentFlagsError(valueFlag, fileFlag)
	case strings.TrimSpace(value) != "":
		return value, nil
	case filePath == "":
		return "", nil
	}

	var data []byte
	var err error
	if filePath == "-" {
		data, err = io.ReadAll(cmd.InOrStdin())
	} else {
		data, err = os.ReadFile(filePath)
	}
	if err != nil {
		return "", output.NewError("CLI_ERROR", "读取文本输入失败", map[string]any{
			"path":    filePath,
			"details": err.Error(),
		})
	}

	text := strings.TrimSpace(string(data))
	if text == "" {
		return "", invalidFlagValueError("--"+fileFlag, filePath, "输入文本不能为空")
	}
	return text, nil
}

func looksLikeURL(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://")
}

func looksLikeDataURL(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(value, "data:image/")
}

func putString(payload map[string]any, key, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	payload[key] = strings.TrimSpace(value)
}

func putFloat(payload map[string]any, key string, value float64) {
	if value == 0 {
		return
	}
	payload[key] = value
}

func putInt(payload map[string]any, key string, value int) {
	if value == 0 {
		return
	}
	payload[key] = value
}

func putBool(payload map[string]any, key string, value bool) {
	if !value {
		return
	}
	payload[key] = value
}

func putStringSlice(payload map[string]any, key string, values []string) {
	if len(values) == 0 {
		return
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		out = append(out, strings.TrimSpace(value))
	}
	if len(out) == 0 {
		return
	}
	payload[key] = out
}

func flagFloat64(cmd *cobra.Command, name string) float64 {
	if cmd == nil {
		return 0
	}
	value, _ := cmd.Flags().GetFloat64(name)
	return value
}

func flagInt(cmd *cobra.Command, name string) int {
	if cmd == nil {
		return 0
	}
	value, _ := cmd.Flags().GetInt(name)
	return value
}

func flagStringArray(cmd *cobra.Command, name string) []string {
	if cmd == nil {
		return nil
	}
	value, _ := cmd.Flags().GetStringArray(name)
	return value
}
