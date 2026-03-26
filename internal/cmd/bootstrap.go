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
	GeneratedFiles       []bootstrapGeneratedFile `json:"generated_files,omitempty"`
	NextSteps            []string                 `json:"next_steps,omitempty"`
}

func newBootstrapCmd() *cobra.Command {
	var agents []string
	var completions []string
	var key string
	var withDefaultSkills bool
	var noAgentConfig bool

	bootstrapCmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "初始化本地 PopiArt 环境与 agent 引导文件",
		RunE: func(cmd *cobra.Command, args []string) error {
			normalizedAgents, err := normalizeChoices(agents, supportedBootstrapAgents, "agent")
			if err != nil {
				return err
			}

			normalizedCompletions, err := normalizeChoices(completions, supportedCompletionShells, "completion shell")
			if err != nil {
				return err
			}

			if key != "" {
				if _, err := config.SavePatch(config.Patch{Token: &key}); err != nil {
					return output.NewError("CLI_ERROR", "保存 key 失败", map[string]any{"details": err.Error()})
				}
			}

			cfg := config.Load()
			result := bootstrapResult{
				CLIVersion:       cmd.Root().Version,
				ConfigPath:       config.Path(),
				Endpoint:         cfg.Endpoint,
				Project:          cfg.Project,
				KeySaved:         key != "",
				Agents:           normalizedAgents,
				CompletionShells: normalizedCompletions,
			}

			if withDefaultSkills {
				path, err := writeDefaultSkillset()
				if err != nil {
					return output.NewError("CLI_ERROR", "写入默认 skill profile 失败", map[string]any{"details": err.Error()})
				}
				result.DefaultSkillsProfile = "default"
				result.GeneratedFiles = append(result.GeneratedFiles, bootstrapGeneratedFile{
					Kind: "skillset",
					Name: "default",
					Path: path,
				})
			}

			if !noAgentConfig {
				for _, agent := range normalizedAgents {
					files, err := writeAgentEnvFiles(agent, cfg)
					if err != nil {
						return output.NewError("CLI_ERROR", "写入 agent 引导文件失败", map[string]any{
							"details": err.Error(),
							"agent":   agent,
						})
					}
					result.GeneratedFiles = append(result.GeneratedFiles, files...)
				}
			}

			for _, shell := range normalizedCompletions {
				path, err := writeCompletionFile(cmd.Root(), shell)
				if err != nil {
					return output.NewError("CLI_ERROR", "写入 shell completion 失败", map[string]any{
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
				return output.NewError("CLI_ERROR", "写入 bootstrap manifest 失败", map[string]any{"details": err.Error()})
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
	bootstrapCmd.Flags().BoolVar(&noAgentConfig, "no-agent-config", false, "跳过 agent 引导文件生成")

	return bootstrapCmd
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
		steps = append(steps, "运行 `popiart skills list --search popiskill-creator` 或 `popiart skills list --tag image`")
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
