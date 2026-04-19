package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/seed"
)

var supportedBootstrapAgents = map[string]string{
	"claude-code": "Anthropic Claude Code",
	"codex":       "OpenAI Codex",
	"openclaw":    "OpenClaw",
	"opencode":    "OpenCode",
}

var supportedCompletionShells = map[string]string{
	"bash":       "Bash",
	"fish":       "Fish",
	"powershell": "PowerShell",
	"zsh":        "Zsh",
}

type bootstrapGeneratedFile struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	Path string `json:"path"`
}

type bootstrapResult struct {
	CLIVersion           string                   `json:"cli_version"`
	ConfigPath           string                   `json:"config_path"`
	ManifestPath         string                   `json:"manifest_path"`
	Endpoint             string                   `json:"endpoint"`
	Project              string                   `json:"project,omitempty"`
	KeySaved             bool                     `json:"key_saved"`
	Agents               []string                 `json:"agents,omitempty"`
	CompletionShells     []string                 `json:"completion_shells,omitempty"`
	DefaultSkillsProfile string                   `json:"default_skills_profile,omitempty"`
	RuntimeBaseline      string                   `json:"runtime_baseline,omitempty"`
	GeneratedFiles       []bootstrapGeneratedFile `json:"generated_files,omitempty"`
	NextSteps            []string                 `json:"next_steps,omitempty"`
}

type bootstrapOptions struct {
	Key                 string
	Agents              []string
	Completions         []string
	WithDefaultSkills   bool
	WithRuntimeBaseline bool
	InstallMCP          bool
	InstallSkill        bool
	Discoverable        bool
	NoAgentConfig       bool
}

func newBootstrapCmd() *cobra.Command {
	var agents []string
	var completions []string
	var key string
	var withDefaultSkills bool
	var withRuntimeBaseline bool
	var installMCP bool
	var installSkill bool
	var discoverable bool
	var noAgentConfig bool

	bootstrapCmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "初始化本地 PopiArt 环境与 agent 引导文件",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := executeBootstrap(cmd, bootstrapOptions{
				Key:                 key,
				Agents:              agents,
				Completions:         completions,
				WithDefaultSkills:   withDefaultSkills,
				WithRuntimeBaseline: withRuntimeBaseline,
				InstallMCP:          installMCP,
				InstallSkill:        installSkill,
				Discoverable:        discoverable,
				NoAgentConfig:       noAgentConfig,
			})
			if err != nil {
				return err
			}
			if plainOutput(cmd) {
				writeBootstrapPlain(cmd.OutOrStdout(), result)
				return nil
			}
			return writeOutput(cmd, result)
		},
	}

	bootstrapCmd.Flags().StringVarP(&key, "key", "k", "", "直接保存 API key 到本地配置")
	bootstrapCmd.Flags().StringArrayVar(&agents, "agent", nil, "生成指定 agent 的引导文件，可重复传递")
	bootstrapCmd.Flags().StringArrayVar(&completions, "completion", nil, "生成指定 shell 的 completion，可重复传递")
	bootstrapCmd.Flags().BoolVar(&withDefaultSkills, "with-default-skills", false, "生成默认的远程 skill discovery profile")
	bootstrapCmd.Flags().BoolVar(&withRuntimeBaseline, "with-runtime-baseline", false, "生成官方 runtime baseline 清单")
	bootstrapCmd.Flags().BoolVar(&installMCP, "install-mcp", false, "为指定 agent 生成 PopiArt MCP 配置片段")
	bootstrapCmd.Flags().BoolVar(&installSkill, "install-skill", false, "为指定 agent 生成 PopiArt skill wrapper")
	bootstrapCmd.Flags().BoolVar(&discoverable, "discoverable", false, "一次性生成 discoverability 所需的 MCP、skill 和 runtime baseline 资产")
	bootstrapCmd.Flags().BoolVar(&noAgentConfig, "no-agent-config", false, "跳过 agent 引导文件生成")

	return bootstrapCmd
}

