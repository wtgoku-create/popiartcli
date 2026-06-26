package popiart

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/wtgoku-create/popiartcli/internal/api"
	"github.com/wtgoku-create/popiartcli/internal/output"
)

// CreateTask 提交统一任务到主站 task/create 接口。
func CreateTask(ctx context.Context, client *api.Client, req TaskRequest) (TaskDetail, error) {
	var payload any
	if err := client.PostJSON(ctx, "/api_client/anime/task/create", normalizeTaskRequest(req), &payload); err != nil {
		return TaskDetail{}, NormalizeAPIError(err)
	}
	task := normalizeTaskCreateResponse(payload)
	return task, nil
}

// GetTaskDetail 查询单个任务的当前状态与元数据。
func GetTaskDetail(ctx context.Context, client *api.Client, id string) (TaskDetail, error) {
	var payload any
	if err := client.GetJSON(ctx, "/api_client/anime/task/detail", map[string]string{"id": id}, &payload); err != nil {
		return TaskDetail{}, NormalizeAPIError(err)
	}
	task := normalizeTaskDetail(payload)
	if task.TaskID == "" && task.ID != "" {
		task.TaskID = task.ID
	}
	return task, nil
}

// GetTaskDownloadURLs 查询任务成功后的最终下载地址集合。
func GetTaskDownloadURLs(ctx context.Context, client *api.Client, id string) ([]string, error) {
	var payload any
	if err := client.GetJSON(ctx, "/api_client/anime/task/downloadUrls", map[string]string{"id": id}, &payload); err != nil {
		return nil, NormalizeAPIError(err)
	}
	return extractDownloadURLs(payload), nil
}

// ListTasks 查询主站任务列表分页结果。
func ListTasks(ctx context.Context, client *api.Client, status, limit, offset string) (TaskListPage, error) {
	query := map[string]string{
		"status": status,
		"limit":  limit,
		"offset": offset,
	}

	var payload any
	if err := client.GetJSON(ctx, "/api_client/anime/task/list", query, &payload); err != nil {
		return TaskListPage{}, NormalizeAPIError(err)
	}
	return normalizeTaskListPage(payload, limit, offset), nil
}

// EnhancePromptWithLLM 使用主站 llmChat 获取增强后的 prompt 文本。
func EnhancePromptWithLLM(ctx context.Context, client *api.Client, req LLMChatRequest) (LLMChatResponse, error) {
	var resp LLMChatResponse
	if err := client.PostJSON(ctx, "/api_client/anime/task/llmChat", req, &resp); err != nil {
		return LLMChatResponse{}, NormalizeAPIError(err)
	}
	return resp, nil
}

// WaitForTask 轮询任务直到成功、失败、取消或超时。
func WaitForTask(ctx context.Context, client *api.Client, id string, interval time.Duration, maxPolls int) (TaskDetail, error) {
	for pollIndex := 0; pollIndex < maxPolls; pollIndex++ {
		task, err := GetTaskDetail(ctx, client, id)
		if err != nil {
			return TaskDetail{}, err
		}

		switch task.Status {
		case FlexibleInt(2):
			urls, err := GetTaskDownloadURLs(ctx, client, id)
			if err == nil && len(urls) > 0 {
				task.DownloadURLs = append([]string(nil), urls...)
				task.ResultURLs = append([]string(nil), urls...)
			}
			return task, nil
		case FlexibleInt(-1):
			return task, nil
		case FlexibleInt(-2):
			return TaskDetail{}, output.NewError("JOB_FAILED", taskFailureMessage(task), map[string]any{
				"task_id": id,
				"status":  task.Status,
			})
		}

		time.Sleep(interval)
	}

	return TaskDetail{}, output.NewError("POLL_TIMEOUT", fmt.Sprintf("Task %s did not complete within the timeout", id), map[string]any{
		"task_id":         id,
		"timeout_seconds": int(interval.Seconds()) * maxPolls,
	})
}

// extractDownloadURLs 兼容主站返回的多种下载地址包装格式。
func extractDownloadURLs(payload any) []string {
	switch typed := payload.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if value, ok := item.(string); ok && value != "" {
				out = append(out, value)
				continue
			}
			if object, ok := item.(map[string]any); ok {
				if value := strings.TrimSpace(firstNonEmptyAny(object["url"], object["downloadUrl"], object["download_url"])); value != "" {
					out = append(out, value)
				}
			}
		}
		return out
	case map[string]any:
		if value := strings.TrimSpace(firstNonEmptyAny(typed["url"], typed["downloadUrl"], typed["download_url"])); value != "" {
			return []string{value}
		}
		for _, key := range []string{"download_urls", "downloadUrls", "urls", "items"} {
			if raw, ok := typed[key]; ok {
				return extractDownloadURLs(raw)
			}
		}
	case []string:
		return append([]string(nil), typed...)
	}
	return nil
}

func normalizeTaskListPage(payload any, limitRaw, offsetRaw string) TaskListPage {
	page := TaskListPage{
		Limit:  parseOptionalInt(limitRaw),
		Offset: parseOptionalInt(offsetRaw),
	}

	switch typed := payload.(type) {
	case []any:
		page.Items = normalizeTaskDetailList(typed)
		page.Total = len(page.Items)
		return page
	case map[string]any:
		if total, ok := toInt(typed["total"]); ok {
			page.Total = total
		}
		if limit, ok := toInt(typed["limit"]); ok {
			page.Limit = limit
		}
		if offset, ok := toInt(typed["offset"]); ok {
			page.Offset = offset
		}
		for _, key := range []string{"items", "list", "records", "data"} {
			if raw, ok := typed[key]; ok {
				page.Items = normalizeTaskDetailSlice(raw)
				if page.Total == 0 {
					page.Total = len(page.Items)
				}
				return page
			}
		}
	case []TaskDetail:
		page.Items = append([]TaskDetail(nil), typed...)
		page.Total = len(page.Items)
		return page
	}

	return page
}

