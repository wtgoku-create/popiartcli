package cmd

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/types"
)

func newImageCmd() *cobra.Command {
	imageCmd := &cobra.Command{
		Use:   "image",
		Short: "围绕官方 image runtime 的意图化命令面",
	}

	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "通过官方 text2image runtime 生成图片",
		RunE: func(cmd *cobra.Command, args []string) error {
			prompt := strings.TrimSpace(flagString(cmd, "prompt"))
			if prompt == "" {
				return invalidFlagValueError("--prompt", "", "请传入图片生成提示词")
			}

			payload := map[string]any{
				"prompt": prompt,
			}
			putString(payload, "negative_prompt", flagString(cmd, "negative-prompt"))
			putString(payload, "style", flagString(cmd, "style"))
			putString(payload, "size", flagString(cmd, "size"))
			putString(payload, "aspect_ratio", flagString(cmd, "aspect-ratio"))
			putString(payload, "notes", flagString(cmd, "notes"))
			putFloat(payload, "seed", flagFloat64(cmd, "seed"))

			return executeSkillRun(cmd, officialText2ImageSkillID, payload, "image.generate", nil)
		},
	}
	addCommonExecutionFlags(generateCmd)
	generateCmd.Flags().String("prompt", "", "图片提示词")
	generateCmd.Flags().String("negative-prompt", "", "排除项或不希望出现的元素")
	generateCmd.Flags().String("style", "", "风格提示，例如 anime、product render、cinematic realism")
	generateCmd.Flags().String("size", "", "精确尺寸，例如 1024x1024")
	generateCmd.Flags().String("aspect-ratio", "", "画幅比例，例如 1:1、16:9、9:16")
	generateCmd.Flags().Float64("seed", 0, "可选复现种子")
	generateCmd.Flags().String("notes", "", "额外约束说明")

	img2imgCmd := &cobra.Command{
		Use:   "img2img",
		Short: "基于一张源图生成新图片",
		Long:  "默认映射到官方 img2img runtime。传入 --image 时可直接使用稳定 URL 或本地文件路径；本地文件会先自动上传为 source artifact。",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, preview, err := resolveImageTransformInput(cmd)
			if err != nil {
				return err
			}
			return executeSkillRun(cmd, officialImage2ImageSkillID, payload, "image.img2img", preview)
		},
	}
	addCommonExecutionFlags(img2imgCmd)
	img2imgCmd.Flags().String("image", "", "源图 URL 或本地文件路径")
	img2imgCmd.Flags().String("source-artifact-id", "", "已上传源图的 artifact_id")
	img2imgCmd.Flags().String("prompt", "", "转换提示词")
	img2imgCmd.Flags().Float64("strength", 0, "转换强度")
	img2imgCmd.Flags().String("style", "", "视觉风格提示")
	img2imgCmd.Flags().String("size", "", "精确尺寸，例如 1024x1024")
	img2imgCmd.Flags().String("aspect-ratio", "", "画幅比例，例如 1:1、16:9、9:16")
	img2imgCmd.Flags().Float64("seed", 0, "可选复现种子")
	img2imgCmd.Flags().String("notes", "", "额外约束说明")

	imageCmd.AddCommand(generateCmd, img2imgCmd)
	return imageCmd
}

