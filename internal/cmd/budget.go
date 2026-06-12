package cmd

import (
	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/output"
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
			return output.NewError("UNSUPPORTED_IN_POPI_ART_MODE", "当前模式不支持通过 CLI 查询预算摘要", map[string]any{
				"hint": "budget 仍依赖旧后端接口；请在网站中查看消耗与余额，或等待主站预算 API 迁移完成",
			})
		},
	}
	statusCmd.Flags().String("project", "", "限定到特定项目")

	usageCmd := &cobra.Command{
		Use:   "usage",
		Short: "按技能和时间段进行详细的使用情况细分",
		RunE: func(cmd *cobra.Command, args []string) error {
			return output.NewError("UNSUPPORTED_IN_POPI_ART_MODE", "当前模式不支持通过 CLI 查询预算使用明细", map[string]any{
				"hint": "budget 仍依赖旧后端接口；请在网站中查看消耗与余额，或等待主站预算 API 迁移完成",
			})
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
			return output.NewError("UNSUPPORTED_IN_POPI_ART_MODE", "当前模式不支持通过 CLI 查询预算限制", map[string]any{
				"hint": "budget 仍依赖旧后端接口；请在网站中查看配额信息，或等待主站预算 API 迁移完成",
			})
		},
	}

	budgetCmd.AddCommand(statusCmd, usageCmd, limitsCmd)
	return budgetCmd
}
