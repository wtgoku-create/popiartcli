package popiart

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// BuildTextToImageTaskRequest 把图片生成 payload 映射为主站 task/create 请求。
func BuildTextToImageTaskRequest(payload map[string]any, model Model) TaskRequest {
	aspectRatio := resolveModelAspectRatio(stringField(payload, "aspect_ratio"), model, false)
	resolution := resolvedTaskResolution(stringField(payload, "size"), model, 1)
	width, height := taskCanvasDimensions(aspectRatio, resolution, 1)
	req := TaskRequest{
		Type:             1,
		SubType:          103,
		ProjectID:        -1,
		Model:            model.Code,
		AIModelCode:      model.Code,
		AIModelCodeAlias: preferredModelAlias(model),
		AIModelName:      preferredModelName(model),
		AIModelID:        model.ID,
		StyleID:          0,
		Width:            width,
		Height:           height,
		ChatPrompt:       stringField(payload, "prompt"),
		AspectRatio:      aspectRatio,
		Ratio:            aspectRatio,
		Resolution:       resolution,
		BatchSize:        1,
		Metadata:         map[string]any{},
	}
	putMetadataString(req.Metadata, "style", payload["style"])
	putMetadataString(req.Metadata, "negative_prompt", payload["negative_prompt"])
	putMetadataString(req.Metadata, "notes", payload["notes"])
	putMetadataFloat(req.Metadata, "seed", payload["seed"])
	if len(req.Metadata) == 0 {
		req.Metadata = nil
	}
	return req
}

// BuildTextToSpeechTaskRequest 把语音命令 payload 映射为主站音频任务请求。
func BuildTextToSpeechTaskRequest(payload map[string]any, model Model) TaskRequest {
	width, height := taskCanvasDimensions("", "", 3)
	req := TaskRequest{
		Type:             3,
		SubType:          301,
		ProjectID:        -1,
		Model:            model.Code,
		AIModelCode:      model.Code,
		AIModelCodeAlias: preferredModelAlias(model),
		AIModelName:      preferredModelName(model),
		AIModelID:        model.ID,
		StyleID:          0,
		Width:            width,
		Height:           height,
		ChatPrompt:       stringField(payload, "text"),
		VoiceID:          stringField(payload, "voice"),
		BatchSize:        1,
		Metadata:         map[string]any{},
	}
	for _, key := range []string{
		"language", "voice_style", "emotion", "format", "sound_effect", "notes",
	} {
		putMetadataString(req.Metadata, key, payload[key])
	}
	for _, key := range []string{"speed", "volume", "pitch", "sample_rate_hz", "seed"} {
		putMetadataFloat(req.Metadata, key, payload[key])
	}
	for _, key := range []string{"bitrate", "channels"} {
		putMetadataInt(req.Metadata, key, payload[key])
	}
	putMetadataBool(req.Metadata, "subtitles", payload["subtitles"])
	if values := stringSliceField(payload["pronunciation"]); len(values) > 0 {
		req.Metadata["pronunciation"] = values
	}
	if len(req.Metadata) == 0 {
		req.Metadata = nil
	}
	return req
}

// BuildMusicTaskRequest 把音乐命令 payload 映射为主站音频任务请求。
func BuildMusicTaskRequest(payload map[string]any, model Model) TaskRequest {
	width, height := taskCanvasDimensions("", "", 3)
	req := TaskRequest{
		Type:             3,
		SubType:          resolveMusicTaskSubType(payload, model),
		ProjectID:        -1,
		Model:            model.Code,
		AIModelCode:      model.Code,
		AIModelCodeAlias: preferredModelAlias(model),
		AIModelName:      preferredModelName(model),
		AIModelID:        model.ID,
		StyleID:          0,
		Width:            width,
		Height:           height,
		ChatPrompt:       buildMusicPrompt(payload),
		BatchSize:        1,
		Metadata:         map[string]any{},
	}
	for _, key := range []string{
		"prompt", "lyrics", "output_format", "audio_url", "audio_base64",
	} {
		putMetadataString(req.Metadata, key, payload[key])
	}
	for _, key := range []string{"lyrics_optimizer", "is_instrumental", "stream", "aigc_watermark"} {
		putMetadataBool(req.Metadata, key, payload[key])
	}
	if audioSetting, ok := payload["audio_setting"].(map[string]any); ok && len(audioSetting) > 0 {
		req.Metadata["audio_setting"] = audioSetting
	}
	if len(req.Metadata) == 0 {
		req.Metadata = nil
	}
	return req
}