func newVideoCmd() *cobra.Command {
	videoCmd := &cobra.Command{
		Use:   "video",
		Short: "围绕官方 video runtime 的意图化命令面",
	}

	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "基于一张源图生成视频",
		Long:  "默认映射到官方 image2video runtime。传入 --image 时可直接使用稳定 URL 或本地文件路径；本地文件会先自动上传为 source artifact。",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, preview, err := resolveVideoGenerateInput(cmd)
			if err != nil {
				return err
			}
			return executeSkillRun(cmd, officialImage2VideoSkillID, payload, "video.generate", preview)
		},
	}
	addCommonExecutionFlags(generateCmd)
	generateCmd.Flags().String("image", "", "源图 URL 或本地文件路径")
	generateCmd.Flags().String("source-artifact-id", "", "已上传源图的 artifact_id")
	generateCmd.Flags().String("prompt", "", "动作或镜头提示词")
	generateCmd.Flags().String("negative-prompt", "", "排除项或不希望出现的运动/风格")
	generateCmd.Flags().Float64("duration", 0, "视频时长（秒）")
	generateCmd.Flags().Float64("fps", 0, "帧率提示")
	generateCmd.Flags().String("camera-motion", "", "镜头运动提示")
	generateCmd.Flags().String("motion-intensity", "", "运动强度提示")
	generateCmd.Flags().String("style", "", "视觉风格提示")
	generateCmd.Flags().String("aspect-ratio", "", "画幅比例，例如 16:9、9:16")
	generateCmd.Flags().Float64("seed", 0, "可选复现种子")
	generateCmd.Flags().String("notes", "", "额外约束说明")

	img2videoCmd := &cobra.Command{
		Use:   "img2video",
		Short: "显式的 image-to-video 入口",
		Long:  "与 `popiart video generate` 等价，但用更直接的 img2video 命名暴露官方 image2video runtime。",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, preview, err := resolveVideoGenerateInput(cmd)
			if err != nil {
				return err
			}
			return executeSkillRun(cmd, officialImage2VideoSkillID, payload, "video.img2video", preview)
		},
	}
	addCommonExecutionFlags(img2videoCmd)
	img2videoCmd.Flags().String("image", "", "源图 URL 或本地文件路径")
	img2videoCmd.Flags().String("source-artifact-id", "", "已上传源图的 artifact_id")
	img2videoCmd.Flags().String("prompt", "", "动作或镜头提示词")
	img2videoCmd.Flags().String("negative-prompt", "", "排除项或不希望出现的运动/风格")
	img2videoCmd.Flags().Float64("duration", 0, "视频时长（秒）")
	img2videoCmd.Flags().Float64("fps", 0, "帧率提示")
	img2videoCmd.Flags().String("camera-motion", "", "镜头运动提示")
	img2videoCmd.Flags().String("motion-intensity", "", "运动强度提示")
	img2videoCmd.Flags().String("style", "", "视觉风格提示")
	img2videoCmd.Flags().String("aspect-ratio", "", "画幅比例，例如 16:9、9:16")
	img2videoCmd.Flags().Float64("seed", 0, "可选复现种子")
	img2videoCmd.Flags().String("notes", "", "额外约束说明")

	videoCmd.AddCommand(generateCmd, img2videoCmd)
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
			putString(payload, "provider", flagString(cmd, "provider"))
			putString(payload, "voice_style", flagString(cmd, "voice-style"))
			putString(payload, "emotion", flagString(cmd, "emotion"))
			putString(payload, "format", flagString(cmd, "format"))
			putString(payload, "notes", flagString(cmd, "notes"))
			putFloat(payload, "speed", flagFloat64(cmd, "speed"))
			putFloat(payload, "sample_rate_hz", flagFloat64(cmd, "sample-rate-hz"))
			putFloat(payload, "seed", flagFloat64(cmd, "seed"))

			return executeSkillRun(cmd, officialTTSMultimodelSkillID, payload, "audio.tts", nil)
		},
	}
	addCommonExecutionFlags(ttsCmd)
	ttsCmd.Flags().String("text", "", "要合成的文本")
	ttsCmd.Flags().String("text-file", "", "从文件读取文本；传 - 表示标准输入")
	ttsCmd.Flags().String("voice", "", "语音 ID 或预设名")
	ttsCmd.Flags().String("language", "", "语言标签，例如 zh-CN、en-US")
	ttsCmd.Flags().String("provider", "", "可选 provider / route hint")
	ttsCmd.Flags().String("voice-style", "", "语气、说话风格或表演方向")
	ttsCmd.Flags().Float64("speed", 0, "语速倍率")
	ttsCmd.Flags().String("emotion", "", "情感方向")
	ttsCmd.Flags().String("format", "", "输出格式，例如 mp3、wav")
	ttsCmd.Flags().Float64("sample-rate-hz", 0, "输出采样率提示")
	ttsCmd.Flags().Float64("seed", 0, "可选复现种子")
	ttsCmd.Flags().String("notes", "", "额外约束说明")

	audioCmd.AddCommand(ttsCmd)
	return audioCmd
}

