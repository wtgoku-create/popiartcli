package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/types"
)

const (
	popiartMCPServerName = "PopiArt"
	popiartMCPServerID   = "popiart"
)

type officialRuntimeSkill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ModelType   string `json:"model_type"`
}

type mcpTool struct {
	Name         string         `json:"name"`
	Title        string         `json:"title,omitempty"`
	Description  string         `json:"description"`
	InputSchema  map[string]any `json:"input_schema,omitempty"`
	OutputSchema map[string]any `json:"output_schema,omitempty"`
	Annotations  map[string]any `json:"annotations,omitempty"`
}

type doctorCheck struct {
	ID      string         `json:"id"`
	Status  string         `json:"status"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func newMCPCmd() *cobra.Command {
	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "生成 PopiArt 的 MCP discoverability 配置与诊断信息",
	}

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "启动 PopiArt 的 stdio MCP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagBool(cmd, "describe") {
				cfg := config.Load()
				return writeOutput(cmd, map[string]any{
					"server_name": popiartMCPServerName,
					"server_id":   popiartMCPServerID,
					"transport":   "stdio",
					"status":      "implemented",
					"endpoint":    cfg.Endpoint,
					"project":     cfg.Project,
					"tools":       mcpTools(),
				})
			}
			return runMCPServer(cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), cmd.Root().Version)
		},
	}
	serveCmd.Flags().Bool("describe", false, "打印 server 元数据而不是启动 stdio transport")

	printConfigCmd := &cobra.Command{
		Use:   "print-config",
		Short: "打印通用的 PopiArt MCP server 配置片段",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			agent := flagString(cmd, "agent")
			snippet := buildAgentMCPConfig(agent, cfg)
			payload := map[string]any{
				"agent":                    agent,
				"server_name":              popiartMCPServerName,
				"server_id":                popiartMCPServerID,
				"mcp_server_config":        snippet,
				"official_runtime_skills":  officialRuntimeSkills(),
				"bootstrap_asset_location": filepath.Join(config.Dir(), "agents", agent),
			}
			if agent != "" {
				if paths, err := resolveNativeAgentPaths(agent); err == nil {
					payload["native_mcp_config_path"] = paths.MCPConfigPath
					payload["native_skill_dir"] = paths.SkillDir
				}
			}
			return writeOutput(cmd, payload)
		},
	}
	printConfigCmd.Flags().String("agent", "", "目标 agent 名称，可选")

	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "检查 PopiArt 的 discoverability 与 runtime baseline 状态",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			agent := flagString(cmd, "agent")

			checks := []doctorCheck{
				passCheck("config_dir", "PopiArt 配置目录可用", map[string]any{
					"path": config.Dir(),
				}),
				passCheck("endpoint", "API endpoint 已配置", map[string]any{
					"endpoint": cfg.Endpoint,
				}),
			}

			if cfg.Token == "" {
				checks = append(checks, failCheck("key", "缺少可用 key", map[string]any{
					"hint": "先运行 `popiart auth login` 或注入 `POPIART_KEY`",
				}))
			} else {
				checks = append(checks, passCheck("key", "检测到本地 key", map[string]any{
					"source": "config or environment",
				}))
			}

			client := currentClient()
			checks = append(checks, runDoctorAPICheck(client, cfg.Token != "", "auth_me", "验证当前登录会话", func(ctx context.Context) error {
				var resp any
				return client.GetJSON(ctx, "/auth/me", nil, &resp)
			}))
			checks = append(checks, runDoctorAPICheck(client, true, "skills_api", "技能列表 API 可访问", func(ctx context.Context) error {
				var resp any
				return client.GetJSON(ctx, "/skills", map[string]string{"limit": "1"}, &resp)
			}))
			for _, skill := range officialRuntimeSkills() {
				skill := skill
				checks = append(checks, runDoctorAPICheck(client, true, "runtime_skill:"+skill.ID, "检测官方 runtime skill "+skill.ID, func(ctx context.Context) error {
					var resp types.Skill
					if err := client.GetJSON(ctx, "/skills/"+skill.ID, nil, &resp); err != nil {
						return err
					}
					if isOfficialRuntimePlaceholderSkill(resp) {
						return output.NewError("RUNTIME_SKILL_PLACEHOLDER", "官方 runtime skill 仍是占位符", map[string]any{
							"skill_id":    skill.ID,
							"description": resp.Description,
							"hint":        officialRuntimePlaceholderHint(skill.ID),
						})
					}
					return nil
				}))
			}
			checks = append(checks, runDoctorAPICheck(client, true, "model_routes", "模型路由表 API 可访问", func(ctx context.Context) error {
				var resp any
				return client.GetJSON(ctx, "/models/routes", nil, &resp)
			}))

			if agent != "" {
				base := filepath.Join(config.Dir(), "agents", agent)
				checks = append(checks, checkFileExists("agent_env_sh", filepath.Join(base, "env.sh"), "agent shell 环境文件"))
				checks = append(checks, checkFileExists("agent_env_ps1", filepath.Join(base, "env.ps1"), "agent PowerShell 环境文件"))
				checks = append(checks, checkFileExists("agent_mcp_config", filepath.Join(base, "mcp.json"), "agent MCP 配置片段"))
				checks = append(checks, checkFileExists("agent_skill_wrapper", filepath.Join(base, "SKILL.md"), "agent skill wrapper"))
				if paths, err := resolveNativeAgentPaths(agent); err != nil {
					checks = append(checks, failCheck("agent_native_paths", "解析 agent 原生目录失败", map[string]any{
						"agent":   agent,
						"details": err.Error(),
					}))
				} else {
					checks = append(checks, checkFileExists("agent_native_mcp_config", paths.MCPConfigPath, "agent 原生 MCP 配置"))
					checks = append(checks, checkFileExists("agent_native_skill_wrapper", filepath.Join(paths.SkillDir, popiartMCPServerID, "SKILL.md"), "agent 原生 skill wrapper"))
				}
			}

			status := "pass"
			for _, check := range checks {
				if check.Status == "fail" {
					status = "fail"
					break
				}
				if check.Status == "warn" {
					status = "warn"
				}
			}

			discoverabilityStatus := doctorStatusForChecks(checks, isDiscoverabilityDoctorCheck)
			runtimeStatus := doctorStatusForChecks(checks, isRuntimeDoctorCheck)

			return writeOutput(cmd, map[string]any{
				"server_name":             popiartMCPServerName,
				"server_id":               popiartMCPServerID,
				"overall_status":          status,
				"discoverability_status":  discoverabilityStatus,
				"runtime_status":          runtimeStatus,
				"status_hint":             "discoverability_status 代表本地 agent 是否能发现 PopiArt；runtime_status 代表远端 baseline skill 与路由是否更接近可执行",
				"agent":                   agent,
				"official_runtime_skills": officialRuntimeSkills(),
				"checks":                  checks,
			})
		},
	}
	doctorCmd.Flags().String("agent", "", "同时检查指定 agent 的 bootstrap 资产")

	mcpCmd.AddCommand(serveCmd, printConfigCmd, doctorCmd)
	return mcpCmd
}

func officialRuntimeSkills() []officialRuntimeSkill {
	items := make([]officialRuntimeSkill, 0, len(officialRuntimeSkillIDs))
	for _, skillID := range officialRuntimeSkillIDs {
		summary, ok := officialRuntimeSkillSummaryForID(skillID)
		if !ok {
			continue
		}
		items = append(items, officialRuntimeSkill{
			ID:          summary.ID,
			Name:        summary.Name,
			Description: summary.Description,
			ModelType:   summary.ModelType,
		})
	}
	return items
}

func mcpTools() []mcpTool {
	defs := mcpToolDefinitions()
	tools := make([]mcpTool, 0, len(defs))
	for _, def := range defs {
		tools = append(tools, mcpTool{
			Name:         def.Name,
			Title:        def.Title,
			Description:  def.Description,
			InputSchema:  def.InputSchema,
			OutputSchema: def.OutputSchema,
			Annotations:  def.Annotations,
		})
	}
	return tools
}

func buildAgentMCPConfig(agent string, cfg config.Config) map[string]any {
	env := map[string]any{}
	for key, value := range buildMCPEnvMap(cfg) {
		env[key] = value
	}

	return map[string]any{
		"name":      popiartMCPServerName,
		"id":        popiartMCPServerID,
		"agent":     agent,
		"transport": "stdio",
		"command":   "popiart",
		"args":      []string{"mcp", "serve"},
		"env":       env,
	}
}

func runDoctorAPICheck(client interface {
	GetJSON(context.Context, string, map[string]string, any) error
}, allowed bool, id, message string, fn func(context.Context) error) doctorCheck {
	if !allowed {
		return warnCheck(id, message, map[string]any{
			"hint": "当前缺少必要的本地认证信息，跳过远端检查",
		})
	}
	if err := fn(context.Background()); err != nil {
		cliErr, ok := err.(*output.CLIError)
		if ok {
			details := map[string]any{
				"code": cliErr.Code,
			}
			for key, value := range cliErr.Details {
				details[key] = value
			}
			return failCheck(id, message, details)
		}
		return failCheck(id, message, map[string]any{
			"details": err.Error(),
		})
	}
	return passCheck(id, message, nil)
}

func checkFileExists(id, path, message string) doctorCheck {
	if _, err := filepath.Abs(path); err != nil {
		return failCheck(id, message, map[string]any{
			"path":    path,
			"details": err.Error(),
		})
	}
	if _, err := os.Stat(path); err != nil {
		return failCheck(id, message, map[string]any{
			"path": path,
			"hint": "先运行 `popiart setup --agent <agent>` 或 `popiart bootstrap --agent <agent> --discoverable`",
		})
	}
	return passCheck(id, message, map[string]any{
		"path": path,
	})
}

func doctorStatusForChecks(checks []doctorCheck, include func(string) bool) string {
	status := "pass"
	found := false
	for _, check := range checks {
		if !include(check.ID) {
			continue
		}
		found = true
		if check.Status == "fail" {
			return "fail"
		}
		if check.Status == "warn" {
			status = "warn"
		}
	}
	if !found {
		return "not_applicable"
	}
	return status
}

func isDiscoverabilityDoctorCheck(id string) bool {
	switch {
	case id == "config_dir":
		return true
	case strings.HasPrefix(id, "agent_"):
		return true
	default:
		return false
	}
}

func isRuntimeDoctorCheck(id string) bool {
	switch {
	case id == "endpoint", id == "key", id == "auth_me", id == "skills_api", id == "model_routes":
		return true
	case strings.HasPrefix(id, "runtime_skill:"):
		return true
	default:
		return false
	}
}

func passCheck(id, message string, details map[string]any) doctorCheck {
	return doctorCheck{ID: id, Status: "pass", Message: message, Details: details}
}

func warnCheck(id, message string, details map[string]any) doctorCheck {
	return doctorCheck{ID: id, Status: "warn", Message: message, Details: details}
}

func failCheck(id, message string, details map[string]any) doctorCheck {
	return doctorCheck{ID: id, Status: "fail", Message: message, Details: details}
}
