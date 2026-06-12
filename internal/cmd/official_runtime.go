package cmd

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/poll"
	"github.com/wtgoku-create/popiartcli/internal/seed"
	"github.com/wtgoku-create/popiartcli/internal/types"
)

const (
	officialText2ImageSkillID          = "popiskill-image-text2image-basic-v1"
	officialImage2ImageSkillID         = "popiskill-image-img2img-basic-v1"
	officialAliceImageShowcaseSkillID  = "popiskill-image-img2img-popistudio-alice-showcase-v1"
	officialImage2VideoSkillID         = "popiskill-video-image2video-basic-v1"
	officialAliceVideoShowcaseSkillID  = "popiskill-video-image2video-popistudio-alice-showcase-v1"
	officialTTSMultimodelSkillID       = "popiskill-audio-tts-multimodel-v1"
	officialSTTLocalSkillID            = "popiskill-audio-stt-local-v1"

	officialPlaceholderSnippet = "reserved image2video test skill"
	officialNotConnectedText   = "runtime is not connected yet"
)

var officialRuntimeSkillIDs = []string{
	officialText2ImageSkillID,
	officialImage2ImageSkillID,
	officialAliceImageShowcaseSkillID,
	officialImage2VideoSkillID,
	officialAliceVideoShowcaseSkillID,
	officialTTSMultimodelSkillID,
	officialSTTLocalSkillID,
}

type officialRuntimeContract struct {
	Name        string
	Description string
}

var officialRuntimeContracts = map[string]officialRuntimeContract{
	officialText2ImageSkillID: {
		Name: "Basic Text2Image",
	},
	officialImage2ImageSkillID: {
		Name: "Basic Img2Img",
	},
	officialAliceImageShowcaseSkillID: {
		Name: "Alice Image Showcase",
	},
	officialImage2VideoSkillID: {
		Name:        "Basic Image2Video",
		Description: "Built-in PopiArt image2video baseline. It accepts a source artifact or image URL and bridges execution to the main-site task pipeline.",
	},
	officialAliceVideoShowcaseSkillID: {
		Name: "Alice Video Showcase",
	},
	officialTTSMultimodelSkillID: {
		Name: "TTS Multimodel",
	},
	officialSTTLocalSkillID: {
		Name: "STT Local",
	},
}

func officialRuntimeSkillForID(skillID string) (types.Skill, bool) {
	contract, ok := officialRuntimeContracts[strings.TrimSpace(skillID)]
	if !ok {
		return types.Skill{}, false
	}
	skill, ok := seed.FindBundledSkill(skillID)
	if !ok {
		return types.Skill{}, false
	}
	skill.Name = contract.Name
	skill.Source = "official-runtime"
	if strings.TrimSpace(contract.Description) != "" {
		skill.Description = contract.Description
	}
	return cloneOfficialRuntimeSkill(skill), true
}

func officialRuntimeSkillSummaryForID(skillID string) (types.SkillSummary, bool) {
	skill, ok := officialRuntimeSkillForID(skillID)
	if !ok {
		return types.SkillSummary{}, false
	}
	return types.SkillSummary{
		ID:                 skill.ID,
		Name:               skill.Name,
		Description:        skill.Description,
		Tags:               append([]string(nil), skill.Tags...),
		Version:            skill.Version,
		ModelType:          skill.ModelType,
		EstimatedDurationS: skill.EstimatedDurationS,
		Source:             skill.Source,
	}, true
}

func officialRuntimeSkillSchemaForID(skillID string) (types.SkillSchemaResponse, bool) {
	schema, ok := seed.FindBundledSkillSchema(skillID)
	if !ok {
		return types.SkillSchemaResponse{}, false
	}
	return schema, true
}

func matchingOfficialRuntimeSkillSummaries(tag, search string) []types.SkillSummary {
	items := make([]types.SkillSummary, 0, len(officialRuntimeSkillIDs))
	for _, skillID := range officialRuntimeSkillIDs {
		summary, ok := officialRuntimeSkillSummaryForID(skillID)
		if !ok {
			continue
		}
		if officialRuntimeSummaryMatches(summary, tag, search) {
			items = append(items, summary)
		}
	}
	return items
}

