package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDirPrefersPopiartConfigDir(t *testing.T) {
	customDir := filepath.Join(t.TempDir(), "popiart-config")
	xdgDir := filepath.Join(t.TempDir(), "xdg-home")

	t.Setenv("POPIART_CONFIG_DIR", customDir)
	t.Setenv("XDG_CONFIG_HOME", xdgDir)

	if got := Dir(); got != customDir {
		t.Fatalf("Dir() = %q, want %q", got, customDir)
	}
}

func TestLoadUsesStoredConfigAndEnvOverrides(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)

	stored := Config{
		Endpoint: "https://stored.example/v1",
		Token:    "stored-token",
		Project:  "proj_stored",
	}
	writeStoredConfig(t, configDir, stored)

	got := Load()
	if got != stored {
		t.Fatalf("Load() without env overrides = %#v, want %#v", got, stored)
	}

	t.Setenv("POPIART_ENDPOINT", "https://env.example/v1")
	t.Setenv("POPIART_KEY", "env-key")
	t.Setenv("POPIART_PROJECT", "proj_env")

	got = Load()
	want := Config{
		Endpoint: "https://env.example/v1",
		Token:    "env-key",
		Project:  "proj_env",
	}
	if got != want {
		t.Fatalf("Load() with env overrides = %#v, want %#v", got, want)
	}

	t.Setenv("POPIART_KEY", "")
	t.Setenv("POPIART_TOKEN", "legacy-token")

	got = Load()
	if got.Token != "legacy-token" {
		t.Fatalf("Load() token override = %q, want %q", got.Token, "legacy-token")
	}
}

func TestSavePatchPersistsMergedConfig(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)

	token := "token-1"
	saved, err := SavePatch(Patch{Token: &token})
	if err != nil {
		t.Fatalf("SavePatch(token) error = %v", err)
	}
	if saved.Endpoint != defaultEndpoint {
		t.Fatalf("SavePatch(token) endpoint = %q, want %q", saved.Endpoint, defaultEndpoint)
	}
	if saved.Token != token {
		t.Fatalf("SavePatch(token) token = %q, want %q", saved.Token, token)
	}

	project := "proj_123"
	saved, err = SavePatch(Patch{Project: &project})
	if err != nil {
		t.Fatalf("SavePatch(project) error = %v", err)
	}
	if saved.Token != token {
		t.Fatalf("SavePatch(project) token = %q, want %q", saved.Token, token)
	}
	if saved.Project != project {
		t.Fatalf("SavePatch(project) project = %q, want %q", saved.Project, project)
	}

	data, err := os.ReadFile(Path())
	if err != nil {
		t.Fatalf("ReadFile(Path()) error = %v", err)
	}

	var onDisk Config
	if err := json.Unmarshal(data, &onDisk); err != nil {
		t.Fatalf("json.Unmarshal(config) error = %v", err)
	}
	if onDisk.Token != token || onDisk.Project != project || onDisk.Endpoint != defaultEndpoint {
		t.Fatalf("config on disk = %#v, want token=%q project=%q endpoint=%q", onDisk, token, project, defaultEndpoint)
	}
}

func TestRequireTokenUsesCurrentLoadSource(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)

	if _, err := RequireToken(); err == nil {
		t.Fatal("RequireToken() error = nil, want missing token")
	}

	token := "from-env"
	t.Setenv("POPIART_KEY", token)

	got, err := RequireToken()
	if err != nil {
		t.Fatalf("RequireToken() error = %v", err)
	}
	if got != token {
		t.Fatalf("RequireToken() = %q, want %q", got, token)
	}
}

func writeStoredConfig(t *testing.T, configDir string, cfg Config) {
	t.Helper()

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", configDir, err)
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal(cfg) error = %v", err)
	}

	if err := os.WriteFile(filepath.Join(configDir, "config.json"), data, 0o600); err != nil {
		t.Fatalf("WriteFile(config.json) error = %v", err)
	}
}
