package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wtgoku-create/popiartcli/internal/config"
)

func TestWriteAgentEnvFilesCreatesShellAndPowerShellFiles(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)

	files, err := writeAgentEnvFiles("codex", config.Config{
		Endpoint: "https://example.com/v1",
		Project:  "demo-project",
	})
	if err != nil {
		t.Fatalf("writeAgentEnvFiles returned error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 generated files, got %d", len(files))
	}

	shPath := filepath.Join(configDir, "agents", "codex", "env.sh")
	psPath := filepath.Join(configDir, "agents", "codex", "env.ps1")

	shData, err := os.ReadFile(shPath)
	if err != nil {
		t.Fatalf("read env.sh: %v", err)
	}
	if !strings.Contains(string(shData), `export POPIART_ENDPOINT='https://example.com/v1'`) {
		t.Fatalf("expected shell env to contain endpoint, got %q", string(shData))
	}
	if !strings.Contains(string(shData), `export POPIART_PROJECT='demo-project'`) {
		t.Fatalf("expected shell env to contain project, got %q", string(shData))
	}

	psData, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("read env.ps1: %v", err)
	}
	if !strings.Contains(string(psData), `$env:POPIART_ENDPOINT = 'https://example.com/v1'`) {
		t.Fatalf("expected powershell env to contain endpoint, got %q", string(psData))
	}
	if !strings.Contains(string(psData), `$env:POPIART_PROJECT = 'demo-project'`) {
		t.Fatalf("expected powershell env to contain project, got %q", string(psData))
	}
}

func TestWriteAgentMCPConfigFileCreatesSnippet(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)

	file, err := writeAgentMCPConfigFile("codex", config.Config{
		Endpoint: "https://example.com/v1",
		Project:  "demo-project",
	})
	if err != nil {
		t.Fatalf("writeAgentMCPConfigFile returned error: %v", err)
	}
	if file.Kind != "agent-mcp" {
		t.Fatalf("expected agent-mcp kind, got %q", file.Kind)
	}

	data, err := os.ReadFile(filepath.Join(configDir, "agents", "codex", "mcp.json"))
	if err != nil {
		t.Fatalf("read mcp.json: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, `"command": "popiart"`) {
		t.Fatalf("expected mcp snippet to contain popiart command, got %q", text)
	}
	if !strings.Contains(text, `"mcp"`) || !strings.Contains(text, `"serve"`) {
		t.Fatalf("expected mcp snippet to contain mcp serve args, got %q", text)
	}
	if !strings.Contains(text, `"POPIART_PROJECT": "demo-project"`) {
		t.Fatalf("expected mcp snippet to contain project, got %q", text)
	}
}

func TestWriteAgentSkillWrapperCreatesWrapper(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)

	file, err := writeAgentSkillWrapper("codex")
	if err != nil {
		t.Fatalf("writeAgentSkillWrapper returned error: %v", err)
	}
	if file.Kind != "agent-skill" {
		t.Fatalf("expected agent-skill kind, got %q", file.Kind)
	}

	data, err := os.ReadFile(filepath.Join(configDir, "agents", "codex", "SKILL.md"))
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "# PopiArt") {
		t.Fatalf("expected wrapper heading, got %q", text)
	}
	if !strings.Contains(text, "popiskill-image-img2img-basic-v1") {
		t.Fatalf("expected runtime baseline skill in wrapper, got %q", text)
	}
}
