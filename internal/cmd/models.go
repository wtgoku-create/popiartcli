package cmd

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/input"
	"github.com/wtgoku-create/popiartcli/internal/output"
)

func newModelsCmd() *cobra.Command {
	modelsCmd := &cobra.Command{
		Use:   "models",
		Short: "查询模型库存、路由与直接推理能力",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "列出已注册的可用模型库存",
		Long:  "列出后端当前注册的模型库存。该命令显示的是可用模型清单，不等同于当前项目真正生效的路由结果。",
		RunE: func(cmd *cobra.Command, args []string) error {
			var models any
			if err := currentClient().GetJSON(context.Background(), "/models", map[string]string{
				"type":       flagString(cmd, "type"),
				"provider":   flagString(cmd, "provider"),
				"capability": flagString(cmd, "capability"),
			}, &models); err != nil {
				return err
			}
			return writeOutput(cmd, models)
		},
	}
	listCmd.Flags().String("type", "", "按模型类型过滤")
	listCmd.Flags().String("provider", "", "按供应商过滤")
	listCmd.Flags().String("capability", "", "按能力过滤，例如 text2image、img2img、image2video")

	routesCmd := &cobra.Command{
		Use:   "routes",
		Short: "查看当前生效的 route key 路由表",
		Long:  "显示当前项目真正生效的 route_key -> model_id 路由结果。它和 models list 的模型库存不是一回事。",
		RunE: func(cmd *cobra.Command, args []string) error {
			var routes any
			if err := currentClient().GetJSON(context.Background(), "/models/routes", map[string]string{
				"project_id": flagString(cmd, "project"),
				"route_key":  routeKeyFlagValue(cmd),
				"skill_type": legacyRouteKeyFlagValue(cmd),
			}, &routes); err != nil {
				return err
			}
			return writeOutput(cmd, routes)
		},
	}
	routesCmd.Flags().String("project", "", "按项目查看路由覆盖")
	addRouteKeyFlags(routesCmd)

	inferCmd := &cobra.Command{
		Use:   "infer <model-id>",
		Short: "直接提交模型推理任务",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateJobExecutionFlags(cmd); err != nil {
				return err
			}

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
			if dryRunMode(cmd) {
				return writeDryRunPreview(cmd, "models.infer", map[string]any{
					"model_id": args[0],
					"request": map[string]any{
						"method": "POST",
						"path":   "/models/infer",
						"body":   body,
					},
				})
			}

			var job map[string]any
			if err := currentClient().PostJSON(context.Background(), "/models/infer", body, &job); err != nil {
				return err
			}
			return writeJobResultOrWait(cmd, job)
		},
	}
	inferCmd.Flags().StringP("input", "i", "", "输入 JSON 字符串、@file.json，或用 - 表示标准输入")
	inferCmd.Flags().BoolP("wait", "w", false, "阻塞进程直到作业完成")
	inferCmd.Flags().String("interval", "2000", "轮询间隔（毫秒，默认：2000）")
	inferCmd.Flags().String("priority", "normal", "作业优先级: low | normal | high")
	inferCmd.Flags().String("idempotency-key", "", "用于安全重试的幂等键")

	overrideCmd := &cobra.Command{
		Use:   "route-override",
		Short: "管理项目级 route key 路由覆盖",
	}

	setCmd := &cobra.Command{
		Use:   "set",
		Short: "设置项目级 route key 覆盖",
		RunE: func(cmd *cobra.Command, args []string) error {
			routeKey, err := requiredRouteKey(cmd)
			if err != nil {
				return err
			}

			var resp any
			if err := currentClient().PostJSON(context.Background(), "/models/routes/overrides", map[string]any{
				"project_id": flagString(cmd, "project"),
				"route_key":  routeKey,
				"skill_type": routeKey,
				"model_id":   flagString(cmd, "model"),
			}, &resp); err != nil {
				return err
			}
			return writeOutput(cmd, resp)
		},
	}
	setCmd.Flags().String("project", "", "项目 ID")
	setCmd.Flags().String("model", "", "模型 ID")
	addRouteKeyFlags(setCmd)
	setCmd.MarkFlagRequired("project")
	setCmd.MarkFlagRequired("model")

	unsetCmd := &cobra.Command{
		Use:   "unset",
		Short: "删除项目级 route key 覆盖",
		RunE: func(cmd *cobra.Command, args []string) error {
			routeKey, err := requiredRouteKey(cmd)
			if err != nil {
				return err
			}

			var resp any
			if err := currentClient().PostJSON(context.Background(), "/models/routes/overrides/unset", map[string]any{
				"project_id": flagString(cmd, "project"),
				"route_key":  routeKey,
				"skill_type": routeKey,
			}, &resp); err != nil {
				return err
			}
			return writeOutput(cmd, resp)
		},
	}
	unsetCmd.Flags().String("project", "", "项目 ID")
	addRouteKeyFlags(unsetCmd)
	unsetCmd.MarkFlagRequired("project")

	overrideListCmd := &cobra.Command{
		Use:   "list",
		Short: "列出项目级 route key 覆盖",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp any
			if err := currentClient().GetJSON(context.Background(), "/models/routes/overrides", map[string]string{
				"project_id": flagString(cmd, "project"),
				"route_key":  routeKeyFlagValue(cmd),
				"skill_type": legacyRouteKeyFlagValue(cmd),
			}, &resp); err != nil {
				return err
			}
			return writeOutput(cmd, resp)
		},
	}
	overrideListCmd.Flags().String("project", "", "项目 ID")
	addRouteKeyFlags(overrideListCmd)
	overrideListCmd.MarkFlagRequired("project")

	overrideCmd.AddCommand(setCmd, unsetCmd, overrideListCmd)
	modelsCmd.AddCommand(listCmd, routesCmd, inferCmd, overrideCmd)
	return modelsCmd
}

func addRouteKeyFlags(cmd *cobra.Command) {
	cmd.Flags().String("route", "", "路由键，例如 image.text2image、image.img2img、video.image2video")
	cmd.Flags().String("skill-type", "", "已弃用：请改用 --route")
	_ = cmd.Flags().MarkDeprecated("skill-type", "请改用 --route")
}

func routeKeyFlagValue(cmd *cobra.Command) string {
	return strings.TrimSpace(flagString(cmd, "route"))
}

func legacyRouteKeyFlagValue(cmd *cobra.Command) string {
	return strings.TrimSpace(flagString(cmd, "skill-type"))
}

func requiredRouteKey(cmd *cobra.Command) (string, error) {
	routeKey := routeKeyFlagValue(cmd)
	legacy := legacyRouteKeyFlagValue(cmd)

	switch {
	case routeKey == "" && legacy == "":
		return "", output.NewError("VALIDATION_ERROR", "缺少路由键", map[string]any{
			"flag":  "route",
			"alias": "skill-type",
			"hint":  "请传入类似 image.text2image、image.img2img、video.image2video 的 route key",
		})
	case routeKey != "" && legacy != "" && routeKey != legacy:
		return "", output.NewError("VALIDATION_ERROR", "--route 与 --skill-type 不一致", map[string]any{
			"route":      routeKey,
			"skill_type": legacy,
			"hint":       "请只保留 --route，或保证两个值一致",
		})
	case routeKey != "":
		return routeKey, nil
	default:
		return legacy, nil
	}
}
