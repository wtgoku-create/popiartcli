package popiart

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// StringList 用于兼容主站把同一字段返回成 CSV 字符串或字符串数组的情况。
type StringList []string

func (l *StringList) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*l = nil
		return nil
	}

	if len(data) > 0 && data[0] == '[' {
		var raw []any
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		*l = normalizeAnyStringList(raw)
		return nil
	}

	var single string
	if err := json.Unmarshal(data, &single); err != nil {
		var number json.Number
		if err := json.Unmarshal(data, &number); err != nil {
			return err
		}
		*l = []string{number.String()}
		return nil
	}
	*l = splitCSV(single)
	return nil
}

// IntList 用于兼容主站把数值候选集返回成单值、CSV 字符串或数组的情况。
type IntList []int

func (l *IntList) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*l = nil
		return nil
	}

	if len(data) > 0 && data[0] == '[' {
		var raw []any
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		values := make([]int, 0, len(raw))
		for _, item := range raw {
			value, ok := toInt(item)
			if !ok {
				continue
			}
			values = append(values, value)
		}
		*l = values
		return nil
	}

	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		parts := splitCSV(single)
		values := make([]int, 0, len(parts))
		for _, part := range parts {
			value, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil {
				continue
			}
			values = append(values, value)
		}
		*l = values
		return nil
	}

	var number int
	if err := json.Unmarshal(data, &number); err == nil {
		*l = IntList{number}
		return nil
	}

	return nil
}

// UserInfo 表示主站 user/info 接口返回的基础用户信息。
type UserInfo struct {
	ID     string   `json:"id"`
	Email  string   `json:"email"`
	Name   string   `json:"name"`
	Scopes []string `json:"scopes,omitempty"`
}

// MediaObject 表示主站媒体上传后的标准返回结构。
type MediaObject struct {
	ID          string `json:"id"`
	ArtifactID  string `json:"artifact_id,omitempty"`
	ProjectID   string `json:"project_id,omitempty"`
	Filename    string `json:"filename,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	SizeBytes   int64  `json:"size_bytes,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	URL         string `json:"url,omitempty"`
	Visibility  string `json:"visibility,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
}

// ModelCategory 表示模型支持的任务子类型。
type ModelCategory struct {
	TaskSubType int  `json:"taskSubType"`
	Status      *int `json:"status,omitempty"`
}

// Model 表示 ai/model/list 中会被 CLI 消费的核心能力字段。
type Model struct {
	ID                int64           `json:"id"`
	Code              string          `json:"code"`
	AIModelCodeAlias  StringList      `json:"aiModelCodeAlias,omitempty"`
	Name              string          `json:"name,omitempty"`
	Ratio             StringList      `json:"ratio,omitempty"`
	VideoRatio        StringList      `json:"videoRatio,omitempty"`
	Resolution        StringList      `json:"resolution,omitempty"`
	Duration          IntList         `json:"duration,omitempty"`
	DisplayDimensions json.RawMessage `json:"displayDimensions,omitempty"`
	BillingDimensions json.RawMessage `json:"billingDimensions,omitempty"`
	IsSupportImages   bool            `json:"isSupportImages,omitempty"`
	IsSupportVideos   bool            `json:"isSupportVideos,omitempty"`
	IsSupportAudios   bool            `json:"isSupportAudios,omitempty"`
	UploadImageLimit  *int            `json:"uploadImageLimit,omitempty"`
	UploadVideoLimit  *int            `json:"uploadVideoLimit,omitempty"`
	UploadAudioLimit  *int            `json:"uploadAudioLimit,omitempty"`
	Categories        []ModelCategory `json:"categories,omitempty"`
	Providers         json.RawMessage `json:"providers,omitempty"`
	BillingBindings   json.RawMessage `json:"billingBindings,omitempty"`
}

// TaskRequest 表示提交到主站 task/create 的统一任务结构。
type TaskRequest struct {
	Type             int            `json:"type"`
	SubType          int            `json:"subType"`
	ProjectID        int            `json:"projectId,omitempty"`
	Model            string         `json:"model,omitempty"`
	AIModelCode      string         `json:"aiModelCode,omitempty"`
	AIModelCodeAlias string         `json:"aiModelCodeAlias,omitempty"`
	AIModelName      string         `json:"aiModelname,omitempty"`
	AIModelID        int64          `json:"aiModelId,omitempty"`
	StyleID          int            `json:"styleId,omitempty"`
	Width            int            `json:"width,omitempty"`
	Height           int            `json:"height,omitempty"`
	ChatPrompt       string         `json:"chatPrompt,omitempty"`
	Images           []string       `json:"images,omitempty"`
	Videos           []string       `json:"videos,omitempty"`
	Voices           []string       `json:"voices,omitempty"`
	AspectRatio      string         `json:"aspectRatio,omitempty"`
	Ratio            string         `json:"ratio,omitempty"`
	Resolution       string         `json:"resolution,omitempty"`
	Duration         int            `json:"duration,omitempty"`
	VoiceID          string         `json:"voiceId,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	BatchSize        int            `json:"batchSize,omitempty"`
}

