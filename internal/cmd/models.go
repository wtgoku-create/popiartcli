package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/input"
	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/poll"
)

func newModelsCmd() *cobra.Command {
	modelsCmd := &cobra.Command{
		Use:   "models",
		Short: "查询模型路由与直接推理能力",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "列出可用模型",
		RunE: func(cmd *cobra.Command, args []string) error {
			var models any
			if err := currentClient().GetJSON(context.Background(), "/models", map[string]string{
				"type":     flagString(cmd, "type"),
				"provider": flagString(cmd, "provider"),
			}, &models); err != nil {
				return err
			}
			return writeOutput(cmd, models)
		},
	}
	listCmd.Flags().String("type", "", "按模型类型过滤")
	listCmd.Flags().String("provider", "", "按供应商过滤")

	routesCmd := &cobra.Command{
		Use:   "routes",
		Short: "查看当前生效的模型路由表",
		RunE: func(cmd *cobra.Command, args []string) error {
			var routes any
			if err := currentClient().GetJSON(context.Background(), "/models/routes", map[string]string{
				"project_id": flagString(cmd, "project"),
			}, &routes); err != nil {
				return err
			}
			return writeOutput(cmd, routes)
		},
	}
	routesCmd.Flags().String("project", "", "按项目查看路由覆盖")

	inferCmd := &cobra.Command{
		Use:   "infer <model-id>",
		Short: "直接提交模型推理任务",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := input.Resolve(flagString(cmd, "input"))
			if err != nil {
				return err
			}

			cfg := config.Load()
			body := map[string]any{
				"model_id": args[0],
				"input":    payload,
				"priority": flagString(cmd, "priority"),
			}
			if cfg.Project != "" {
				body["project_id"] = cfg.Project
			}
			if value := flagString(cmd, "idempotency-key"); value != "" {
				body["idempotency_key"] = value
			}

			var job map[string]any
			if err := currentClient().PostJSON(context.Background(), "/models/infer", body, &job); err != nil {
				return err
			}

			if !flagBool(cmd, "wait") {
				return writeOutput(cmd, job)
			}

			jobID := stringValue(job["job_id"])
			if jobID == "" {
				return output.NewError("CLI_ERROR", "推理响应中缺少 job_id", nil)
			}

			interval, err := intervalDuration(cmd, "interval")
			if err != nil {
				return err
			}
			done, err := poll.WaitForJob(context.Background(), currentClient(), jobID, interval, 300)
			if err != nil {
				return err
			}
			return writeOutput(cmd, done)
		},
	}
	inferCmd.Flags().StringP("input", "i", "", "输入 JSON 字符串、@file.json，或用 - 表示标准输入")
	inferCmd.Flags().BoolP("wait", "w", false, "阻塞进程直到作业完成")
	inferCmd.Flags().String("interval", "2000", "轮询间隔（毫秒，默认：2000）")
	inferCmd.Flags().String("priority", "normal", "作业优先级: low | normal | high")
	inferCmd.Flags().String("idempotency-key", "", "用于安全重试的幂等键")

	overrideCmd := &cobra.Command{
		Use:   "route-override",
		Short: "管理项目级模型路由覆盖",
	}

	setCmd := &cobra.Command{
		Use:   "set",
		Short: "设置项目级路由覆盖",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp any
			if err := currentClient().PostJSON(context.Background(), "/models/routes/overrides", map[string]any{
				"project_id": flagString(cmd, "project"),
				"skill_type": flagString(cmd, "skill-type"),
				"model_id":   flagString(cmd, "model"),
			}, &resp); err != nil {
				return err
			}
			return writeOutput(cmd, resp)
		},
	}
	setCmd.Flags().String("project", "", "项目 ID")
	setCmd.Flags().String("skill-type", "", "技能类型")
	setCmd.Flags().String("model", "", "模型 ID")
	setCmd.MarkFlagRequired("project")
	setCmd.MarkFlagRequired("skill-type")
	setCmd.MarkFlagRequired("model")

	unsetCmd := &cobra.Command{
		Use:   "unset",
		Short: "删除项目级路由覆盖",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp any
			if err := currentClient().PostJSON(context.Background(), "/models/routes/overrides/unset", map[string]any{
				"project_id": flagString(cmd, "project"),
				"skill_type": flagString(cmd, "skill-type"),
			}, &resp); err != nil {
				return err
			}
			return writeOutput(cmd, resp)
		},
	}
	unsetCmd.Flags().String("project", "", "项目 ID")
	unsetCmd.Flags().String("skill-type", "", "技能类型")
	unsetCmd.MarkFlagRequired("project")
	unsetCmd.MarkFlagRequired("skill-type")

	overrideListCmd := &cobra.Command{
		Use:   "list",
		Short: "列出项目级路由覆盖",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp any
			if err := currentClient().GetJSON(context.Background(), "/models/routes/overrides", map[string]string{
				"project_id": flagString(cmd, "project"),
			}, &resp); err != nil {
				return err
			}
			return writeOutput(cmd, resp)
		},
	}
	overrideListCmd.Flags().String("project", "", "项目 ID")
	overrideListCmd.MarkFlagRequired("project")

	overrideCmd.AddCommand(setCmd, unsetCmd, overrideListCmd)
	modelsCmd.AddCommand(listCmd, routesCmd, inferCmd, overrideCmd)
	return modelsCmd
}
