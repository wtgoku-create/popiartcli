package input

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/wtgoku-create/popiartcli/internal/output"
)

func Resolve(raw string) (any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}

	switch {
	case raw == "-":
		data, err := os.ReadFile("/dev/stdin")
		if err != nil {
			return nil, output.NewError("INPUT_ERROR", "Failed to read from stdin", nil)
		}
		return parseJSON(data, "标准输入")
	case strings.HasPrefix(raw, "@"):
		path := strings.TrimPrefix(raw, "@")
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, output.NewError("INPUT_NOT_FOUND", "未找到文件: "+path, nil)
		}
		return parseJSON(data, path)
	default:
		return parseJSON([]byte(raw), "内联输入")
	}
}

func parseJSON(data []byte, label string) (any, error) {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, output.NewError("INPUT_PARSE_ERROR", label+" 中存在无效的 JSON: "+err.Error(), map[string]any{
			"hint": "使用 @file.json 进行文件输入，或者传入有效的 JSON 字符串",
		})
	}
	return value, nil
}
