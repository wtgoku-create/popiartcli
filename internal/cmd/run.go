package cmd

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/input"
	"github.com/wtgoku-create/popiartcli/internal/localskills"
	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/popiart"
	"github.com/wtgoku-create/popiartcli/internal/seed"
)

func newRunCmd() *cobra.Command {
	runCmd := &cobra.Command{
		Use:   "run <skill-id>",
		Short: "调用一个技能并返回一个 job_id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateJobExecutionFlags(cmd); err != nil {
				return err
			}

			payload, err := input.Resolve(flagString(cmd, "input"))
			if err != nil {
				return err
			}
			normalizeImagePayloadOptions(payload)
			resolvedSkillID, err := resolveRunnableSkillID(context.Background(), args[0])
			if err != nil {
				return err
			}
			if dryRunMode(cmd) {
				return writeDryRunPreview(cmd, "run", map[string]any{
					"skill_id": resolvedSkillID,
					"hint":     "当前模式下 `run` 仍依赖旧 jobs/skills 体系；请改用 image/video/audio/speech/music 等主站任务命令",
				})
			}
			if bridgedPayload, mapper, extras, bridgeErr := resolveRunTaskBridgePayload(cmd, resolvedSkillID, payload); bridgeErr != nil {
				return bridgeErr
			} else if bridgedPayload != nil && mapper != nil {
				return executeTaskCommand(cmd, runBridgeActionForSkill(resolvedSkillID), bridgedPayload, mapper, extras)
			}
			return output.NewError("UNSUPPORTED_IN_POPI_ART_MODE", "当前模式不支持通过 CLI 调用远端 skill 运行入口", map[string]any{
				"skill_id": resolvedSkillID,
				"hint":     "run 仍依赖旧 jobs/skills 体系；请改用 image/video/audio/speech/music 等主站任务命令，或等待 run API 迁移完成",
			})
		},
	}

	runCmd.Flags().StringP("input", "i", "", "输入 JSON 字符串、@file.json，或用 - 表示标准输入")
	runCmd.Flags().BoolP("wait", "w", false, "阻塞进程直到作业完成")
	runCmd.Flags().String("interval", "2000", "轮询间隔（毫秒，默认：2000）")
	runCmd.Flags().String("priority", "normal", "作业优先级: low | normal | high")
	runCmd.Flags().String("idempotency-key", "", "用于安全重试的幂等键")
	return runCmd
}

func resolveRunnableSkillID(ctx context.Context, skillID string) (string, error) {
	skillID = strings.TrimSpace(skillID)

	if installed, shouldUseLocal, err := resolveRunnableInstalledSkill(ctx, skillID); err != nil {
		return "", err
	} else if shouldUseLocal {
		return localskillsEffectiveRuntimeSkillID(installed), nil
	}

	if err := validateBundledSkillRun(ctx, skillID); err != nil {
		return "", err
	}
	return skillID, nil
}

func resolveRunnableInstalledSkill(ctx context.Context, skillID string) (localskills.InstalledSkill, bool, error) {
	skill, ok, err := localskills.FindInstalled(skillID)
	if err != nil {
		return localskills.InstalledSkill{}, false, err
	}
	if !ok {
		return localskills.InstalledSkill{}, false, nil
	}

	active, err := localskills.IsActive(skill.Manifest.Slug)
	if err != nil {
		return localskills.InstalledSkill{}, false, err
	}
	if !active {
		exists, err := remoteSkillExists(ctx, skillID)
		if err != nil {
			return localskills.InstalledSkill{}, false, err
		}
		if exists {
			return localskills.InstalledSkill{}, false, nil
		}
	}

	if skill.Manifest.RequiresPopiartAuth {
		if _, err := config.RequireToken(); err != nil {
			return localskills.InstalledSkill{}, false, requireTokenError()
		}
	}

	if skill.Manifest.Execution.Runner != "" && skill.Manifest.Execution.Runner != "popiart" {
		return localskills.InstalledSkill{}, false, output.NewError("LOCAL_SKILL_UNSUPPORTED", "当前仅支持由 popiart 执行的本地 skill", map[string]any{
			"skill_id": skillID,
			"runner":   skill.Manifest.Execution.Runner,
		})
	}
	if skill.Manifest.Execution.Mode != "remote-runtime" {
		return localskills.InstalledSkill{}, false, output.NewError("LOCAL_SKILL_UNSUPPORTED", "当前仅支持 execution.mode=remote-runtime 的本地 skill", map[string]any{
			"skill_id": skillID,
			"mode":     skill.Manifest.Execution.Mode,
		})
	}

	return skill, true, nil
}

