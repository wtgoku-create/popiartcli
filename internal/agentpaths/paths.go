package agentpaths

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/wtgoku-create/popiartcli/internal/output"
)

type Paths struct {
	Agent         string
	HomeDir       string
	SkillDir      string
	MCPConfigPath string
}

func Resolve(agent string) (Paths, error) {
	agent = normalizeAgent(agent)
	if agent == "" {
		return Paths{}, output.NewError("VALIDATION_ERROR", "缺少 agent 名称", nil)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, output.NewError("CLI_ERROR", "解析用户目录失败", map[string]any{
			"details": err.Error(),
		})
	}

	switch agent {
	case "codex":
		base := firstNonEmpty(os.Getenv("CODEX_HOME"), filepath.Join(home, ".codex"))
		return Paths{
			Agent:         agent,
			HomeDir:       base,
			SkillDir:      filepath.Join(base, "skills"),
			MCPConfigPath: filepath.Join(base, "config.toml"),
		}, nil
	case "claude-code":
		base := firstNonEmpty(os.Getenv("CLAUDE_HOME"), filepath.Join(home, ".claude"))
		return Paths{
			Agent:         agent,
			HomeDir:       base,
			SkillDir:      firstNonEmpty(os.Getenv("CLAUDE_SKILLS_DIR"), filepath.Join(base, "skills")),
			MCPConfigPath: firstNonEmpty(os.Getenv("CLAUDE_CONFIG_PATH"), filepath.Join(home, ".claude.json")),
		}, nil
	case "openclaw":
		base := firstNonEmpty(os.Getenv("OPENCLAW_HOME"), os.Getenv("OPENCLAW_STATE_DIR"), filepath.Join(home, ".openclaw"))
		return Paths{
			Agent:         agent,
			HomeDir:       base,
			SkillDir:      firstNonEmpty(os.Getenv("OPENCLAW_SKILLS_DIR"), filepath.Join(base, "skills")),
			MCPConfigPath: firstNonEmpty(os.Getenv("OPENCLAW_MCP_CONFIG"), filepath.Join(base, "mcp.json")),
		}, nil
	case "opencode":
		base := strings.TrimSpace(os.Getenv("OPENCODE_HOME"))
		if base == "" {
			if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
				base = filepath.Join(xdg, "opencode")
			} else {
				base = filepath.Join(home, ".config", "opencode")
			}
		}
		return Paths{
			Agent:         agent,
			HomeDir:       base,
			SkillDir:      firstNonEmpty(os.Getenv("OPENCODE_SKILL_DIR"), filepath.Join(base, "skill")),
			MCPConfigPath: firstNonEmpty(os.Getenv("OPENCODE_MCP_CONFIG"), filepath.Join(base, "mcp.json")),
		}, nil
	default:
		return Paths{}, output.NewError("VALIDATION_ERROR", "不支持的 agent", map[string]any{
			"agent":     agent,
			"supported": []string{"claude-code", "codex", "openclaw", "opencode"},
		})
	}
}

func normalizeAgent(agent string) string {
	return strings.TrimSpace(strings.ToLower(agent))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return filepath.Clean(trimmed)
		}
	}
	return ""
}