func newSetupCmd() *cobra.Command {
	var agents []string
	var completions []string
	var key string
	var noAgentConfig bool

	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "面向新用户和 agent 的一键初始化入口",
		Long: "面向首次接入的默认入口。\n\n" +
			"`popiart setup --agent codex` 会在不改变底层架构的前提下，直接完成默认 skill profile、runtime baseline、agent env、原生 MCP 配置、原生 skill wrapper 等 discoverability 资产。\n\n" +
			"如果你只想做细粒度引导，仍然可以继续使用 `popiart bootstrap`。",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := executeBootstrap(cmd, bootstrapOptions{
				Key:                 key,
				Agents:              agents,
				Completions:         completions,
				WithDefaultSkills:   true,
				WithRuntimeBaseline: true,
				InstallMCP:          true,
				InstallSkill:        true,
				Discoverable:        true,
				NoAgentConfig:       noAgentConfig,
			})
			if err != nil {
				return err
			}
			if plainOutput(cmd) {
				writeBootstrapPlain(cmd.OutOrStdout(), result)
				return nil
			}
			return writeOutput(cmd, result)
		},
	}

	setupCmd.Flags().StringVarP(&key, "key", "k", "", "直接保存 API key 到本地配置")
	setupCmd.Flags().StringArrayVar(&agents, "agent", nil, "目标 agent，例如 codex、claude-code、openclaw、opencode")
	setupCmd.Flags().StringArrayVar(&completions, "completion", nil, "可选生成 shell completion，可重复传递")
	setupCmd.Flags().BoolVar(&noAgentConfig, "no-agent-config", false, "跳过 ~/.popiart/agents/<agent>/ 下的 env 引导文件生成")
	return setupCmd
}

func executeBootstrap(cmd *cobra.Command, opts bootstrapOptions) (bootstrapResult, error) {
	if opts.Discoverable {
		opts.WithDefaultSkills = true
		opts.WithRuntimeBaseline = true
		opts.InstallMCP = true
		opts.InstallSkill = true
	}

	normalizedAgents, err := normalizeChoices(opts.Agents, supportedBootstrapAgents, "agent")
	if err != nil {
		return bootstrapResult{}, err
	}

	normalizedCompletions, err := normalizeChoices(opts.Completions, supportedCompletionShells, "completion shell")
	if err != nil {
		return bootstrapResult{}, err
	}

	if opts.Key != "" {
		if _, err := config.SavePatch(config.Patch{Token: &opts.Key}); err != nil {
			return bootstrapResult{}, output.NewError("CLI_ERROR", "保存 key 失败", map[string]any{"details": err.Error()})
		}
	}
	if (opts.InstallMCP || opts.InstallSkill) && len(normalizedAgents) == 0 {
		return bootstrapResult{}, output.NewError("VALIDATION_ERROR", "discoverability 产物需要显式指定至少一个 agent", map[string]any{
			"hint": "传入 `--agent codex`、`--agent claude-code`、`--agent openclaw` 或 `--agent opencode`",
		})
	}

	cfg := config.Load()
	result := bootstrapResult{
		CLIVersion:       cmd.Root().Version,
		ConfigPath:       config.Path(),
		Endpoint:         cfg.Endpoint,
		Project:          cfg.Project,
		KeySaved:         opts.Key != "",
		Agents:           normalizedAgents,
		CompletionShells: normalizedCompletions,
	}

	if opts.WithDefaultSkills {
		path, err := writeDefaultSkillset()
		if err != nil {
			return bootstrapResult{}, output.NewError("CLI_ERROR", "写入默认 skill profile 失败", map[string]any{"details": err.Error()})
		}
		result.DefaultSkillsProfile = "default"
		result.GeneratedFiles = append(result.GeneratedFiles, bootstrapGeneratedFile{
			Kind: "skillset",
			Name: "default",
			Path: path,
		})
	}
	if opts.WithRuntimeBaseline {
		path, err := writeRuntimeBaselineProfile()
		if err != nil {
			return bootstrapResult{}, output.NewError("CLI_ERROR", "写入 runtime baseline 失败", map[string]any{"details": err.Error()})
		}
		result.RuntimeBaseline = "runtime-baseline"
		result.GeneratedFiles = append(result.GeneratedFiles, bootstrapGeneratedFile{
			Kind: "runtime-baseline",
			Name: "runtime-baseline",
			Path: path,
		})
	}

	if !opts.NoAgentConfig {
		for _, agent := range normalizedAgents {
			files, err := writeAgentEnvFiles(agent, cfg)
			if err != nil {
				return bootstrapResult{}, output.NewError("CLI_ERROR", "写入 agent 引导文件失败", map[string]any{
					"details": err.Error(),
					"agent":   agent,
				})
			}
			result.GeneratedFiles = append(result.GeneratedFiles, files...)
		}
	}
	if opts.InstallMCP {
		for _, agent := range normalizedAgents {
			file, err := writeAgentMCPConfigFile(agent, cfg)
			if err != nil {
				return bootstrapResult{}, output.NewError("CLI_ERROR", "写入 agent MCP 配置片段失败", map[string]any{
					"details": err.Error(),
					"agent":   agent,
				})
			}
			result.GeneratedFiles = append(result.GeneratedFiles, file)

			nativeFile, err := writeNativeAgentMCPConfigFile(agent, cfg)
			if err != nil {
				return bootstrapResult{}, output.NewError("CLI_ERROR", "写入 agent 原生 MCP 配置失败", map[string]any{
					"details": err.Error(),
					"agent":   agent,
				})
			}
			result.GeneratedFiles = append(result.GeneratedFiles, nativeFile)
		}
	}
	if opts.InstallSkill {
		for _, agent := range normalizedAgents {
			file, err := writeAgentSkillWrapper(agent)
			if err != nil {
				return bootstrapResult{}, output.NewError("CLI_ERROR", "写入 agent skill wrapper 失败", map[string]any{
					"details": err.Error(),
					"agent":   agent,
				})
			}
			result.GeneratedFiles = append(result.GeneratedFiles, file)

			nativeFile, err := writeNativeAgentSkillWrapper(agent)
			if err != nil {
				return bootstrapResult{}, output.NewError("CLI_ERROR", "写入 agent 原生 skill wrapper 失败", map[string]any{
					"details": err.Error(),
					"agent":   agent,
				})
			}
			result.GeneratedFiles = append(result.GeneratedFiles, nativeFile)
		}
	}

	for _, shell := range normalizedCompletions {
		path, err := writeCompletionFile(cmd.Root(), shell)
		if err != nil {
			return bootstrapResult{}, output.NewError("CLI_ERROR", "写入 shell completion 失败", map[string]any{
				"details": err.Error(),
				"shell":   shell,
			})
		}
		result.GeneratedFiles = append(result.GeneratedFiles, bootstrapGeneratedFile{
			Kind: "completion",
			Name: shell,
			Path: path,
		})
	}

	result.NextSteps = bootstrapNextSteps(result)
	manifestPath := filepath.Join(config.Dir(), "bootstrap.json")
	result.ManifestPath = manifestPath
	if err := writeJSONFile(manifestPath, result); err != nil {
		return bootstrapResult{}, output.NewError("CLI_ERROR", "写入 bootstrap manifest 失败", map[string]any{"details": err.Error()})
	}
	return result, nil
}

