package cmd

import (
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/api"
	"github.com/wtgoku-create/popiartcli/internal/output"
)

func flagString(cmd *cobra.Command, name string) string {
	if cmd == nil {
		return ""
	}
	value, _ := cmd.Flags().GetString(name)
	return value
}

func flagBool(cmd *cobra.Command, name string) bool {
	if cmd == nil {
		return false
	}
	value, _ := cmd.Flags().GetBool(name)
	return value
}

func apiOpts(token, accept string) api.RequestOptions {
	return api.RequestOptions{
		Token:  token,
		Accept: accept,
	}
}

func intervalDuration(cmd *cobra.Command, name string) (time.Duration, error) {
	raw := flagString(cmd, name)
	ms, err := strconv.Atoi(raw)
	if err != nil || ms <= 0 {
		return 0, output.NewError("VALIDATION_ERROR", "无效的轮询间隔", map[string]any{
			"flag":  name,
			"value": raw,
			"hint":  "请传入一个大于 0 的整数毫秒值，例如 2000",
		})
	}
	return time.Duration(ms) * time.Millisecond, nil
}
