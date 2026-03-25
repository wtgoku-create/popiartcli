package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

func newBudgetCmd() *cobra.Command {
	budgetCmd := &cobra.Command{
		Use:   "budget",
		Short: "查看令牌使用情况和剩余预算",
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "显示当前周期的预算和使用情况摘要",
		RunE: func(cmd *cobra.Command, args []string) error {
			var budget any
			if err := currentClient().GetJSON(context.Background(), "/budget", map[string]string{
				"project_id": flagString(cmd, "project"),
			}, &budget); err != nil {
				return err
			}
			return writeOutput(cmd, budget)
		},
	}
	statusCmd.Flags().String("project", "", "限定到特定项目")

	usageCmd := &cobra.Command{
		Use:   "usage",
		Short: "按技能和时间段进行详细的使用情况细分",
		RunE: func(cmd *cobra.Command, args []string) error {
			var usage any
			if err := currentClient().GetJSON(context.Background(), "/budget/usage", map[string]string{
				"since":      flagString(cmd, "since"),
				"until":      flagString(cmd, "until"),
				"group_by":   flagString(cmd, "group-by"),
				"project_id": flagString(cmd, "project"),
			}, &usage); err != nil {
				return err
			}
			return writeOutput(cmd, usage)
		},
	}
	usageCmd.Flags().String("since", "", "开始日期 (ISO 8601)")
	usageCmd.Flags().String("until", "", "结束日期 (ISO 8601，默认：当前时间)")
	usageCmd.Flags().String("group-by", "skill", "分组方式: skill|day|project")
	usageCmd.Flags().String("project", "", "限定到特定项目")

	limitsCmd := &cobra.Command{
		Use:   "limits",
		Short: "显示速率限制和配额配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			var limits any
			if err := currentClient().GetJSON(context.Background(), "/budget/limits", nil, &limits); err != nil {
				return err
			}
			return writeOutput(cmd, limits)
		},
	}

	budgetCmd.AddCommand(statusCmd, usageCmd, limitsCmd)
	return budgetCmd
}