func normalizeChoices(values []string, allowed map[string]string, label string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	seen := map[string]bool{}
	var normalized []string
	for _, value := range values {
		key := strings.TrimSpace(strings.ToLower(value))
		if key == "" {
			continue
		}
		if _, ok := allowed[key]; !ok {
			supported := make([]string, 0, len(allowed))
			for name := range allowed {
				supported = append(supported, name)
			}
			slices.Sort(supported)
			return nil, output.NewError("VALIDATION_ERROR", "不支持的 bootstrap 参数", map[string]any{
				"field":     label,
				"value":     value,
				"supported": supported,
			})
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		normalized = append(normalized, key)
	}
	slices.Sort(normalized)
	return normalized, nil
}

func writeDefaultSkillset() (string, error) {
	path := filepath.Join(config.Dir(), "skillsets", "default.json")
	skillset := seed.NewDefaultSkillset(time.Now().UTC())
	return path, writeJSONFile(path, skillset)
}

func writeRuntimeBaselineProfile() (string, error) {
	path := filepath.Join(config.Dir(), "skillsets", "runtime-baseline.json")
	profile := map[string]any{
		"name":         "runtime-baseline",
		"description":  "Official PopiArt runtime baseline for the current seven runtime skills across image, video, and audio.",
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"source":       "popiart bootstrap",
		"mcp_server":   popiartMCPServerName,
		"skills":       officialRuntimeSkills(),
	}
	return path, writeJSONFile(path, profile)
}

func writeAgentEnvFiles(agent string, cfg config.Config) ([]bootstrapGeneratedFile, error) {
	dir := filepath.Join(config.Dir(), "agents", agent)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}

	shPath := filepath.Join(dir, "env.sh")
	shLines := []string{
		"# Generated by `popiart bootstrap`.",
		fmt.Sprintf("export POPIART_CONFIG_DIR=%s", shellQuote(config.Dir())),
		fmt.Sprintf("export POPIART_ENDPOINT=%s", shellQuote(cfg.Endpoint)),
	}
	if cfg.Project != "" {
		shLines = append(shLines, fmt.Sprintf("export POPIART_PROJECT=%s", shellQuote(cfg.Project)))
	}
	shLines = append(shLines, "")
	if err := os.WriteFile(shPath, []byte(strings.Join(shLines, "\n")), 0o600); err != nil {
		return nil, err
	}

	psPath := filepath.Join(dir, "env.ps1")
	psLines := []string{
		"# Generated by `popiart bootstrap`.",
		fmt.Sprintf("$env:POPIART_CONFIG_DIR = %s", powerShellQuote(config.Dir())),
		fmt.Sprintf("$env:POPIART_ENDPOINT = %s", powerShellQuote(cfg.Endpoint)),
	}
	if cfg.Project != "" {
		psLines = append(psLines, fmt.Sprintf("$env:POPIART_PROJECT = %s", powerShellQuote(cfg.Project)))
	}
	psLines = append(psLines, "")
	if err := os.WriteFile(psPath, []byte(strings.Join(psLines, "\n")), 0o600); err != nil {
		return nil, err
	}

	return []bootstrapGeneratedFile{
		{Kind: "agent-env", Name: agent + " (sh)", Path: shPath},
		{Kind: "agent-env", Name: agent + " (powershell)", Path: psPath},
	}, nil
}

