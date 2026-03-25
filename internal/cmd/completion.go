package cmd

import (
	"fmt"
	"slices"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/output"
)

func newCompletionCmd() *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "completion <bash|zsh|fish|powershell>",
		Short: "生成 shell completion 脚本",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := args[0]
			if _, ok := supportedCompletionShells[shell]; !ok {
				supported := make([]string, 0, len(supportedCompletionShells))
				for name := range supportedCompletionShells {
					supported = append(supported, name)
				}
				slices.Sort(supported)
				return output.NewError("VALIDATION_ERROR", "不支持的 shell", map[string]any{
					"shell":     shell,
					"supported": supported,
				})
			}

			switch shell {
			case "bash":
				return cmd.Root().GenBashCompletionV2(cmd.OutOrStdout(), true)
			case "zsh":
				return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return fmt.Errorf("unsupported shell: %s", shell)
			}
		},
	}

	return completionCmd
}
