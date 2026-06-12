package popiart

import (
	"strings"

	"github.com/wtgoku-create/popiartcli/internal/output"
)

// NormalizeAPIError 把主站返回的错误文案映射到 CLI 既有错误码体系。
func NormalizeAPIError(err error) error {
	cliErr, ok := err.(*output.CLIError)
	if !ok {
		return err
	}

	code := mapRemoteMessageToCode(cliErr.Message)
	if code == "" || code == cliErr.Code {
		return err
	}
	return output.NewError(code, cliErr.Message, cliErr.Details)
}

func mapRemoteMessageToCode(message string) string {
	switch strings.TrimSpace(message) {
	case "Please login first", "invalid user":
		return "UNAUTHENTICATED"
	case "Your account has been logged out.":
		return "SESSION_EXPIRED"
	default:
		return ""
	}
}

// taskFailureMessage 按迁移文档定义的优先级提取任务失败文案。
func taskFailureMessage(task TaskDetail) string {
	if strings.TrimSpace(task.ErrorMsg) != "" {
		return strings.TrimSpace(task.ErrorMsg)
	}
	if strings.TrimSpace(task.UserErrorTipMsg) != "" {
		return strings.TrimSpace(task.UserErrorTipMsg)
	}
	return "Task failed"
}
