package cmd

import (
	"context"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/input"
	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/popiart"
)

func newModelsCmd() *cobra.Command {
	modelsCmd := &cobra.Command{
		Use:   "models",
		Short: "查询模型库存、路由与直接推理能力",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "列出已注册的可用模型库存",
		Long:  "列出主站当前可见的模型能力清单。该命令显示的是模型能力源，不等同于最终命令执行时的默认模型选择结果。",
		RunE: func(cmd *cobra.Command, args []string) error {
			models, err := popiart.FetchModels(context.Background(), currentClient())
			if err != nil {
				return err
			}
			filtered := filterModelsForList(models, flagString(cmd, "capability"))
			if len(strings.TrimSpace(flagString(cmd, "type"))) > 0 {
				filtered = filterModelsForList(filtered, flagString(cmd, "type"))
			}
			if len(strings.TrimSpace(flagString(cmd, "provider"))) > 0 {
				filtered = filterModelsForList(filtered, flagString(cmd, "provider"))
			}
			if err := writeOutput(cmd, map[string]any{
				"items":  summarizeModels(filtered),
				"total":  len(filtered),
				"source": "/api_client/anime/ai/model/list",
			}); err != nil {
				return err
			}
			return nil
		},
	}
	listCmd.Flags().String("type", "", "按模型类型过滤")
	listCmd.Flags().String("provider", "", "按供应商过滤")
	listCmd.Flags().String("capability", "", "按能力过滤，例如 text2image、img2img、image2video")

	routesCmd := &cobra.Command{
		Use:   "routes",
		Short: "查看当前默认模型选择结果",
		Long:  "显示 CLI 当前命令入口对应的默认 aiModelCode，以及基于主站模型列表解析出的 aiModelId 和支持子类型。",
		RunE: func(cmd *cobra.Command, args []string) error {
			models, err := popiart.FetchModels(context.Background(), currentClient())
			if err != nil {
				return err
			}
			items, err := resolveRouteSummaries(models, routeKeyFlagValue(cmd), legacyRouteKeyFlagValue(cmd))
			if err != nil {
				return err
			}
			return writeOutput(cmd, map[string]any{
				"items":  items,
				"total":  len(items),
				"source": "/api_client/anime/ai/model/list",
			})
		},
	}
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

type routeModelView struct {
	Command            string   `json:"command"`
	DefaultAIModelCode string   `json:"default_ai_model_code"`
	ResolvedAIModelID  string   `json:"resolved_ai_model_id,omitempty"`
	SupportedSubTypes  []int    `json:"supported_sub_types,omitempty"`
	SelectedBy         string   `json:"selected_by"`
	Aliases            []string `json:"aliases,omitempty"`
}

type defaultRouteMapping struct {
	RouteKey string
	Command  string
}

// filterModelsForList 先满足迁移期最常见的 capability 过滤需求，其余过滤词走宽松匹配。
func filterModelsForList(models []popiart.Model, filter string) []popiart.Model {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return append([]popiart.Model(nil), models...)
	}

	filtered := make([]popiart.Model, 0, len(models))
	for _, model := range models {
		if modelMatchesFilter(model, filter) {
			filtered = append(filtered, model)
		}
	}
	return filtered
}

// summarizeModels 把主站模型对象压缩成更适合 CLI 输出的摘要结构。
func summarizeModels(models []popiart.Model) []map[string]any {
	items := make([]map[string]any, 0, len(models))
	for _, model := range models {
		item := map[string]any{
			"id":                  model.ID,
			"supported_sub_types": modelSubTypes(model),
			"is_support_images":   model.IsSupportImages,
			"is_support_videos":   model.IsSupportVideos,
			"is_support_audios":   model.IsSupportAudios,
		}
		if len(model.Ratio) > 0 {
			item["ratio"] = []string(model.Ratio)
		}
		if len(model.VideoRatio) > 0 {
			item["video_ratio"] = []string(model.VideoRatio)
		}
		if len(model.Resolution) > 0 {
			item["resolution"] = []string(model.Resolution)
		}
		if len(model.Duration) > 0 {
			item["duration"] = []int(model.Duration)
		}
		if model.UploadImageLimit.Value != nil {
			item["upload_image_limit"] = *model.UploadImageLimit.Value
		}
		if model.UploadVideoLimit.Value != nil {
			item["upload_video_limit"] = *model.UploadVideoLimit.Value
		}
		if model.UploadAudioLimit.Value != nil {
			item["upload_audio_limit"] = *model.UploadAudioLimit.Value
		}
		items = append(items, item)
	}
	return items
}

// resolveRouteSummaries 输出迁移文档要求的“默认模型选择结果”视图。
func resolveRouteSummaries(models []popiart.Model, routeKey, legacyRouteKey string) ([]routeModelView, error) {
	requested := strings.TrimSpace(routeKey)
	if requested == "" {
		requested = strings.TrimSpace(legacyRouteKey)
	}

	mappings := defaultRouteMappings()
	items := make([]routeModelView, 0, len(mappings))
	for _, mapping := range mappings {
		if requested != "" && mapping.RouteKey != requested {
			continue
		}

		candidates := popiart.DefaultModelCodes(mapping.Command)
		item := routeModelView{
			Command:    mapping.RouteKey,
			SelectedBy: "default",
		}
		if len(candidates) > 0 {
			item.DefaultAIModelCode = candidates[0]
			item.Aliases = append([]string(nil), candidates...)
		}
		if len(candidates) > 0 {
			if model, ok := popiart.ResolveModelByCode(models, candidates[0]); ok {
				item.ResolvedAIModelID = strconv.FormatInt(model.ID, 10)
				item.SupportedSubTypes = modelSubTypes(model)
			}
		}
		items = append(items, item)
	}

	if requested != "" && len(items) == 0 {
		return nil, output.NewError("NOT_FOUND", "未找到对应 route 的默认模型映射", map[string]any{
			"route": requested,
		})
	}
	return items, nil
}

func defaultRouteMappings() []defaultRouteMapping {
	return []defaultRouteMapping{
		{RouteKey: "video.seedance", Command: "video.seedance"},
		{RouteKey: "video.action-transfer", Command: "video.action-transfer"},
		{RouteKey: "audio.tts", Command: "audio.tts"},
		{RouteKey: "speech.synthesize", Command: "speech.synthesize"},
		{RouteKey: "music.generate", Command: "music.generate"},
	}
}

func modelMatchesFilter(model popiart.Model, filter string) bool {
	switch strings.ToLower(filter) {
	case "text2image":
		return hasSubType(model, 103)
	case "img2img":
		return hasSubType(model, 103)
	case "image2video":
		return hasSubType(model, 202) || hasSubType(model, 203) || hasSubType(model, 204) || hasSubType(model, 205)
	case "tts", "text2speech":
		return hasSubType(model, 301)
	case "multimodal", "image-describe":
		return hasSubType(model, 501)
	}

	filter = strings.ToLower(strings.TrimSpace(filter))
	if strings.Contains(strings.ToLower(model.Code), filter) || strings.Contains(strings.ToLower(model.Name), filter) {
		return true
	}
	for _, alias := range model.AIModelCodeAlias {
		if strings.Contains(strings.ToLower(alias), filter) {
			return true
		}
	}
	return false
}

func hasSubType(model popiart.Model, subType int) bool {
	return popiart.SupportsSubType(model, subType)
}

func modelSubTypes(model popiart.Model) []int {
	values := make([]int, 0, len(model.Categories))
	for _, category := range model.Categories {
		if category.TaskSubType == 0 {
			continue
		}
		values = append(values, category.TaskSubType)
	}
	return values
}