func validateBundledSkillRun(ctx context.Context, skillID string) error {
	if _, ok := seed.FindBundledSkill(skillID); !ok {
		return nil
	}
	if _, ok := officialRuntimeSkillForID(skillID); ok {
		return nil
	}

	exists, err := remoteSkillExists(ctx, skillID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return output.NewError("LOCAL_ONLY_SKILL", "该 skill 仅是 CLI 内置 seed helper，不能直接提交到远端执行", map[string]any{
		"skill_id": skillID,
		"hint":     "使用 `popiart skills get " + skillID + "` 查看说明，或选择对应的远程 runtime skill 再执行 `popiart run`",
	})
}

func runBridgeActionForSkill(skillID string) string {
	switch strings.TrimSpace(skillID) {
	case officialText2ImageSkillID:
		return "image.generate"
	case officialImage2ImageSkillID, officialAliceImageShowcaseSkillID:
		return "image.img2img"
	case officialImage2VideoSkillID, officialAliceVideoShowcaseSkillID:
		return "video.img2video"
	case officialTTSMultimodelSkillID:
		return "audio.tts"
	default:
		return ""
	}
}

func resolveRunTaskBridgePayload(cmd *cobra.Command, skillID string, raw any) (map[string]any, func(map[string]any, popiart.Model) popiart.TaskRequest, map[string]any, error) {
	payload, ok := raw.(map[string]any)
	if !ok {
		return nil, nil, nil, output.NewError("VALIDATION_ERROR", "run skill 需要 JSON object 输入", map[string]any{
			"skill_id": skillID,
		})
	}

	switch strings.TrimSpace(skillID) {
	case officialText2ImageSkillID:
		taskPayload, err := bridgeRunTextToImagePayload(payload)
		return taskPayload, popiart.BuildTextToImageTaskRequest, runBridgeExtras(skillID), err
	case officialImage2ImageSkillID, officialAliceImageShowcaseSkillID:
		taskPayload, extras, err := bridgeRunImageToImagePayload(cmd, payload)
		return taskPayload, popiart.BuildImageTransformTaskRequest, mergeRunBridgeExtras(skillID, extras), err
	case officialImage2VideoSkillID, officialAliceVideoShowcaseSkillID:
		taskPayload, extras, err := bridgeRunImageToVideoPayload(cmd, payload)
		return taskPayload, popiart.BuildVideoGenerateTaskRequest, mergeRunBridgeExtras(skillID, extras), err
	case officialTTSMultimodelSkillID:
		taskPayload, err := bridgeRunTTSPayload(payload)
		return taskPayload, popiart.BuildTextToSpeechTaskRequest, runBridgeExtras(skillID), err
	default:
		return nil, nil, nil, nil
	}
}

func runBridgeExtras(skillID string) map[string]any {
	return map[string]any{
		"requested_skill_id": skillID,
		"execution_mode":     "run-task-bridge",
	}
}

func mergeRunBridgeExtras(skillID string, extras map[string]any) map[string]any {
	merged := runBridgeExtras(skillID)
	for key, value := range extras {
		merged[key] = value
	}
	return merged
}

func bridgeRunTextToImagePayload(payload map[string]any) (map[string]any, error) {
	taskPayload := map[string]any{}
	prompt := strings.TrimSpace(firstNonEmptyString(
		stringValue(payload["prompt"]),
		stringValue(payload["scene_prompt"]),
	))
	if prompt == "" {
		return nil, invalidFlagValueError("--input.prompt", "", "text2image skill 需要 prompt")
	}
	putString(taskPayload, "prompt", prompt)
	putString(taskPayload, "style", stringValue(payload["style"]))
	putString(taskPayload, "size", stringValue(payload["size"]))
	putString(taskPayload, "notes", stringValue(payload["notes"]))
	putString(taskPayload, "negative_prompt", stringValue(payload["negative_prompt"]))
	putString(taskPayload, "aspect_ratio", normalizePortableAspectRatio(firstNonEmptyString(
		stringValue(payload["aspect_ratio"]),
		stringValue(payload["ratio"]),
	)))
	putFloat(taskPayload, "seed", numericValue(payload["seed"]))
	return taskPayload, nil
}

func bridgeRunImageToImagePayload(cmd *cobra.Command, payload map[string]any) (map[string]any, map[string]any, error) {
	taskPayload := map[string]any{}
	extras := map[string]any{}
	prompt := strings.TrimSpace(firstNonEmptyString(
		stringValue(payload["prompt"]),
		stringValue(payload["scene_prompt"]),
	))
	if prompt == "" {
		return nil, nil, invalidFlagValueError("--input.prompt", "", "img2img skill 需要 prompt")
	}
	images, preview, err := bridgeRunImageSources(cmd, payload)
	if err != nil {
		return nil, nil, err
	}
	putString(taskPayload, "prompt", prompt)
	putStringSlice(taskPayload, "images", images)
	putString(taskPayload, "style", stringValue(payload["style"]))
	putString(taskPayload, "notes", stringValue(payload["notes"]))
	putString(taskPayload, "negative_prompt", stringValue(payload["negative_prompt"]))
	putString(taskPayload, "aspect_ratio", normalizePortableAspectRatio(firstNonEmptyString(
		stringValue(payload["aspect_ratio"]),
		stringValue(payload["ratio"]),
	)))
	putFloat(taskPayload, "strength", numericValue(payload["strength"]))
	putFloat(taskPayload, "seed", numericValue(payload["seed"]))
	putBool(taskPayload, "preserve_composition", boolValue(payload["preserve_composition"]))
	mergeStringAnyMaps(extras, preview)
	return taskPayload, extras, nil
}

func bridgeRunImageToVideoPayload(cmd *cobra.Command, payload map[string]any) (map[string]any, map[string]any, error) {
	normalized, err := normalizeOfficialImage2VideoDirectInput(payload)
	if err != nil {
		return nil, nil, err
	}
	taskPayload := map[string]any{}
	extras := map[string]any{}
	prompt := strings.TrimSpace(firstNonEmptyString(
		stringValue(normalized["prompt"]),
		stringValue(normalized["motion_prompt"]),
	))
	if prompt == "" {
		return nil, nil, invalidFlagValueError("--input.prompt", "", "image2video skill 需要 prompt")
	}
	images, preview, err := bridgeRunVideoSources(cmd, normalized)
	if err != nil {
		return nil, nil, err
	}
	putString(taskPayload, "prompt", prompt)
	putStringSlice(taskPayload, "images", images)
	putString(taskPayload, "style", stringValue(normalized["style"]))
	putString(taskPayload, "notes", stringValue(normalized["notes"]))
	putString(taskPayload, "negative_prompt", stringValue(normalized["negative_prompt"]))
	putString(taskPayload, "camera_motion", stringValue(normalized["camera_motion"]))
	putString(taskPayload, "motion_intensity", stringValue(normalized["motion_intensity"]))
	putString(taskPayload, "size", firstNonEmptyString(stringValue(normalized["size"]), stringValue(normalized["resolution"])))
	putString(taskPayload, "aspect_ratio", normalizePortableAspectRatio(firstNonEmptyString(
		stringValue(normalized["aspect_ratio"]),
		stringValue(normalized["ratio"]),
	)))
	duration := numericValue(normalized["duration_s"])
	if duration == 0 {
		duration = numericValue(normalized["seconds"])
	}
	putFloat(taskPayload, "duration", duration)
	putFloat(taskPayload, "fps", numericValue(normalized["fps"]))
	putFloat(taskPayload, "seed", numericValue(normalized["seed"]))
	mergeStringAnyMaps(extras, preview)
	return taskPayload, extras, nil
}

func bridgeRunTTSPayload(payload map[string]any) (map[string]any, error) {
	taskPayload := map[string]any{}
	text := strings.TrimSpace(stringValue(payload["text"]))
	if text == "" {
		return nil, invalidFlagValueError("--input.text", "", "tts skill 需要 text")
	}
	putString(taskPayload, "text", text)
	putString(taskPayload, "voice", stringValue(payload["voice"]))
	putString(taskPayload, "language", stringValue(payload["language"]))
	putString(taskPayload, "voice_style", stringValue(payload["voice_style"]))
	putString(taskPayload, "emotion", stringValue(payload["emotion"]))
	putString(taskPayload, "format", stringValue(payload["format"]))
	putString(taskPayload, "sound_effect", stringValue(payload["sound_effect"]))
	putString(taskPayload, "notes", stringValue(payload["notes"]))
	putFloat(taskPayload, "speed", numericValue(payload["speed"]))
	putFloat(taskPayload, "volume", numericValue(payload["volume"]))
	putFloat(taskPayload, "pitch", numericValue(payload["pitch"]))
	putFloat(taskPayload, "sample_rate_hz", numericValue(payload["sample_rate_hz"]))
	putFloat(taskPayload, "seed", numericValue(payload["seed"]))
	putInt(taskPayload, "bitrate", int(numericValue(payload["bitrate"])))
	putInt(taskPayload, "channels", int(numericValue(payload["channels"])))
	putBool(taskPayload, "subtitles", boolValue(payload["subtitles"]))
	if pronunciation := stringSliceValue(payload["pronunciation"]); len(pronunciation) > 0 {
		taskPayload["pronunciation"] = pronunciation
	}
	return taskPayload, nil
}

func bridgeRunImageSources(cmd *cobra.Command, payload map[string]any) ([]string, map[string]any, error) {
	normalized := normalizeOfficialImage2ImageDirectInput(payload)
	urlInputs := []string{}
	if source := strings.TrimSpace(stringValue(normalized["image"])); source != "" {
		urlInputs = append(urlInputs, source)
	}
	urlInputs = append(urlInputs, stringSliceValue(normalized["images"])...)
	artifactIDs := append([]string{}, stringSliceValue(normalized["reference_artifact_ids"])...)
	if sourceArtifactID := strings.TrimSpace(stringValue(normalized["source_artifact_id"])); sourceArtifactID != "" {
		artifactIDs = append([]string{sourceArtifactID}, artifactIDs...)
	}
	return bridgeRunMixedSources(cmd, urlInputs, artifactIDs, "reference")
}

func bridgeRunVideoSources(cmd *cobra.Command, payload map[string]any) ([]string, map[string]any, error) {
	urlInputs := []string{}
	if source := strings.TrimSpace(firstNonEmptyString(stringValue(payload["image_url"]), stringValue(payload["reference_image_url"]))); source != "" {
		urlInputs = append(urlInputs, source)
	}
	urlInputs = append(urlInputs, stringSliceValue(payload["images"])...)
	artifactIDs := []string{}
	if sourceArtifactID := strings.TrimSpace(stringValue(payload["source_artifact_id"])); sourceArtifactID != "" {
		artifactIDs = append(artifactIDs, sourceArtifactID)
	}
	return bridgeRunMixedSources(cmd, urlInputs, artifactIDs, "source")
}

func bridgeRunMixedSources(cmd *cobra.Command, urlInputs, artifactIDs []string, role string) ([]string, map[string]any, error) {
	urls := []string{}
	preview := map[string]any{}
	if len(artifactIDs) > 0 {
		artifactURLs, artifactPreview, err := resolveTaskArtifactURLs(cmd, artifactIDs, role)
		if err != nil {
			return nil, nil, err
		}
		urls = append(urls, artifactURLs...)
		mergeStringAnyMaps(preview, artifactPreview)
	}
	if len(urlInputs) > 0 {
		mediaURLs, mediaPreview, err := resolveTaskMediaURLs(cmd, urlInputs, role, true)
		if err != nil {
			return nil, nil, err
		}
		urls = append(urls, mediaURLs...)
		mergeStringAnyMaps(preview, mediaPreview)
	}
	if len(urls) == 0 {
		return nil, nil, output.NewError("VALIDATION_ERROR", "缺少可用媒体输入", map[string]any{
			"role": role,
		})
	}
	return urls, preview, nil
}

func boolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "on":
			return true
		}
	case float64:
		return typed != 0
	case int:
		return typed != 0
	}
	return false
}