// BuildImageTransformTaskRequest 把 img2img/transform payload 映射为主站图片编辑任务请求。
func BuildImageTransformTaskRequest(payload map[string]any, model Model) TaskRequest {
	aspectRatio := resolveModelAspectRatio(stringField(payload, "aspect_ratio"), model, false)
	resolution := resolvedTaskResolution(stringField(payload, "size"), model, 1)
	width, height := taskCanvasDimensions(aspectRatio, resolution, 1)
	req := TaskRequest{
		Type:             1,
		SubType:          103,
		ProjectID:        -1,
		Model:            model.Code,
		AIModelCode:      model.Code,
		AIModelCodeAlias: preferredModelAlias(model),
		AIModelName:      preferredModelName(model),
		AIModelID:        model.ID,
		StyleID:          0,
		Width:            width,
		Height:           height,
		ChatPrompt:       stringField(payload, "prompt"),
		AspectRatio:      aspectRatio,
		Ratio:            aspectRatio,
		Resolution:       resolution,
		BatchSize:        1,
		Metadata:         map[string]any{},
	}
	req.Images = append(req.Images, stringSliceField(payload["images"])...)
	putMetadataString(req.Metadata, "style", payload["style"])
	putMetadataString(req.Metadata, "negative_prompt", payload["negative_prompt"])
	putMetadataString(req.Metadata, "notes", payload["notes"])
	putMetadataFloat(req.Metadata, "strength", payload["strength"])
	putMetadataFloat(req.Metadata, "seed", payload["seed"])
	putMetadataBool(req.Metadata, "preserve_composition", payload["preserve_composition"])
	if len(req.Metadata) == 0 {
		req.Metadata = nil
	}
	return req
}

// BuildVideoGenerateTaskRequest 把通用视频 payload 映射为主站视频任务请求。
func BuildVideoGenerateTaskRequest(payload map[string]any, model Model) TaskRequest {
	subType := intField(payload, "sub_type", 0)
	if subType == 0 {
		subType = resolvedVideoTaskSubType(payload, model)
	}
	aspectRatio := resolveModelAspectRatio(stringField(payload, "aspect_ratio"), model, true)
	resolution := resolvedTaskResolution(stringField(payload, "size"), model, 2)
	width, height := taskCanvasDimensions(aspectRatio, resolution, 2)
	req := TaskRequest{
		Type:             2,
		SubType:          subType,
		ProjectID:        -1,
		Model:            model.Code,
		AIModelCode:      model.Code,
		AIModelCodeAlias: preferredModelAlias(model),
		AIModelName:      preferredModelName(model),
		AIModelID:        model.ID,
		StyleID:          0,
		Width:            width,
		Height:           height,
		ChatPrompt:       stringField(payload, "prompt"),
		AspectRatio:      aspectRatio,
		Ratio:            aspectRatio,
		Resolution:       resolution,
		Duration:         resolvedTaskDuration(intField(payload, "duration", 0), model),
		BatchSize:        1,
		Metadata:         map[string]any{},
	}
	if _, ok := payload["images"]; ok {
		req.Images = stringSliceField(payload["images"])
	}
	if _, ok := payload["videos"]; ok {
		req.Videos = stringSliceField(payload["videos"])
	}
	if _, ok := payload["audios"]; ok {
		req.Voices = stringSliceField(payload["audios"])
	}
	putMetadataString(req.Metadata, "camera_motion", payload["camera_motion"])
	putMetadataString(req.Metadata, "movement_amplitude", payload["motion_intensity"])
	putMetadataString(req.Metadata, "negative_prompt", payload["negative_prompt"])
	putMetadataString(req.Metadata, "style", payload["style"])
	putMetadataString(req.Metadata, "notes", payload["notes"])
	putMetadataFloat(req.Metadata, "fps", payload["fps"])
	putMetadataFloat(req.Metadata, "seed", payload["seed"])
	if len(req.Metadata) == 0 {
		req.Metadata = nil
	}
	return req
}

