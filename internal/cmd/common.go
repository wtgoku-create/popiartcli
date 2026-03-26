package cmd

import (
	"bufio"
	"fmt"
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

func plainOutput(cmd *cobra.Command) bool {
	value, _ := cmd.Flags().GetBool("plain")
	return value
}

func writeOutput(cmd *cobra.Command, data any) error {
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
	fmt.Fprint(os.Stderr, label)
	return readPromptLine()
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
	fmt.Fprint(os.Stderr, label)
	if termutil.IsTerminal(int(os.Stdin.Fd())) {
		value, err := termutil.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err == nil {
			return strings.TrimSpace(string(value)), nil
		}
	}
	return readPromptLine()
}

func requireTokenError() error {
	return output.NewError("UNAUTHENTICATED", "没有可用 key。请运行: popiart auth login", nil)
}
