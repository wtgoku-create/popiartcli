package popiart

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/wtgoku-create/popiartcli/internal/api"
	"github.com/wtgoku-create/popiartcli/internal/output"
)

// FetchModels 读取主站模型列表，作为运行期能力源。
func FetchModels(ctx context.Context, client *api.Client) ([]Model, error) {
	var models []Model
	if err := client.GetJSON(ctx, "/api_client/anime/ai/model/list", map[string]string{
		"origin": "web",
	}, &models); err != nil {
		return nil, NormalizeAPIError(err)
	}
	return models, nil
}

// ResolveModelByCode 按 code 或 alias 从模型列表中查找候选模型。
func ResolveModelByCode(models []Model, code string) (Model, bool) {
	code = strings.TrimSpace(code)
	if code == "" {
		return Model{}, false
	}

	for _, model := range models {
		if strings.EqualFold(strings.TrimSpace(model.Code), code) {
			return model, true
		}
		for _, alias := range model.AIModelCodeAlias {
			if strings.EqualFold(strings.TrimSpace(alias), code) {
				return model, true
			}
		}
	}
	return Model{}, false
}

// ResolveModelByCodeAndSubtype 在重复 code/alias 的情况下优先返回真正支持目标子类型的模型。
func ResolveModelByCodeAndSubtype(models []Model, code string, subType int) (Model, bool) {
	code = strings.TrimSpace(code)
	if code == "" {
		return Model{}, false
	}

	var fallback Model
	var hasFallback bool
	for _, model := range models {
		if !matchesModelCode(model, code) {
			continue
		}
		if !hasFallback {
			fallback = model
			hasFallback = true
		}
		if subType == 0 || SupportsSubType(model, subType) {
			return model, true
		}
	}
	return fallback, hasFallback
}

// ResolveCandidateModel 根据显式模型或默认候选池解析最终模型。
func ResolveCandidateModel(models []Model, requestedCode string, defaults []string, subType int) (Model, error) {
	if strings.TrimSpace(requestedCode) != "" {
		model, ok := ResolveModelByCodeAndSubtype(models, requestedCode, subType)
		if !ok {
			return Model{}, output.NewError("MODEL_NOT_FOUND", "未找到匹配模型", map[string]any{
				"model": requestedCode,
			})
		}
		if subType != 0 && !SupportsSubType(model, subType) {
			return Model{}, output.NewError("MODEL_SUBTYPE_UNSUPPORTED", "模型不支持当前任务子类型", map[string]any{
				"model":    requestedCode,
				"sub_type": subType,
			})
		}
		return model, nil
	}

	for _, candidate := range defaults {
		model, ok := ResolveModelByCodeAndSubtype(models, candidate, subType)
		if !ok {
			continue
		}
		if subType != 0 && !SupportsSubType(model, subType) {
			continue
		}
		return model, nil
	}

	return Model{}, output.NewError("MODEL_NOT_FOUND", "未找到可用默认模型", map[string]any{
		"sub_type": subType,
	})
}

// SupportsSubType 判断模型是否声明支持目标任务子类型。
func SupportsSubType(model Model, subType int) bool {
	if subType == 0 {
		return true
	}
	for _, category := range model.Categories {
		if category.Status != nil && *category.Status < 0 {
			continue
		}
		if category.TaskSubType == subType {
			return true
		}
		if subType == 301 && (category.TaskSubType == 304 || category.TaskSubType == 305) {
			return true
		}
	}
	return false
}

func matchesModelCode(model Model, code string) bool {
	if strings.EqualFold(strings.TrimSpace(model.Code), code) {
		return true
	}
	for _, alias := range model.AIModelCodeAlias {
		if strings.EqualFold(strings.TrimSpace(alias), code) {
			return true
		}
	}
	return false
}