func applyOfficialRuntimeSkillSummaryOverlay(item types.SkillSummary) types.SkillSummary {
	overlay, ok := officialRuntimeSkillSummaryForID(item.ID)
	if !ok || !isOfficialRuntimePlaceholderDescription(item.Description) {
		return item
	}
	item.Name = overlay.Name
	item.Description = overlay.Description
	if len(item.Tags) == 0 {
		item.Tags = overlay.Tags
	}
	if strings.TrimSpace(item.Version) == "" {
		item.Version = overlay.Version
	}
	if strings.TrimSpace(item.ModelType) == "" {
		item.ModelType = overlay.ModelType
	}
	if item.EstimatedDurationS <= 0 {
		item.EstimatedDurationS = overlay.EstimatedDurationS
	}
	return item
}

func applyOfficialRuntimeSkillOverlay(skill types.Skill) types.Skill {
	overlay, ok := officialRuntimeSkillForID(skill.ID)
	if !ok || !isOfficialRuntimePlaceholderSkill(skill) {
		return skill
	}
	skill.Name = overlay.Name
	skill.Description = overlay.Description
	if len(skill.Tags) == 0 {
		skill.Tags = overlay.Tags
	}
	if strings.TrimSpace(skill.Version) == "" {
		skill.Version = overlay.Version
	}
	if strings.TrimSpace(skill.ModelType) == "" {
		skill.ModelType = overlay.ModelType
	}
	if skill.EstimatedDurationS <= 0 {
		skill.EstimatedDurationS = overlay.EstimatedDurationS
	}
	skill.InputSchema = cloneMapAny(overlay.InputSchema)
	skill.OutputSchema = cloneMapAny(overlay.OutputSchema)
	return skill
}

func applyOfficialRuntimeSchemaOverlay(skillID string, schema types.SkillSchemaResponse) types.SkillSchemaResponse {
	if officialRuntimeSchemaLooksUsable(schema) {
		return schema
	}
	if overlay, ok := officialRuntimeSkillSchemaForID(skillID); ok {
		return overlay
	}
	return schema
}

func isOfficialRuntimePlaceholderSkill(skill types.Skill) bool {
	if _, ok := officialRuntimeContracts[strings.TrimSpace(skill.ID)]; !ok {
		return false
	}
	return isOfficialRuntimePlaceholderDescription(skill.Description) || !officialRuntimeSchemaLooksUsable(types.SkillSchemaResponse{
		InputSchema:  skill.InputSchema,
		OutputSchema: skill.OutputSchema,
	})
}

func isOfficialRuntimePlaceholderDescription(description string) bool {
	normalized := strings.ToLower(strings.TrimSpace(description))
	if normalized == "" {
		return true
	}
	return strings.Contains(normalized, officialPlaceholderSnippet) || strings.Contains(normalized, officialNotConnectedText)
}

func officialRuntimePlaceholderHint(skillID string) string {
	switch strings.TrimSpace(skillID) {
	case officialImage2VideoSkillID:
		return "当前 CLI 会对 image2video 自动桥接到主站任务链路"
	default:
		return "当前 skill 需要由 popiartServer 完成正式注册与执行路由"
	}
}

func officialRuntimeSchemaLooksUsable(schema types.SkillSchemaResponse) bool {
	return len(schema.InputSchema) > 0 || len(schema.OutputSchema) > 0
}

func officialRuntimeSummaryMatches(summary types.SkillSummary, tag, search string) bool {
	tagNeedle := strings.ToLower(strings.TrimSpace(tag))
	if tagNeedle != "" {
		matched := false
		for _, value := range summary.Tags {
			if strings.EqualFold(value, tagNeedle) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	searchNeedle := strings.ToLower(strings.TrimSpace(search))
	if searchNeedle == "" {
		return true
	}
	fields := []string{summary.ID, summary.Name, summary.Description, summary.ModelType}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), searchNeedle) {
			return true
		}
	}
	for _, value := range summary.Tags {
		if strings.Contains(strings.ToLower(value), searchNeedle) {
			return true
		}
	}
	return false
}

func normalizeOfficialRuntimeDirectInput(skillID string, payload map[string]any) (map[string]any, error) {
	switch strings.TrimSpace(skillID) {
	case officialImage2ImageSkillID:
		return normalizeOfficialImage2ImageDirectInput(payload), nil
	case officialImage2VideoSkillID:
		return normalizeOfficialImage2VideoDirectInput(payload)
	default:
		return cloneMapAny(payload), nil
	}
}

func normalizeOfficialImage2ImageDirectInput(payload map[string]any) map[string]any {
	input := cloneMapAny(payload)
	if input == nil {
		input = map[string]any{}
	}

	if stringValue(input["image"]) == "" {
		if alias := stringValue(input["image_url"]); alias != "" {
			input["image"] = alias
		} else if alias := stringValue(input["reference_image_url"]); alias != "" {
			input["image"] = alias
		}
	}
	delete(input, "image_url")
	delete(input, "reference_image_url")
	return input
}