func addCommonExecutionFlags(cmd *cobra.Command) {
	cmd.Flags().BoolP("wait", "w", false, "阻塞进程直到作业完成")
	cmd.Flags().String("interval", "2000", "轮询间隔（毫秒，默认：2000）")
	cmd.Flags().String("priority", "normal", "作业优先级: low | normal | high")
	cmd.Flags().String("idempotency-key", "", "用于安全重试的幂等键")
}

func executeSkillRun(cmd *cobra.Command, skillID string, payload map[string]any, action string, extras map[string]any) error {
	if err := validateJobExecutionFlags(cmd); err != nil {
		return err
	}

	resolvedSkillID, err := resolveRunnableSkillID(context.Background(), skillID)
	if err != nil {
		return err
	}

	body := buildSkillJobBody(resolvedSkillID, payload, flagString(cmd, "priority"), flagString(cmd, "idempotency-key"))
	if dryRunMode(cmd) {
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

func resolveVideoGenerateInput(cmd *cobra.Command) (map[string]any, map[string]any, error) {
	payload, preview, err := resolveImageSourceInput(cmd)
	if err != nil {
		return nil, nil, err
	}

	putString(payload, "prompt", flagString(cmd, "prompt"))
	putString(payload, "negative_prompt", flagString(cmd, "negative-prompt"))
	putString(payload, "camera_motion", flagString(cmd, "camera-motion"))
	putString(payload, "motion_intensity", flagString(cmd, "motion-intensity"))
	putString(payload, "style", flagString(cmd, "style"))
	putString(payload, "aspect_ratio", flagString(cmd, "aspect-ratio"))
	putString(payload, "notes", flagString(cmd, "notes"))
	putFloat(payload, "duration_s", flagFloat64(cmd, "duration"))
	putFloat(payload, "fps", flagFloat64(cmd, "fps"))
	putFloat(payload, "seed", flagFloat64(cmd, "seed"))

	return payload, preview, nil
}

func resolveImageTransformInput(cmd *cobra.Command) (map[string]any, map[string]any, error) {
	payload, preview, err := resolveImageSourceInput(cmd)
	if err != nil {
		return nil, nil, err
	}

	prompt := strings.TrimSpace(flagString(cmd, "prompt"))
	if prompt == "" {
		return nil, nil, invalidFlagValueError("--prompt", "", "请传入 img2img 转换提示词")
	}
	payload["prompt"] = prompt
	putString(payload, "style", flagString(cmd, "style"))
	putString(payload, "size", flagString(cmd, "size"))
	putString(payload, "aspect_ratio", flagString(cmd, "aspect-ratio"))
	putString(payload, "notes", flagString(cmd, "notes"))
	putFloat(payload, "strength", flagFloat64(cmd, "strength"))
	putFloat(payload, "seed", flagFloat64(cmd, "seed"))
	return payload, preview, nil
}

func resolveImageSourceInput(cmd *cobra.Command) (map[string]any, map[string]any, error) {
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
	text := flagString(cmd, "text")
	textFile := strings.TrimSpace(flagString(cmd, "text-file"))

	switch {
	case strings.TrimSpace(text) != "" && textFile != "":
		return "", conflictingAgentFlagsError("text", "text-file")
	case strings.TrimSpace(text) != "":
		return text, nil
	case textFile == "":
		return "", invalidFlagValueError("--text", "", "请传入 --text 或 --text-file")
	}

	var data []byte
	var err error
	if textFile == "-" {
		data, err = io.ReadAll(cmd.InOrStdin())
	} else {
		data, err = os.ReadFile(textFile)
	}
	if err != nil {
		return "", output.NewError("CLI_ERROR", "读取文本输入失败", map[string]any{
			"path":    textFile,
			"details": err.Error(),
		})
	}

	value := strings.TrimSpace(string(data))
	if value == "" {
		return "", invalidFlagValueError("--text-file", textFile, "输入文本不能为空")
	}
	return value, nil
}

func looksLikeURL(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://")
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

func flagFloat64(cmd *cobra.Command, name string) float64 {
	if cmd == nil {
		return 0
	}
	value, _ := cmd.Flags().GetFloat64(name)
	return value
}
