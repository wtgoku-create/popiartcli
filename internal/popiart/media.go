package popiart

import (
	"context"
	"strings"

	"github.com/wtgoku-create/popiartcli/internal/api"
	"github.com/wtgoku-create/popiartcli/internal/output"
)

// UploadOptions 描述主站媒体上传时需要的文件与重试参数。
type UploadOptions struct {
	Filename    string
	ContentType string
	Fields      map[string]string
	MaxRetries  int
}

type mediaUploadResponse struct {
	ID          any    `json:"id"`
	ArtifactID  string `json:"artifact_id,omitempty"`
	ProjectID   string `json:"project_id,omitempty"`
	Filename    string `json:"filename,omitempty"`
	Name        string `json:"name,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	SizeBytes   int64  `json:"size_bytes,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	CreateTime  string `json:"createTime,omitempty"`
	URL         string `json:"url,omitempty"`
	FileURL     string `json:"fileUrl,omitempty"`
	FilePath    string `json:"filePath,omitempty"`
	Path        string `json:"path,omitempty"`
	Visibility  string `json:"visibility,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
}

// FetchMediaDetail 读取主站 media/detail，并把字段别名归一化成统一媒体对象。
func FetchMediaDetail(ctx context.Context, client *api.Client, id string) (MediaObject, error) {
	var payload mediaUploadResponse
	if err := client.GetJSON(ctx, "/api_client/media/detail", map[string]string{"id": strings.TrimSpace(id)}, &payload); err != nil {
		return MediaObject{}, NormalizeAPIError(err)
	}

	media := normalizeUploadedMedia(payload)
	if strings.TrimSpace(media.ID) == "" {
		media.ID = strings.TrimSpace(id)
	}
	return media, nil
}

// UploadMedia 统一走主站 media/upload，并按迁移文档执行有限重试。
func UploadMedia(ctx context.Context, client *api.Client, path string, opts UploadOptions) (MediaObject, error) {
	maxRetries := opts.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		var payload mediaUploadResponse
		err := client.UploadFile(ctx, "/api_client/media/upload", path, api.UploadFileOptions{
			Filename:    opts.Filename,
			ContentType: opts.ContentType,
			Fields:      opts.Fields,
		}, &payload)
		if err == nil {
			media := normalizeUploadedMedia(payload)
			if strings.TrimSpace(media.URL) == "" {
				return MediaObject{}, output.NewError("UPLOAD_FAILED", "上传响应缺少 url", map[string]any{
					"path": path,
				})
			}
			return media, nil
		}

		lastErr = NormalizeAPIError(err)
		if !shouldRetryUpload(lastErr) || attempt == maxRetries {
			return MediaObject{}, lastErr
		}
	}

	if lastErr == nil {
		lastErr = output.NewError("UPLOAD_FAILED", "上传失败", map[string]any{"path": path})
	}
	return MediaObject{}, lastErr
}

// normalizeUploadedMedia 把主站上传响应中的字段别名归一化为 CLI 统一媒体对象。
func normalizeUploadedMedia(payload mediaUploadResponse) MediaObject {
	return MediaObject{
		ID:          stringifyUserID(payload.ID),
		ArtifactID:  payload.ArtifactID,
		ProjectID:   payload.ProjectID,
		Filename:    firstNonEmpty(payload.Filename, payload.Name),
		ContentType: payload.ContentType,
		SizeBytes:   payload.SizeBytes,
		CreatedAt:   firstNonEmpty(payload.CreatedAt, payload.CreateTime),
		URL:         firstNonEmpty(payload.URL, payload.FileURL, payload.FilePath, payload.Path),
		Visibility:  payload.Visibility,
		SHA256:      payload.SHA256,
	}
}

// firstNonEmpty 返回第一个非空字符串，用于兼容主站同义字段。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

// shouldRetryUpload 只允许网络错误和 5xx 服务错误进入重试分支。
func shouldRetryUpload(err error) bool {
	cliErr, ok := err.(*output.CLIError)
	if !ok {
		return false
	}

	switch strings.ToUpper(strings.TrimSpace(cliErr.Code)) {
	case "NETWORK_ERROR", "SERVER_ERROR", "SERVICE_UNAVAILABLE":
		return true
	case "HTTP_ERROR":
		status, ok := toInt(cliErr.Details["status"])
		return ok && status >= 500
	default:
		return false
	}
}