// TaskDetail 表示主站任务查询结果及 CLI 兼容输出所需字段。
type TaskDetail struct {
	ID              FlexibleString `json:"id,omitempty"`
	TaskID          FlexibleString `json:"task_id,omitempty"`
	Status          FlexibleInt    `json:"status,omitempty"`
	Type            FlexibleInt    `json:"type,omitempty"`
	SubType         FlexibleInt    `json:"subType,omitempty"`
	AIModelCode     string         `json:"aiModelCode,omitempty"`
	AIModelID       int64          `json:"aiModelId,omitempty"`
	DownloadURLs    []string       `json:"download_urls,omitempty"`
	ResultURLs      []string       `json:"result_urls,omitempty"`
	OutputText      string         `json:"output_text,omitempty"`
	Text            string         `json:"text,omitempty"`
	UserErrorTipMsg string         `json:"user_error_tip_msg,omitempty"`
	ErrorMsg        string         `json:"error_msg,omitempty"`
	Raw             map[string]any `json:"-"`
}

// TaskListPage 表示主站 task/list 的分页结果。
type TaskListPage struct {
	Items  []TaskDetail `json:"items,omitempty"`
	Total  int          `json:"total,omitempty"`
	Limit  int          `json:"limit,omitempty"`
	Offset int          `json:"offset,omitempty"`
}

// LLMChatRequest 表示主站 llmChat 请求结构。
type LLMChatRequest struct {
	Messages  []LLMChatMessage `json:"messages"`
	Model     string           `json:"model,omitempty"`
	Stream    bool             `json:"stream"`
	AIModelID int64            `json:"aiModelId,omitempty"`
}

type LLMChatMessage struct {
	Role    string               `json:"role"`
	Content []LLMChatContentPart `json:"content"`
}

type LLMChatContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

// LLMChatResponse 表示主站 llmChat 响应里 CLI 关心的字段。
type LLMChatResponse struct {
	Choices []LLMChatChoice `json:"choices,omitempty"`
	Model   string          `json:"model,omitempty"`
	ID      string          `json:"id,omitempty"`
	Object  string          `json:"object,omitempty"`
	Created int64           `json:"created,omitempty"`
}

type LLMChatChoice struct {
	Index        int                     `json:"index,omitempty"`
	FinishReason string                  `json:"finish_reason,omitempty"`
	Message      LLMChatAssistantMessage `json:"message"`
}

type LLMChatAssistantMessage struct {
	Role             string `json:"role,omitempty"`
	Content          string `json:"content,omitempty"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

// Identifier 返回 CLI 内部统一使用的任务主键。
func (t TaskDetail) Identifier() string {
	if strings.TrimSpace(string(t.TaskID)) != "" {
		return strings.TrimSpace(string(t.TaskID))
	}
	return strings.TrimSpace(string(t.ID))
}

// ModelValidationSpec 描述一次命令执行前需要校验的模型能力约束。
type ModelValidationSpec struct {
	SubType         int
	AllowedSubTypes []int
	ImageCount      int
	VideoCount      int
	AudioCount      int
	RequiresImages  bool
	RequiresVideos  bool
	RequiresAudios  bool
	Ratio           string
	VideoRatio      string
	Resolution      string
	Duration        int
}

func normalizeStringList(values []string) []string {
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

// normalizeAnyStringList 把主站返回的字符串、数字混合数组归一化成字符串列表。
func normalizeAnyStringList(values []any) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		switch typed := value.(type) {
		case string:
			typed = strings.TrimSpace(typed)
			if typed != "" {
				out = append(out, typed)
			}
		case float64:
			out = append(out, strconv.FormatFloat(typed, 'f', -1, 64))
		case json.Number:
			if typed.String() != "" {
				out = append(out, typed.String())
			}
		}
	}
	return out
}

func splitCSV(value string) []string {
	parts := strings.Split(strings.TrimSpace(value), ",")
	return normalizeStringList(parts)
}

// FlexibleString 用于兼容主站把标识符字段返回为字符串或数字的情况。
type FlexibleString string

func (s *FlexibleString) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*s = ""
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = FlexibleString(str)
		return nil
	}

	var number json.Number
	if err := json.Unmarshal(data, &number); err == nil {
		*s = FlexibleString(number.String())
		return nil
	}

	return nil
}

// FlexibleInt 用于兼容主站把数值字段返回为数字或数字字符串的情况。
type FlexibleInt int

func (v *FlexibleInt) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*v = 0
		return nil
	}

	var number int
	if err := json.Unmarshal(data, &number); err == nil {
		*v = FlexibleInt(number)
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		parsed, err := strconv.Atoi(strings.TrimSpace(text))
		if err == nil {
			*v = FlexibleInt(parsed)
			return nil
		}
	}

	return nil
}

func toInt(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		return parsed, err == nil
	default:
		return 0, false
	}
}
