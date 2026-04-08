package cmd

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/poll"
	"github.com/wtgoku-create/popiartcli/internal/types"
)

const (
	officialImage2VideoSkillID         = "popiskill-video-image2video-basic-v1"
	officialImage2VideoPrimaryModelID  = "viduq3-turbo"
	officialImage2VideoFallbackModelID = "viduq2-pro-fast"

	officialPlaceholderSnippet = "reserved image2video test skill"
	officialNotConnectedText   = "runtime is not connected yet"
)

type officialRuntimeDirectInfer struct {
	ModelIDs []string
}

type officialRuntimeContract struct {
	Skill       types.Skill
	DirectInfer *officialRuntimeDirectInfer
}

var officialRuntimeContracts = map[string]officialRuntimeContract{
	officialImage2VideoSkillID: {
		Skill: types.Skill{
			ID:          officialImage2VideoSkillID,
			Name:        "Basic Image2Video",
			Description: "Built-in PopiArt image2video skill. It accepts a source artifact or image URL and, when the remote catalog entry is still a placeholder, the CLI bridges execution to direct models infer with viduq3-turbo and falls back to viduq2-pro-fast.",
			Tags:        []string{"official", "builtin", "runtime", "video", "image2video"},
			Version:     "v1",
			ModelType:   "video",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"source_artifact_id": map[string]any{
						"type":        "string",
						"description": "Preferred PopiArt artifact ID for the source image.",
					},
					"image_url": map[string]any{
						"type":        "string",
						"description": "Fallback source image URL when no source_artifact_id exists.",
					},
					"reference_image_url": map[string]any{
						"type":        "string",
						"description": "Compatibility alias for image_url.",
					},
					"prompt":           map[string]any{"type": "string"},
					"negative_prompt":  map[string]any{"type": "string"},
					"duration_s":       map[string]any{"type": "number"},
					"seconds":          map[string]any{"type": "number"},
					"fps":              map[string]any{"type": "number"},
					"camera_motion":    map[string]any{"type": "string"},
					"motion_intensity": map[string]any{"type": "string"},
					"loop":             map[string]any{"type": "boolean"},
					"style":            map[string]any{"type": "string"},
					"aspect_ratio":     map[string]any{"type": "string"},
					"seed":             map[string]any{"type": "number"},
					"notes":            map[string]any{"type": "string"},
				},
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"job_id":             map[string]any{"type": "string"},
					"status":             map[string]any{"type": "string"},
					"model_id":           map[string]any{"type": "string"},
					"requested_skill_id": map[string]any{"type": "string"},
					"execution_mode":     map[string]any{"type": "string"},
					"artifact_ids": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
				},
			},
			EstimatedDurationS: 180,
		},
		DirectInfer: &officialRuntimeDirectInfer{
			ModelIDs: []string{
				officialImage2VideoPrimaryModelID,
				officialImage2VideoFallbackModelID,
			},
		},
	},
}

func officialRuntimeSkillForID(skillID string) (types.Skill, bool) {
	contract, ok := officialRuntimeContracts[strings.TrimSpace(skillID)]
	if !ok {
		return types.Skill{}, false
	}
	return cloneOfficialRuntimeSkill(contract.Skill), true
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
	}, true
}

func officialRuntimeSkillSchemaForID(skillID string) (types.SkillSchemaResponse, bool) {
	skill, ok := officialRuntimeSkillForID(skillID)
	if !ok {
		return types.SkillSchemaResponse{}, false
	}
	return types.SkillSchemaResponse{
		InputSchema:  cloneMapAny(skill.InputSchema),
		OutputSchema: cloneMapAny(skill.OutputSchema),
	}, true
}

