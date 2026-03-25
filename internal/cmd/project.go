package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/output"
)

func newProjectCmd() *cobra.Command {
	projectCmd := &cobra.Command{
		Use:   "project",
		Short: "读取并管理当前项目上下文",
	}

	currentCmd := &cobra.Command{
		Use:   "current",
		Short: "显示当前活动项目",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := config.Load().Project
			if projectID == "" {
				return writeOutput(cmd, map[string]any{
					"project": nil,
					"hint":    "使用以下命令设置: popiart project use <project-id>",
				})
			}
			var project any
			if err := currentClient().GetJSON(context.Background(), "/projects/"+projectID, nil, &project); err != nil {
				return err
			}
			return writeOutput(cmd, project)
		},
	}

	useCmd := &cobra.Command{
		Use:   "use <project-id>",
		Short: "设置活动项目（存储在配置中）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var project map[string]any
			if err := currentClient().GetJSON(context.Background(), "/projects/"+args[0], nil, &project); err != nil {
				return err
			}
			if _, err := config.SavePatch(config.Patch{Project: &args[0]}); err != nil {
				return output.NewError("CLI_ERROR", "保存项目失败", map[string]any{"details": err.Error()})
			}
			return writeOutput(cmd, map[string]any{
				"project_set": args[0],
				"name":        project["name"],
			})
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "列出可访问的项目",
		RunE: func(cmd *cobra.Command, args []string) error {
			var projects any
			if err := currentClient().GetJSON(context.Background(), "/projects", map[string]string{
				"limit": flagString(cmd, "limit"),
			}, &projects); err != nil {
				return err
			}
			return writeOutput(cmd, projects)
		},
	}
	listCmd.Flags().String("limit", "20", "最大结果数量")

	getCmd := &cobra.Command{
		Use:   "get <project-id>",
		Short: "获取项目的完整上下文",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var project any
			if err := currentClient().GetJSON(context.Background(), "/projects/"+args[0], nil, &project); err != nil {
				return err
			}
			return writeOutput(cmd, project)
		},
	}

	contextCmd := &cobra.Command{
		Use:   "context",
		Short: "获取活动项目的完整运行时上下文",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := flagString(cmd, "project")
			if projectID == "" {
				projectID = config.Load().Project
			}
			if projectID == "" {
				return output.NewError("NO_PROJECT", "未设置项目。请使用: popiart project use <id>", nil)
			}

			var ctx any
			if err := currentClient().GetJSON(context.Background(), "/projects/"+projectID+"/context", nil, &ctx); err != nil {
				return err
			}
			return writeOutput(cmd, ctx)
		},
	}
	contextCmd.Flags().String("project", "", "覆盖活动项目")

	projectCmd.AddCommand(currentCmd, useCmd, listCmd, getCmd, contextCmd)
	return projectCmd
}
