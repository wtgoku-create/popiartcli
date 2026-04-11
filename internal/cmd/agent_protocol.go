package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/output"
)

const (
	outputModeJSON  = "json"
	outputModePlain = "plain"
)

func validateAgentProtocolFlags(cmd *cobra.Command) error {
	_, err := outputMode(cmd)
	return err
}

func outputMode(cmd *cobra.Command) (string, error) {
	if cmd == nil {
		return outputModeJSON, nil
	}

	mode := strings.ToLower(strings.TrimSpace(flagString(cmd, "output")))
	if mode == "" {
		mode = outputModeJSON
	}
	if flagBool(cmd, "plain") {
		if mode != outputModeJSON && mode != outputModePlain {
			return "", invalidOutputModeError(mode)
		}
		mode = outputModePlain
	}

	switch mode {
	case outputModeJSON, outputModePlain:
		return mode, nil
	default:
		return "", invalidOutputModeError(mode)
	}
}

func plainOutput(cmd *cobra.Command) bool {
	mode, err := outputMode(cmd)
	if err != nil {
		return false
	}
	return mode == outputModePlain
}

func invalidOutputModeError(mode string) error {
	return fmt.Errorf("unsupported output mode %q", mode)
}

func invalidFlagValueError(flag, value, hint string) error {
	return output.NewError("VALIDATION_ERROR", "无效的命令参数", map[string]any{
		"flag":  flag,
		"value": value,
		"hint":  hint,
	})
}

func nonInteractiveMode(cmd *cobra.Command) bool {
	return flagBool(cmd, "non-interactive")
}

func quietMode(cmd *cobra.Command) bool {
	return flagBool(cmd, "quiet")
}

func autoApproveMode(cmd *cobra.Command) bool {
	return flagBool(cmd, "yes")
}

func dryRunMode(cmd *cobra.Command) bool {
	return flagBool(cmd, "dry-run")
}

func shouldWaitForJob(cmd *cobra.Command) (bool, error) {
	waitFlag := cmd.Flags().Lookup("wait")
	if waitFlag == nil {
		return false, nil
	}

	wait := flagBool(cmd, "wait")
	async := flagBool(cmd, "async")
	if wait && async {
		return false, conflictingAgentFlagsError("wait", "async")
	}
	if async {
		return false, nil
	}
	return wait, nil
}

func validateJobExecutionFlags(cmd *cobra.Command) error {
	_, err := shouldWaitForJob(cmd)
	return err
}

func conflictingAgentFlagsError(left, right string) error {
	return invalidFlagValueError("--"+left+" and --"+right, "cannot be combined", "choose blocking or async mode, not both")
}

func agentProtocolPreview(cmd *cobra.Command) (map[string]any, error) {
	mode, err := outputMode(cmd)
	if err != nil {
		return nil, err
	}
	protocol := map[string]any{
		"command":         cmd.CommandPath(),
		"output":          mode,
		"non_interactive": nonInteractiveMode(cmd),
		"quiet":           quietMode(cmd),
		"yes":             autoApproveMode(cmd),
		"dry_run":         dryRunMode(cmd),
	}

	if cmd.Flags().Lookup("wait") == nil {
		return protocol, nil
	}

	wait, err := shouldWaitForJob(cmd)
	if err != nil {
		return nil, err
	}
	protocol["wait"] = wait
	protocol["async"] = !wait
	return protocol, nil
}

func writeDryRunPreview(cmd *cobra.Command, action string, preview map[string]any) error {
	protocol, err := agentProtocolPreview(cmd)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"dry_run":        true,
		"action":         action,
		"agent_protocol": protocol,
	}
	for key, value := range preview {
		payload[key] = value
	}
	return writeOutput(cmd, payload)
}