func writeAgentMCPConfigFile(agent string, cfg config.Config) (bootstrapGeneratedFile, error) {
	dir := filepath.Join(config.Dir(), "agents", agent)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return bootstrapGeneratedFile{}, err
	}

	path := filepath.Join(dir, "mcp.json")
	if err := writeJSONFile(path, buildAgentMCPConfig(agent, cfg)); err != nil {
		return bootstrapGeneratedFile{}, err
	}

	return bootstrapGeneratedFile{
		Kind: "agent-mcp",
		Name: agent + " MCP",
		Path: path,
	}, nil
}

func writeAgentSkillWrapper(agent string) (bootstrapGeneratedFile, error) {
	dir := filepath.Join(config.Dir(), "agents", agent)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return bootstrapGeneratedFile{}, err
	}

	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, []byte(strings.Join(buildAgentSkillWrapperLines(), "\n")), 0o600); err != nil {
		return bootstrapGeneratedFile{}, err
	}
	return bootstrapGeneratedFile{
		Kind: "agent-skill",
		Name: agent + " skill wrapper",
		Path: path,
	}, nil
}

func writeCompletionFile(root *cobra.Command, shell string) (string, error) {
	path := completionFilePath(shell)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	defer file.Close()

	switch shell {
	case "bash":
		return path, root.GenBashCompletionV2(file, true)
	case "zsh":
		return path, root.GenZshCompletion(file)
	case "fish":
		return path, root.GenFishCompletion(file, true)
	case "powershell":
		return path, root.GenPowerShellCompletionWithDesc(file)
	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}
}

func completionFilePath(shell string) string {
	base := filepath.Join(config.Dir(), "completions")
	switch shell {
	case "bash":
		return filepath.Join(base, "popiart.bash")
	case "zsh":
		return filepath.Join(base, "_popiart")
	case "fish":
		return filepath.Join(base, "popiart.fish")
	case "powershell":
		return filepath.Join(base, "popiart.ps1")
	default:
		return filepath.Join(base, "popiart."+shell)
	}
}