// BuildVideoActionTransferTaskRequest 把动作迁移 payload 映射为主站视频任务请求。
func BuildVideoActionTransferTaskRequest(payload map[string]any, model Model) TaskRequest {
	resolution := resolvedTaskResolution(stringField(payload, "size"), model, 2)
	width, height := taskCanvasDimensions("", resolution, 2)
	req := TaskRequest{
		Type:             2,
		SubType:          205,
		ProjectID:        -1,
		Model:            model.Code,
		AIModelCode:      model.Code,
		AIModelCodeAlias: preferredModelAlias(model),
		AIModelName:      preferredModelName(model),
		AIModelID:        model.ID,
		StyleID:          0,
		Width:            width,
		Height:           height,
		ChatPrompt:       stringField(payload, "prompt"),
		Resolution:       resolution,
		BatchSize:        1,
		Metadata:         map[string]any{},
	}
	req.Images = append(req.Images, stringSliceField(payload["images"])...)
	req.Videos = append(req.Videos, stringSliceField(payload["videos"])...)
	putMetadataString(req.Metadata, "notes", payload["notes"])
	if metadata, ok := payload["metadata"].(map[string]any); ok {
		for key, value := range metadata {
			req.Metadata[key] = value
		}
	}
	if len(req.Metadata) == 0 {
		req.Metadata = nil
	}
	return req
}

// TaskOutput 把任务结果转换为 CLI 兼容输出字段。
func TaskOutput(task TaskDetail, model Model, extras map[string]any) map[string]any {
	taskID := task.Identifier()
	result := map[string]any{
		"job_id":   taskID,
		"task_id":  taskID,
		"status":   int(task.Status),
		"model":    model.ID,
		"type":     int(task.Type),
		"sub_type": int(task.SubType),
	}
	if len(task.DownloadURLs) > 0 {
		result["download_urls"] = append([]string(nil), task.DownloadURLs...)
	}
	if len(task.ResultURLs) > 0 {
		result["result_urls"] = append([]string(nil), task.ResultURLs...)
	}
	for key, value := range extras {
		result[key] = value
	}
	return result
}

func buildMusicPrompt(payload map[string]any) string {
	prompt := strings.TrimSpace(stringField(payload, "prompt"))
	lyrics := strings.TrimSpace(stringField(payload, "lyrics"))
	switch {
	case prompt != "" && lyrics != "":
		return fmt.Sprintf("%s\n\nlyrics:\n%s", prompt, lyrics)
	case prompt != "":
		return prompt
	default:
		return lyrics
	}
}

func preferredModelAlias(model Model) string {
	for _, alias := range model.AIModelCodeAlias {
		alias = strings.TrimSpace(alias)
		if alias != "" {
			return alias
		}
	}
	return strings.TrimSpace(model.Code)
}

func preferredModelName(model Model) string {
	name := strings.TrimSpace(model.Name)
	if name == "" {
		return strings.TrimSpace(model.Code)
	}
	replaced := strings.NewReplacer("Popi ", "popi-", " ", "-", "·", "-", "· ", "-").Replace(name)
	replaced = strings.Trim(strings.ToLower(replaced), "-")
	replaced = strings.ReplaceAll(replaced, "--", "-")
	return replaced
}

func resolveMusicTaskSubType(payload map[string]any, model Model) int {
	requestedModel := strings.TrimSpace(stringField(payload, "requested_model"))
	if strings.EqualFold(requestedModel, "write_full_song") {
		return 305
	}
	if SupportsSubType(model, 305) && !SupportsSubType(model, 304) {
		return 305
	}
	return 304
}