func normalizeOfficialImage2VideoDirectInput(payload map[string]any) (map[string]any, error) {
	input := cloneMapAny(payload)
	if input == nil {
		input = map[string]any{}
	}

	if stringValue(input["image_url"]) == "" {
		if alias := stringValue(input["reference_image_url"]); alias != "" {
			input["image_url"] = alias
		}
	}
	if _, exists := input["duration_s"]; !exists {
		if seconds, ok := input["seconds"]; ok {
			input["duration_s"] = seconds
		}
	}

	if stringValue(input["source_artifact_id"]) == "" && stringValue(input["image_url"]) == "" {
		if stringValue(input["prompt"]) != "" {
			return input, nil
		}
		return nil, output.NewError("VALIDATION_ERROR", "image2video 需要 source_artifact_id 或 image_url", map[string]any{
			"skill_id": officialImage2VideoSkillID,
			"hint":     "先上传图片得到 artifact_id，或直接传 image_url / reference_image_url",
		})
	}
	return input, nil
}

func submitModelInferJob(ctx context.Context, modelID string, payload map[string]any, priority, projectID, idempotencyKey string) (map[string]any, error) {
	return submitModelInferJobWithModelType(ctx, modelID, payload, priority, projectID, idempotencyKey, "")
}

func submitModelInferJobWithModelType(ctx context.Context, modelID string, payload map[string]any, priority, projectID, idempotencyKey, modelType string) (map[string]any, error) {
	cfg := config.Load()
	body := map[string]any{
		"model_id": modelID,
		"input":    payload,
		"priority": defaultString(strings.TrimSpace(priority), "normal"),
	}
	if strings.TrimSpace(modelType) != "" {
		body["model_type"] = strings.TrimSpace(modelType)
	}
	if cfg.Project != "" {
		body["project_id"] = cfg.Project
	}
	if strings.TrimSpace(projectID) != "" {
		body["project_id"] = strings.TrimSpace(projectID)
	}
	if strings.TrimSpace(idempotencyKey) != "" {
		body["idempotency_key"] = strings.TrimSpace(idempotencyKey)
	}

	var job map[string]any
	if err := currentClient().PostJSON(ctx, "/models/infer", body, &job); err != nil {
		return nil, err
	}
	return job, nil
}

func writeJobResultOrWait(cmd *cobra.Command, job map[string]any) error {
	wait, err := shouldWaitForJob(cmd)
	if err != nil {
		return err
	}
	if !wait {
		return writeOutput(cmd, job)
	}

	jobID := stringValue(job["job_id"])
	if jobID == "" {
		return output.NewError("CLI_ERROR", "作业响应中缺少 job_id", nil)
	}

	interval, err := intervalDuration(cmd, "interval")
	if err != nil {
		return err
	}
	done, err := poll.WaitForJob(context.Background(), currentClient(), jobID, interval, 300)
	if err != nil {
		return err
	}
	return writeOutput(cmd, done)
}

func writeTypedJobResultOrWait(cmd *cobra.Command, job types.Job) error {
	wait, err := shouldWaitForJob(cmd)
	if err != nil {
		return err
	}
	if !wait {
		return writeOutput(cmd, job)
	}
	if job.JobID == "" {
		return output.NewError("CLI_ERROR", "作业响应中缺少 job_id", nil)
	}

	interval, err := intervalDuration(cmd, "interval")
	if err != nil {
		return err
	}
	done, err := poll.WaitForJob(context.Background(), currentClient(), job.JobID, interval, 300)
	if err != nil {
		return err
	}
	return writeOutput(cmd, done)
}

func cloneOfficialRuntimeSkill(skill types.Skill) types.Skill {
	skill.Tags = append([]string(nil), skill.Tags...)
	skill.InputSchema = cloneMapAny(skill.InputSchema)
	skill.OutputSchema = cloneMapAny(skill.OutputSchema)
	return skill
}

func cloneMapAny(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = cloneValueAny(value)
	}
	return dst
}

func cloneValueAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMapAny(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, cloneValueAny(item))
		}
		return out
	case []string:
		return append([]string(nil), typed...)
	default:
		return typed
	}
}

func numericValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case int32:
		return float64(typed)
	default:
		return 0
	}
}
