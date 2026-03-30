package agentpaths

import (
	"path/filepath"
	"testing"
)

func TestResolveCodexPaths(t *testing.T) {
	t.Setenv("CODEX_HOME", "/tmp/codex-home")

	paths, err := Resolve("codex")
	if err != nil {
		t.Fatalf("Resolve(codex) returned error: %v", err)
	}
	if paths.SkillDir != filepath.Join("/tmp/codex-home", "skills") {
		t.Fatalf("unexpected codex skill dir: %q", paths.SkillDir)
	}
	if paths.MCPConfigPath != filepath.Join("/tmp/codex-home", "config.toml") {
		t.Fatalf("unexpected codex config path: %q", paths.MCPConfigPath)
	}
}

func TestResolveClaudeCodePaths(t *testing.T) {
	t.Setenv("CLAUDE_HOME", "/tmp/claude-home")
	t.Setenv("CLAUDE_CONFIG_PATH", "/tmp/claude.json")

	paths, err := Resolve("claude-code")
	if err != nil {
		t.Fatalf("Resolve(claude-code) returned error: %v", err)
	}
	if paths.SkillDir != filepath.Join("/tmp/claude-home", "skills") {
		t.Fatalf("unexpected claude skill dir: %q", paths.SkillDir)
	}
	if paths.MCPConfigPath != "/tmp/claude.json" {
		t.Fatalf("unexpected claude config path: %q", paths.MCPConfigPath)
	}
}

func TestResolveOpenCodePaths(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")

	paths, err := Resolve("opencode")
	if err != nil {
		t.Fatalf("Resolve(opencode) returned error: %v", err)
	}
	if paths.SkillDir != filepath.Join("/tmp/xdg", "opencode", "skill") {
		t.Fatalf("unexpected opencode skill dir: %q", paths.SkillDir)
	}
	if paths.MCPConfigPath != filepath.Join("/tmp/xdg", "opencode", "mcp.json") {
		t.Fatalf("unexpected opencode config path: %q", paths.MCPConfigPath)
	}
}

func TestResolveOpenClawPaths(t *testing.T) {
	t.Setenv("OPENCLAW_HOME", "/tmp/openclaw-home")

	paths, err := Resolve("openclaw")
	if err != nil {
		t.Fatalf("Resolve(openclaw) returned error: %v", err)
	}
	if paths.SkillDir != filepath.Join("/tmp/openclaw-home", "skills") {
		t.Fatalf("unexpected openclaw skill dir: %q", paths.SkillDir)
	}
	if paths.MCPConfigPath != filepath.Join("/tmp/openclaw-home", "mcp.json") {
		t.Fatalf("unexpected openclaw config path: %q", paths.MCPConfigPath)
	}
}