// ValidateModelSupport 按迁移文档中的统一顺序校验模型能力约束。
func ValidateModelSupport(model Model, spec ModelValidationSpec) error {
	if len(spec.AllowedSubTypes) > 0 {
		matched := false
		for _, subType := range spec.AllowedSubTypes {
			if SupportsSubType(model, subType) {
				matched = true
				break
			}
		}
		if !matched {
			return output.NewError("MODEL_SUBTYPE_UNSUPPORTED", "模型不支持当前任务子类型", map[string]any{
				"model":             model.Code,
				"allowed_sub_types": spec.AllowedSubTypes,
			})
		}
	}
	if spec.SubType != 0 && !SupportsSubType(model, spec.SubType) {
		return output.NewError("MODEL_SUBTYPE_UNSUPPORTED", "模型不支持当前任务子类型", map[string]any{
			"model":    model.Code,
			"sub_type": spec.SubType,
		})
	}
	if spec.RequiresImages && !model.IsSupportImages {
		return output.NewError("VALIDATION_ERROR", "模型不支持图片输入", map[string]any{"model": model.Code})
	}
	if spec.RequiresVideos && !model.IsSupportVideos {
		return output.NewError("VALIDATION_ERROR", "模型不支持视频输入", map[string]any{"model": model.Code})
	}
	if spec.RequiresAudios && !model.IsSupportAudios {
		return output.NewError("VALIDATION_ERROR", "模型不支持音频输入", map[string]any{"model": model.Code})
	}
	if model.UploadImageLimit.Value != nil && *model.UploadImageLimit.Value > 0 && spec.ImageCount > *model.UploadImageLimit.Value {
		return output.NewError("VALIDATION_ERROR", "参考图数量超过模型限制", map[string]any{
			"model":              model.Code,
			"image_count":        spec.ImageCount,
			"upload_image_limit": *model.UploadImageLimit.Value,
		})
	}
	if model.UploadVideoLimit.Value != nil && *model.UploadVideoLimit.Value > 0 && spec.VideoCount > *model.UploadVideoLimit.Value {
		return output.NewError("VALIDATION_ERROR", "参考视频数量超过模型限制", map[string]any{
			"model":              model.Code,
			"video_count":        spec.VideoCount,
			"upload_video_limit": *model.UploadVideoLimit.Value,
		})
	}
	if model.UploadAudioLimit.Value != nil && *model.UploadAudioLimit.Value > 0 && spec.AudioCount > *model.UploadAudioLimit.Value {
		return output.NewError("VALIDATION_ERROR", "参考音频数量超过模型限制", map[string]any{
			"model":              model.Code,
			"audio_count":        spec.AudioCount,
			"upload_audio_limit": *model.UploadAudioLimit.Value,
		})
	}
	if spec.Ratio != "" && len(model.Ratio) > 0 && !containsString(model.Ratio, spec.Ratio) {
		return output.NewError("VALIDATION_ERROR", "模型不支持当前图片比例", map[string]any{
			"model": model.Code,
			"ratio": spec.Ratio,
		})
	}
	if spec.VideoRatio != "" && len(model.VideoRatio) > 0 && !containsString(model.VideoRatio, spec.VideoRatio) {
		return output.NewError("VALIDATION_ERROR", "模型不支持当前视频比例", map[string]any{
			"model":       model.Code,
			"video_ratio": spec.VideoRatio,
		})
	}
	if spec.Resolution != "" && len(model.Resolution) > 0 && !containsString(model.Resolution, spec.Resolution) {
		return output.NewError("VALIDATION_ERROR", "模型不支持当前分辨率", map[string]any{
			"model":      model.Code,
			"resolution": spec.Resolution,
		})
	}
	if spec.Duration != 0 && len(model.Duration) > 0 && !containsInt(model.Duration, spec.Duration) {
		return output.NewError("VALIDATION_ERROR", "模型不支持当前时长", map[string]any{
			"model":    model.Code,
			"duration": spec.Duration,
		})
	}
	return nil
}

// PreferredSubType 从允许集合中选择当前模型支持的最佳任务子类型。
func PreferredSubType(model Model, allowed []int) int {
	for _, subType := range allowed {
		if SupportsSubType(model, subType) {
			return subType
		}
	}
	return 0
}

