package output

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

const rechargeURL = "https://wwwskillhub.popi.art"

var openRechargeURL = openURL

func maybeHandleRecharge(cliErr *CLIError) {
	if cliErr == nil || !shouldPromptRecharge(cliErr) {
		return
	}

	if cliErr.Details == nil {
		cliErr.Details = map[string]any{}
	}
	if _, exists := cliErr.Details["recharge_url"]; !exists {
		cliErr.Details["recharge_url"] = rechargeURL
	}

	opened, err := openRechargeURL(rechargeURL)
	cliErr.Details["recharge_prompted"] = true
	cliErr.Details["recharge_opened"] = opened
	if err != nil {
		cliErr.Details["recharge_open_error"] = err.Error()
	}
}

func shouldPromptRecharge(cliErr *CLIError) bool {
	if cliErr == nil {
		return false
	}

	if status, ok := cliErr.Details["status"]; ok {
		switch status {
		case 402, float64(402):
			return true
		}
	}

	fields := []string{
		cliErr.Code,
		cliErr.Message,
	}
	for _, key := range []string{"reason", "hint", "details", "error", "type"} {
		if value, ok := cliErr.Details[key]; ok {
			fields = append(fields, fmt.Sprint(value))
		}
	}

	text := strings.ToLower(strings.Join(fields, " "))
	if text == "" {
		return false
	}

	matchers := []string{
		"payment_required",
		"insufficient",
		"insufficient_budget",
		"insufficient_balance",
		"insufficient_credit",
		"insufficient_credits",
		"budget exceeded",
		"budget_exceeded",
		"quota exceeded",
		"quota_exceeded",
		"credit exceeded",
		"credit_exceeded",
		"余额不足",
		"积分不足",
		"额度不足",
		"配额不足",
		"预算不足",
		"需要充值",
	}
	for _, matcher := range matchers {
		if strings.Contains(text, matcher) {
			return true
		}
	}

	return false
}

func openURL(target string) (bool, error) {
	cmdName, args, ok := browserOpenCommand(target)
	if !ok {
		return false, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	cmd := exec.Command(cmdName, args...)
	if err := cmd.Start(); err != nil {
		return false, err
	}
	return true, nil
}

func browserOpenCommand(target string) (string, []string, bool) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{target}, true
	case "linux":
		return "xdg-open", []string{target}, true
	case "windows":
		return "cmd", []string{"/c", "start", "", target}, true
	default:
		return "", nil, false
	}
}