func resolvedVideoTaskSubType(payload map[string]any, model Model) int {
	requested := PreferredSubType(model, []int{204, 203, 202})
	images := stringSliceField(payload["images"])
	videos := stringSliceField(payload["videos"])
	audios := stringSliceField(payload["audios"])

	switch {
	case len(images) == 2 && len(videos) == 0 && len(audios) == 0 && SupportsSubType(model, 204):
		return 204
	case len(images) > 0 || len(videos) > 0 || len(audios) > 0:
		if (len(videos) > 0 || len(audios) > 0 || len(images) == 1) && SupportsSubType(model, 203) {
			return 203
		}
		if len(images) >= 1 && SupportsSubType(model, 202) {
			return 202
		}
	case SupportsSubType(model, 203):
		return 203
	case SupportsSubType(model, 202):
		return 202
	}

	return requested
}

func taskCanvasDimensions(aspectRatio, resolution string, taskType int) (int, int) {
	if width, height, ok := parseExplicitDimensions(resolution); ok {
		return width, height
	}

	ratio := normalizeTaskAspectRatio(strings.TrimSpace(aspectRatio))
	if ratio == "" {
		ratio = "16:9"
	}

	if width, height, ok := taskDimensionsForResolution(ratio, resolution, taskType); ok {
		return width, height
	}

	switch ratio {
	case "21:9":
		return 1680, 720
	case "16:9":
		return 1280, 720
	case "9:16":
		return 720, 1280
	case "4:3":
		return 960, 720
	case "3:4":
		return 720, 960
	case "3:2":
		return 1080, 720
	case "2:3":
		return 720, 1080
	case "1:1":
		return 1024, 1024
	case "5:4":
		return 900, 720
	case "4:5":
		return 720, 900
	default:
		return 1280, 720
	}
}

func resolvedTaskAspectRatio(value string) string {
	value = normalizeTaskAspectRatio(strings.TrimSpace(value))
	if value == "" {
		return "16:9"
	}
	return value
}

func resolveModelAspectRatio(requested string, model Model, isVideo bool) string {
	requested = normalizeTaskAspectRatio(strings.TrimSpace(requested))
	if requested != "" {
		return requested
	}
	candidates := model.Ratio
	if isVideo && len(model.VideoRatio) > 0 {
		candidates = model.VideoRatio
	}
	for _, value := range candidates {
		normalized := normalizeTaskAspectRatio(strings.TrimSpace(value))
		if normalized != "" {
			return normalized
		}
	}
	return resolvedTaskAspectRatio("")
}

func resolvedTaskResolution(requested string, model Model, taskType int) string {
	requested = strings.TrimSpace(requested)
	if requested != "" {
		if _, _, ok := parseExplicitDimensions(requested); ok {
			return ""
		}
		return normalizeResolutionLabel(requested)
	}
	if len(model.Resolution) == 0 {
		return ""
	}
	values := make([]string, 0, len(model.Resolution))
	for _, value := range model.Resolution {
		value = normalizeResolutionLabel(value)
		if value == "" {
			continue
		}
		values = append(values, value)
	}
	if len(values) == 0 {
		return ""
	}
	sort.SliceStable(values, func(i, j int) bool {
		return resolutionRank(values[i], taskType) < resolutionRank(values[j], taskType)
	})
	return values[0]
}