// SupportedSubTypes 返回模型声明支持的有效 taskSubType，按数值升序输出。
func SupportedSubTypes(model Model) []int {
	seen := map[int]struct{}{}
	values := make([]int, 0, len(model.Categories))
	for _, category := range model.Categories {
		if category.TaskSubType == 0 {
			continue
		}
		if category.Status != nil && *category.Status < 0 {
			continue
		}
		if _, ok := seen[category.TaskSubType]; ok {
			continue
		}
		seen[category.TaskSubType] = struct{}{}
		values = append(values, category.TaskSubType)
	}
	sort.Ints(values)
	return values
}

// ResolveModelForCommand 组合执行模型拉取、候选解析和能力校验。
func ResolveModelForCommand(ctx context.Context, client *api.Client, command, requestedCode string, spec ModelValidationSpec) (Model, error) {
	models, err := FetchModels(ctx, client)
	if err != nil {
		return Model{}, err
	}

	model, err := ResolveCandidateModel(models, requestedCode, DefaultModelCodes(command), spec.SubType)
	if err != nil {
		return Model{}, err
	}
	if err := ValidateModelSupport(model, spec); err != nil {
		return Model{}, err
	}
	return model, nil
}

// NormalizePayloadForModel 按模型能力把命令 payload 中的关键候选参数收敛到可用值。
func NormalizePayloadForModel(payload map[string]any, model Model, taskType int, isVideo bool) map[string]any {
	if payload == nil {
		return nil
	}

	normalized := cloneMapAny(payload)

	ratioKey := "aspect_ratio"
	candidates := model.Ratio
	if isVideo {
		candidates = model.VideoRatio
	}
	if requested := strings.TrimSpace(stringValue(normalized[ratioKey])); requested != "" && len(candidates) > 0 && !containsString(candidates, requested) {
		if fallback := firstSupportedAspectRatio(candidates); fallback != "" {
			normalized[ratioKey] = fallback
		}
	}
	if requested := strings.TrimSpace(stringValue(normalized["size"])); requested != "" && len(model.Resolution) > 0 && !containsString(model.Resolution, requested) {
		if fallback := firstSupportedResolution(model.Resolution, taskType); fallback != "" {
			normalized["size"] = fallback
		}
	}
	if isVideo {
		requestedDuration := int(numericValue(normalized["duration"]))
		if requestedDuration == 0 {
			requestedDuration = int(numericValue(normalized["duration_s"]))
		}
		if requestedDuration > 0 && len(model.Duration) > 0 && !containsInt(model.Duration, requestedDuration) {
			if fallback := firstSupportedDuration(model.Duration); fallback > 0 {
				normalized["duration"] = fallback
				normalized["duration_s"] = fallback
			}
		}
	}

	return normalized
}

func firstSupportedAspectRatio(values []string) string {
	for _, value := range values {
		value = normalizeTaskAspectRatio(strings.TrimSpace(value))
		if value != "" {
			return value
		}
	}
	return ""
}

func firstSupportedResolution(values []string, taskType int) string {
	if len(values) == 0 {
		return ""
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = normalizeResolutionLabel(value)
		if value == "" {
			continue
		}
		normalized = append(normalized, value)
	}
	if len(normalized) == 0 {
		return ""
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		return resolutionRank(normalized[i], taskType) < resolutionRank(normalized[j], taskType)
	})
	return normalized[0]
}

func firstSupportedDuration(values []int) int {
	filtered := make([]int, 0, len(values))
	seen := map[int]struct{}{}
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		filtered = append(filtered, value)
	}
	if len(filtered) == 0 {
		return 0
	}
	sort.Ints(filtered)
	return filtered[0]
}

func containsString(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}

func containsInt(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func ModelSummary(model Model) string {
	if strings.TrimSpace(model.Name) != "" {
		return fmt.Sprintf("%s (%s)", model.Name, model.Code)
	}
	return model.Code
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

func stringValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case fmt.Stringer:
		return v.String()
	default:
		return ""
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