func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func bootstrapNextSteps(result bootstrapResult) []string {
	var steps []string
	if !result.KeySaved {
		steps = append(steps, "运行 `popiart auth login` 保存 API key")
	}
	if result.DefaultSkillsProfile != "" {
		steps = append(steps, "运行 `popiart skills list --search popiskill-image-text2image-basic-v1` 或 `popiart skills list --tag image`")
	}
	if result.RuntimeBaseline != "" {
		doctorCmd := "popiart mcp doctor"
		if len(result.Agents) == 1 {
			doctorCmd += " --agent " + result.Agents[0]
		}
		steps = append(steps, fmt.Sprintf("运行 `%s`，分别检查 `discoverability_status` 和 `runtime_status`", doctorCmd))
		steps = append(steps, "注意：discoverability 通过只代表 agent 已能发现 PopiArt；runtime_status 通过才代表远端 runtime baseline 更接近可执行")
	}
	for _, file := range result.GeneratedFiles {
		if file.Kind == "completion" {
			switch file.Name {
			case "zsh":
				steps = append(steps, fmt.Sprintf("执行 `source %s` 立即启用 zsh completion", shellQuote(file.Path)))
			case "bash":
				steps = append(steps, fmt.Sprintf("执行 `source %s` 立即启用 bash completion", shellQuote(file.Path)))
			case "fish":
				steps = append(steps, fmt.Sprintf("执行 `source %s` 立即启用 fish completion", shellQuote(file.Path)))
			case "powershell":
				steps = append(steps, fmt.Sprintf("在 PowerShell 中执行 `. %s` 立即启用 completion", powerShellQuote(file.Path)))
			}
		}
		if file.Kind == "agent-env" {
			switch filepath.Ext(file.Path) {
			case ".ps1":
				steps = append(steps, fmt.Sprintf("如需在 PowerShell 中给对应 agent 注入环境，可执行 `. %s`", powerShellQuote(file.Path)))
			case ".sh":
				steps = append(steps, fmt.Sprintf("如需在 shell 中给对应 agent 注入环境，可执行 `source %s`", shellQuote(file.Path)))
			default:
				steps = append(steps, fmt.Sprintf("如需给对应 agent 注入环境，可引用 `%s`", file.Path))
			}
		}
		if file.Kind == "agent-mcp" {
			steps = append(steps, fmt.Sprintf("将 `%s` 合并到对应 agent 的 MCP server 配置，使 `%s` 可被发现", file.Path, popiartMCPServerName))
		}
		if file.Kind == "agent-mcp-native" {
			steps = append(steps, fmt.Sprintf("`%s` 已写入 `%s`，对应 agent 启动后应可直接发现 `%s` MCP", file.Name, file.Path, popiartMCPServerName))
		}
		if file.Kind == "agent-skill" {
			steps = append(steps, fmt.Sprintf("将 `%s` 复制或链接到对应 agent 的 skill 目录", file.Path))
		}
		if file.Kind == "agent-skill-native" {
			steps = append(steps, fmt.Sprintf("`%s` 已写入 `%s`，对应 agent 应可直接发现 PopiArt skill wrapper", file.Name, file.Path))
		}
	}
	if len(steps) == 0 {
		steps = append(steps, "运行 `popiart skills list` 开始浏览远程 skill 注册表")
	}
	return steps
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func powerShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func writeBootstrapPlain(w io.Writer, result bootstrapResult) {
	fmt.Fprintln(w, "bootstrap complete")
	fmt.Fprintf(w, "config: %s\n", result.ConfigPath)
	fmt.Fprintf(w, "manifest: %s\n", result.ManifestPath)
	fmt.Fprintf(w, "endpoint: %s\n", result.Endpoint)
	if result.Project != "" {
		fmt.Fprintf(w, "project: %s\n", result.Project)
	}
	if result.KeySaved {
		fmt.Fprintln(w, "key: saved")
	}
	if len(result.Agents) > 0 {
		fmt.Fprintf(w, "agents: %s\n", strings.Join(result.Agents, ", "))
	}
	if len(result.CompletionShells) > 0 {
		fmt.Fprintf(w, "completions: %s\n", strings.Join(result.CompletionShells, ", "))
	}
	if result.DefaultSkillsProfile != "" {
		fmt.Fprintf(w, "skill profile: %s\n", result.DefaultSkillsProfile)
	}
	if result.RuntimeBaseline != "" {
		fmt.Fprintf(w, "runtime baseline: %s\n", result.RuntimeBaseline)
		fmt.Fprintln(w, "runtime note: discoverability does not guarantee remote runtime readiness")
	}
	if len(result.GeneratedFiles) > 0 {
		fmt.Fprintln(w, "generated files:")
		for _, file := range result.GeneratedFiles {
			fmt.Fprintf(w, "  - [%s] %s -> %s\n", file.Kind, file.Name, file.Path)
		}
	}
	if len(result.NextSteps) > 0 {
		fmt.Fprintln(w, "next steps:")
		for _, step := range result.NextSteps {
			fmt.Fprintf(w, "  - %s\n", step)
		}
	}
}