func matchingOfficialRuntimeSkillSummaries(tag, search string) []types.SkillSummary {
	items := make([]types.SkillSummary, 0, len(officialRuntimeContracts))
	for skillID := range officialRuntimeContracts {
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

func maybeRunOfficialRuntimeDirectFallbackJob(ctx context.Context, skillID string, payload any, priority, projectID, idempotencyKey string) (map[string]any, bool, error) {
	contract, ok := officialRuntimeContracts[strings.TrimSpace(skillID)]
	if !ok || contract.DirectInfer == nil {
		return nil, false, nil
	}

	shouldFallback, err := officialRuntimeSkillNeedsDirectInferFallback(ctx, skillID)
	if err != nil {
		return nil, false, err
	}
	if !shouldFallback {
		return nil, false, nil
	}

	payloadMap, ok := payload.(map[string]any)
	if !ok {
		return nil, true, output.NewError("VALIDATION_ERROR", "image2video 需要 JSON object 输入", map[string]any{
			"skill_id": skillID,
		})
	}

	input, err := normalizeOfficialRuntimeDirectInput(skillID, payloadMap)
	if err != nil {
		return nil, true, err
	}

	modelIDs := preferredOfficialRuntimeModelIDs(skillID, input, contract.DirectInfer.ModelIDs)
	job, modelID, err := submitModelInferWithFallback(ctx, modelIDs, input, priority, projectID, idempotencyKey)
	if err != nil {
		return nil, true, err
	}
	job["requested_skill_id"] = skillID
	job["model_id"] = modelID
	job["execution_mode"] = "direct-model-fallback"
	return job, true, nil
}

func officialRuntimeSkillNeedsDirectInferFallback(ctx context.Context, skillID string) (bool, error) {
	contract, ok := officialRuntimeContracts[strings.TrimSpace(skillID)]
	if !ok || contract.DirectInfer == nil {
		return false, nil
	}

	var skill types.Skill
	if err := currentClient().GetJSON(ctx, "/skills/"+skillID, nil, &skill); err != nil {
		if cliErr, ok := err.(*output.CLIError); ok && cliErr.Code == "NOT_FOUND" {
			return true, nil
		}
		return false, err
	}
	return isOfficialRuntimePlaceholderSkill(skill), nil
}

func normalizeOfficialRuntimeDirectInput(skillID string, payload map[string]any) (map[string]any, error) {
	switch strings.TrimSpace(skillID) {
	case officialImage2VideoSkillID:
		return normalizeOfficialImage2VideoDirectInput(payload)
	default:
		return cloneMapAny(payload), nil
	}
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
		return nil, output.NewError("VALIDATION_ERROR", "image2video 需要 source_artifact_id 或 image_url", map[string]any{
			"skill_id": officialImage2VideoSkillID,
			"hint":     "先上传图片得到 artifact_id，或直接传 image_url / reference_image_url",
		})
	}
	return input, nil
}

func preferredOfficialRuntimeModelIDs(skillID string, input map[string]any, fallback []string) []string {
	switch strings.TrimSpace(skillID) {
	case officialImage2VideoSkillID:
		return preferredImage2VideoModelIDs(input, fallback)
	default:
		return append([]string(nil), fallback...)
	}
}

func preferredImage2VideoModelIDs(input map[string]any, fallback []string) []string {
	duration := numericValue(input["duration_s"])
	if duration == 0 {
		duration = numericValue(input["seconds"])
	}
	if duration != 0 && duration != 5 && duration != 10 {
		return []string{officialImage2VideoFallbackModelID}
	}
	return append([]string(nil), fallback...)
}

func submitModelInferWithFallback(ctx context.Context, modelIDs []string, payload map[string]any, priority, projectID, idempotencyKey string) (map[string]any, string, error) {
	var lastErr error
	for idx, modelID := range modelIDs {
		job, err := submitModelInferJob(ctx, modelID, payload, priority, projectID, idempotencyKey)
		if err == nil {
			return job, modelID, nil
		}
		lastErr = err
		if idx == len(modelIDs)-1 || !shouldRetryModelInfer(err) {
			break
		}
	}
	if cliErr, ok := lastErr.(*output.CLIError); ok {
		details := cloneMapAny(cliErr.Details)
		details["models_tried"] = append([]string(nil), modelIDs...)
		return nil, "", output.NewError(cliErr.Code, cliErr.Message, details)
	}
	return nil, "", lastErr
}

func submitModelInferJob(ctx context.Context, modelID string, payload map[string]any, priority, projectID, idempotencyKey string) (map[string]any, error) {
	cfg := config.Load()
	body := map[string]any{
		"model_id": modelID,
		"input":    payload,
		"priority": defaultString(strings.TrimSpace(priority), "normal"),
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

func shouldRetryModelInfer(err error) bool {
	cliErr, ok := err.(*output.CLIError)
	if !ok {
		return false
	}
	switch cliErr.Code {
	case "NETWORK_ERROR", "UNAUTHENTICATED", "FORBIDDEN":
		return false
	default:
		return true
	}
}

func writeJobResultOrWait(cmd *cobra.Command, job map[string]any) error {
	if !flagBool(cmd, "wait") {
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