func normalizeTaskDetailSlice(value any) []TaskDetail {
	switch typed := value.(type) {
	case []any:
		return normalizeTaskDetailList(typed)
	case []TaskDetail:
		return append([]TaskDetail(nil), typed...)
	default:
		return nil
	}
}

func normalizeTaskDetailList(items []any) []TaskDetail {
	out := make([]TaskDetail, 0, len(items))
	for _, item := range items {
		detail := normalizeTaskDetail(item)
		if detail.Identifier() == "" && detail.Status == 0 && detail.Type == 0 && detail.SubType == 0 {
			continue
		}
		out = append(out, detail)
	}
	return out
}

func normalizeTaskDetail(value any) TaskDetail {
	rawMap, ok := value.(map[string]any)
	if !ok {
		return TaskDetail{}
	}

	detail := TaskDetail{
		ID:              FlexibleString(stringifyAny(rawMap["id"])),
		TaskID:          FlexibleString(firstNonEmptyAny(rawMap["task_id"], rawMap["taskId"])),
		Status:          FlexibleInt(intFromAny(rawMap["status"])),
		Type:            FlexibleInt(intFromAny(rawMap["type"])),
		SubType:         FlexibleInt(intFromAny(firstNonEmptyValue(rawMap["subType"], rawMap["sub_type"]))),
		AIModelCode:     strings.TrimSpace(firstNonEmptyAny(rawMap["aiModelCode"], rawMap["model"], rawMap["ai_model_code"])),
		AIModelID:       int64(intFromAny(firstNonEmptyValue(rawMap["aiModelId"], rawMap["ai_model_id"]))),
		DownloadURLs:    extractDownloadURLs(firstNonEmptyValue(rawMap["download_urls"], rawMap["downloadUrls"])),
		ResultURLs:      extractDownloadURLs(firstNonEmptyValue(rawMap["result_urls"], rawMap["resultUrls"], rawMap["urls"])),
		OutputText:      strings.TrimSpace(firstNonEmptyAny(rawMap["output_text"], rawMap["outputText"])),
		Text:            strings.TrimSpace(firstNonEmptyAny(rawMap["text"])),
		UserErrorTipMsg: strings.TrimSpace(firstNonEmptyAny(rawMap["user_error_tip_msg"], rawMap["userErrorTipMsg"], rawMap["userErrorTipMsg"])),
		ErrorMsg:        strings.TrimSpace(firstNonEmptyAny(rawMap["error_msg"], rawMap["errorMsg"], rawMap["errorMsg"])),
		Raw:             rawMap,
	}
	if detail.TaskID == "" && detail.ID != "" {
		detail.TaskID = detail.ID
	}
	return detail
}

func normalizeTaskCreateResponse(payload any) TaskDetail {
	task := normalizeTaskDetail(payload)
	if task.Identifier() != "" {
		return task
	}

	items := normalizeTaskDetailSlice(payload)
	for _, item := range items {
		if item.Identifier() != "" {
			return item
		}
	}

	return TaskDetail{}
}

func parseOptionalInt(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return value
}

func stringifyAny(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return ""
	}
}

func intFromAny(value any) int {
	if number, ok := toInt(value); ok {
		return number
	}
	return 0
}

func firstNonEmptyAny(values ...any) string {
	for _, value := range values {
		if text := stringifyAny(value); text != "" {
			return text
		}
	}
	return ""
}

func firstNonEmptyValue(values ...any) any {
	for _, value := range values {
		switch typed := value.(type) {
		case nil:
			continue
		case string:
			if strings.TrimSpace(typed) == "" {
				continue
			}
		}
		return value
	}
	return nil
}

// normalizeTaskRequest 在发往主站前规整字段格式，兼容 metadata 需要字符串的生产行为。
func normalizeTaskRequest(req TaskRequest) map[string]any {
	body := map[string]any{
		"type":             req.Type,
		"subType":          req.SubType,
		"model":            req.Model,
		"aiModelCode":      req.AIModelCode,
		"aiModelCodeAlias": req.AIModelCodeAlias,
		"aiModelname":      req.AIModelName,
		"aiModelId":        req.AIModelID,
		"styleId":          req.StyleID,
		"width":            req.Width,
		"height":           req.Height,
		"batchSize":        req.BatchSize,
	}
	if req.ProjectID != 0 {
		body["projectId"] = req.ProjectID
	}
	if strings.TrimSpace(req.ChatPrompt) != "" {
		body["chatPrompt"] = req.ChatPrompt
	}
	if req.Images != nil {
		body["images"] = append([]string(nil), req.Images...)
	}
	if req.Videos != nil {
		body["videos"] = append([]string(nil), req.Videos...)
	}
	if req.Voices != nil {
		body["voices"] = append([]string(nil), req.Voices...)
	}
	if req.AspectRatio != "" {
		body["aspectRatio"] = req.AspectRatio
	}
	if req.Ratio != "" {
		body["ratio"] = req.Ratio
	}
	if req.Resolution != "" {
		body["resolution"] = req.Resolution
	}
	if req.Duration != 0 {
		body["duration"] = req.Duration
	}
	if req.VoiceID != "" {
		body["voiceId"] = req.VoiceID
	}
	if len(req.Metadata) > 0 {
		if payload, err := json.Marshal(req.Metadata); err == nil {
			body["metadata"] = string(payload)
		}
	}
	return body
}
