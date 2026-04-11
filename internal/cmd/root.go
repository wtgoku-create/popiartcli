package cmd

import (
	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/output"
)

func NewRootCmd(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "popiart",
		Short:         "为 Coding Agent 提供创作者技能入口与多模态授权计费的 CLI",
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := validateAgentProtocolFlags(cmd); err != nil {
				return output.NewError("VALIDATION_ERROR", "无效的 agent 协议参数", map[string]any{
					"details": err.Error(),
					"hint":    "请使用 --output json 或 --output plain",
				})
			}
			if !shouldPersistGlobalOverrides(cmd) {
				return nil
			}

			endpointChanged := cmd.Flags().Changed("endpoint")
			projectChanged := cmd.Flags().Changed("project")
			if !endpointChanged && !projectChanged {
				return nil
			}

			var patch config.Patch
			if endpointChanged {
				value, _ := cmd.Flags().GetString("endpoint")
				patch.Endpoint = &value
			}
			if projectChanged {
				value, _ := cmd.Flags().GetString("project")
				patch.Project = &value
			}
			_, err := config.SavePatch(patch)
			if err != nil {
				return output.NewError("CLI_ERROR", "保存全局配置失败", map[string]any{
					"details": err.Error(),
				})
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().String("endpoint", "", "覆盖本次调用的 API 端点")
	rootCmd.PersistentFlags().String("project", "", "覆盖本次调用的活动项目")
	rootCmd.PersistentFlags().String("output", "json", "输出格式: json | plain（默认：json）")
	rootCmd.PersistentFlags().Bool("plain", false, "人类可读的输出（默认：JSON）")
	rootCmd.PersistentFlags().Bool("non-interactive", false, "在缺少参数时直接失败，不进入交互式提示")
	rootCmd.PersistentFlags().Bool("quiet", false, "抑制非结果输出，便于 agent / CI 管道")
	rootCmd.PersistentFlags().Bool("yes", false, "自动接受确认提示")
	rootCmd.PersistentFlags().Bool("dry-run", false, "预览规范化后的请求，而不执行网络写操作")
	rootCmd.PersistentFlags().Bool("async", false, "显式要求立即返回 job，而不是阻塞等待")
	rootCmd.PersistentFlags().Bool("no-color", false, "在纯文本输出中禁用 ANSI 颜色")

	rootCmd.AddCommand(
		newImageCmd(),
		newVideoCmd(),
		newAudioCmd(),
		newSetupCmd(),
		newAuthCmd(),
		newSkillsCmd(),
		newRunCmd(),
		newJobsCmd(),
		newArtifactsCmd(),
		newMediaCmd(),
		newBudgetCmd(),
		newProjectCmd(),
		newModelsCmd(),
		newMCPCmd(),
		newBootstrapCmd(),
		newCompletionCmd(),
		newUpdateCmd(),
	)

	return rootCmd
}

func shouldPersistGlobalOverrides(cmd *cobra.Command) bool {
	if cmd == nil {
		return true
	}

	return cmd.Name() != "update"
}