func resolvedTaskDuration(requested int, model Model) int {
	if requested > 0 {
		return requested
	}
	if len(model.Duration) == 0 {
		return 0
	}
	values := make([]int, 0, len(model.Duration))
	seen := map[int]struct{}{}
	for _, value := range model.Duration {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	if len(values) == 0 {
		return 0
	}
	sort.Ints(values)
	return values[0]
}

func taskDimensionsForResolution(aspectRatio, resolution string, taskType int) (int, int, bool) {
	resolution = normalizeResolutionLabel(resolution)
	if resolution == "" {
		return 0, 0, false
	}

	shortSide, ok := resolutionShortSide(resolution, taskType)
	if !ok {
		return 0, 0, false
	}

	ratioWidth, ratioHeight, ok := parseAspectRatioPair(aspectRatio)
	if !ok {
		return 0, 0, false
	}

	if ratioWidth >= ratioHeight {
		width := roundedDimension(float64(shortSide) * float64(ratioWidth) / float64(ratioHeight))
		return width, shortSide, true
	}
	height := roundedDimension(float64(shortSide) * float64(ratioHeight) / float64(ratioWidth))
	return shortSide, height, true
}

func resolutionShortSide(resolution string, taskType int) (int, bool) {
	switch normalizeResolutionLabel(resolution) {
	case "480P":
		return 480, true
	case "540P":
		return 540, true
	case "720P":
		return 720, true
	case "1080P":
		return 1080, true
	case "1K":
		if taskType == 1 {
			return 1440, true
		}
		return 1024, true
	case "2K":
		if taskType == 1 {
			return 2880, true
		}
		return 1440, true
	case "4K":
		if taskType == 1 {
			return 4096, true
		}
		return 2160, true
	default:
		return 0, false
	}
}

func parseExplicitDimensions(value string) (int, int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, 0, false
	}
	normalized := strings.NewReplacer("×", "x", "*", "x", "X", "x").Replace(value)
	parts := strings.Split(normalized, "x")
	if len(parts) != 2 {
		return 0, 0, false
	}
	width, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil || width <= 0 || height <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

func parseAspectRatioPair(value string) (int, int, bool) {
	value = normalizeTaskAspectRatio(value)
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	width, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil || width <= 0 || height <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

func roundedDimension(value float64) int {
	if value <= 0 {
		return 0
	}
	rounded := int(math.Round(value/8.0) * 8)
	if rounded <= 0 {
		return int(math.Round(value))
	}
	return rounded
}

func normalizeResolutionLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if _, _, ok := parseExplicitDimensions(value); ok {
		return value
	}
	upper := strings.ToUpper(strings.ReplaceAll(value, " ", ""))
	switch upper {
	case "480P", "540P", "720P", "1080P", "1K", "2K", "4K":
		return upper
	default:
		return value
	}
}

func resolutionRank(value string, taskType int) int {
	switch normalizeResolutionLabel(value) {
	case "480P":
		return 10
	case "540P":
		return 20
	case "720P":
		return 30
	case "1K":
		if taskType == 1 {
			return 35
		}
		return 40
	case "1080P":
		return 50
	case "2K":
		return 60
	case "4K":
		return 70
	default:
		return 1000
	}
}

func normalizeTaskAspectRatio(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	replaced := strings.NewReplacer("：", ":", "／", ":", "/", ":", "×", ":", "*", ":", "x", ":", "X", ":").Replace(value)
	parts := strings.Split(replaced, ":")
	if len(parts) != 2 {
		return value
	}
	width, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil || width <= 0 || height <= 0 {
		return value
	}
	g := gcdLocal(width, height)
	if g <= 0 {
		return fmt.Sprintf("%d:%d", width, height)
	}
	key := fmt.Sprintf("%d:%d", width/g, height/g)
	switch key {
	case "7:3":
		return "21:9"
	case "3:7":
		return "9:21"
	default:
		return key
	}
}

func gcdLocal(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		return -a
	}
	return a
}

func stringField(payload map[string]any, key string) string {
	value, _ := payload[key]
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}

func stringSliceField(value any) []string {
	switch typed := value.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				out = append(out, strings.TrimSpace(text))
			}
		}
		return out
	default:
		return nil
	}
}

func putMetadataString(metadata map[string]any, key string, value any) {
	if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
		metadata[key] = strings.TrimSpace(text)
	}
}

func putMetadataFloat(metadata map[string]any, key string, value any) {
	switch typed := value.(type) {
	case float64:
		if typed != 0 {
			metadata[key] = typed
		}
	case float32:
		if typed != 0 {
			metadata[key] = typed
		}
	}
}

func putMetadataInt(metadata map[string]any, key string, value any) {
	switch typed := value.(type) {
	case int:
		if typed != 0 {
			metadata[key] = typed
		}
	case int64:
		if typed != 0 {
			metadata[key] = typed
		}
	case float64:
		if typed != 0 {
			metadata[key] = int(typed)
		}
	}
}

func putMetadataBool(metadata map[string]any, key string, value any) {
	if flag, ok := value.(bool); ok && flag {
		metadata[key] = true
	}
}

func intField(payload map[string]any, key string, fallback int) int {
	value, ok := payload[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return fallback
	}
}
