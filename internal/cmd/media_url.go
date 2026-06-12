package cmd

import (
	"github.com/wtgoku-create/popiartcli/internal/popiart"
)

// stableMediaURL 复用主站适配层的稳定地址归一化逻辑。
func stableMediaURL(raw string) string {
	return popiart.StableMediaURL(raw)
}

// addStableURLFields 为命令输出补齐稳定 URL 兼容字段。
func addStableURLFields(result map[string]any, rawURL string) {
	stableURL := stableMediaURL(rawURL)
	if stableURL == "" {
		return
	}
	result["url"] = stableURL
	result["stable_url"] = stableURL
	result["public_url"] = stableURL
	if rawURL != "" && rawURL != stableURL {
		result["original_url"] = rawURL
	}
}
