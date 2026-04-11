package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/api"
	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/termutil"
)

func currentClient() *api.Client {
	cfg := config.Load()
	return api.NewClient(cfg.Endpoint, cfg.Token)
}

func writeOutput(cmd *cobra.Command, data any) error {
	if err := validateAgentProtocolFlags(cmd); err != nil {
		return output.NewError("VALIDATION_ERROR", "无效的 agent 协议参数", map[string]any{
			"details": err.Error(),
			"flag":    "output",
			"hint":    "请使用 --output json 或 --output plain",
		})
	}
	return output.WriteData(cmd.OutOrStdout(), data, plainOutput(cmd))
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

func prompt(label string) (string, error) {
	return promptTo(os.Stderr, label)
}

func readPromptLine() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func promptPassword(label string) (string, error) {
	return promptPasswordTo(os.Stderr, label)
}

func promptTo(w io.Writer, label string) (string, error) {
	fmt.Fprint(w, label)
	return readPromptLine()
}

func promptPasswordTo(w io.Writer, label string) (string, error) {
	fmt.Fprint(w, label)
	if termutil.IsTerminal(int(os.Stdin.Fd())) {
		value, err := termutil.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(w)
		if err == nil {
			return strings.TrimSpace(string(value)), nil
		}
	}
	return readPromptLine()
}

func requireTokenError() error {
	return output.NewError("UNAUTHENTICATED", "没有可用 key。请运行: popiart auth login", nil)
}
