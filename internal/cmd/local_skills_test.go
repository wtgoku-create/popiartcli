package cmd

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestSkillsInstallListGetAndSchemaForLocalSkill(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("POPIART_ENDPOINT", "http://127.0.0.1:1")

	archivePath := writeTestSkillArchive(t, testSkillArchiveOptions{
		Slug:           "popiskill-local-audio-avatar-v1",
		DisplayName:    "Local Audio Avatar",
		Description:    "Installable local audio avatar skill.",
		RuntimeSkillID: "popiskill-remote-audio-avatar-v1",
	})

	installResp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"skills", "install", archivePath,
		"--agent", "codex",
		"--agent-skill-dir", filepath.Join(configDir, "codex-skills"),
	})
	if data, ok := installResp["data"].(map[string]any); !ok {
		t.Fatalf("expected install data object, got %#v", installResp["data"])
	} else {
		if data["slug"] != "popiskill-local-audio-avatar-v1" {
			t.Fatalf("unexpected slug: %#v", data["slug"])
		}
		agentPath, _ := data["agent_skill_path"].(string)
		if agentPath == "" {
			t.Fatal("expected agent skill path to be returned")
		}
		if _, err := os.Stat(filepath.Join(agentPath, "SKILL.md")); err != nil {
			t.Fatalf("expected linked agent skill to contain SKILL.md: %v", err)
		}
	}

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{"skills", "list"})
	data := resp["data"].(map[string]any)
	if _, ok := data["items"].([]any); !ok {
		t.Fatalf("expected skills list items, got %#v", data)
	}
	getResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"skills", "get", "popiskill-local-audio-avatar-v1"})
	getData := getResp["data"].(map[string]any)
	if getData["id"] != "popiskill-local-audio-avatar-v1" {
		t.Fatalf("unexpected skill id: %#v", getData["id"])
	}
	if getData["source"] != "installed" {
		t.Fatalf("expected installed source, got %#v", getData["source"])
	}
	schemaResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"skills", "schema", "popiskill-local-audio-avatar-v1"})
	schemaData := schemaResp["data"].(map[string]any)
	if _, ok := schemaData["input_schema"].(map[string]any); !ok {
		t.Fatalf("expected input_schema, got %#v", schemaData)
	}
}

func TestUseLocalInstalledSkillCanRunViaOfficialBridge(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/ai/model/list":
			_, _ = w.Write([]byte(`{"ok":true,"data":[{"id":101,"code":"Nano-banana-pro","resolution":["2K"],"ratio":["16:9"],"categories":[{"taskSubType":103}]}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api_client/anime/task/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode task body: %v", err)
			}
			if body["chatPrompt"] != "hello" {
				t.Fatalf("unexpected prompt: %#v", body["chatPrompt"])
			}
			if body["subType"] != float64(103) {
				t.Fatalf("unexpected subtype: %#v", body["subType"])
			}
			_, _ = w.Write([]byte(`{"ok":true,"data":{"id":"task_local_bridge_1","status":0,"type":1,"subType":103}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"NOT_FOUND","message":"not found"}}`))
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	archivePath := writeTestSkillArchive(t, testSkillArchiveOptions{
		Slug:           "popiskill-shared-runtime-v1",
		DisplayName:    "Shared Runtime Local Wrapper",
		Description:    "Installed local wrapper bridged to official text2image runtime.",
		RuntimeSkillID: officialText2ImageSkillID,
	})

	executeRootJSON(t, NewRootCmd("0.test"), []string{"skills", "install", archivePath})
	executeRootJSON(t, NewRootCmd("0.test"), []string{"skills", "use-local", "popiskill-shared-runtime-v1"})

	getResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"skills", "get", "popiskill-shared-runtime-v1"})
	getData := getResp["data"].(map[string]any)
	if getData["local_active"] != true {
		t.Fatalf("expected local_active=true, got %#v", getData["local_active"])
	}

	runResp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"run", "popiskill-shared-runtime-v1",
		"--input", `{"prompt":"hello"}`,
	})
	runData := runResp["data"].(map[string]any)
	if runData["job_id"] != "task_local_bridge_1" {
		t.Fatalf("unexpected job_id: %#v", runData["job_id"])
	}
}

func TestSkillsInstallUsesNativeAgentSkillDirByDefault(t *testing.T) {
	configDir := t.TempDir()
	xdgDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv("POPIART_ENDPOINT", "http://127.0.0.1:1")

	archivePath := writeTestSkillArchive(t, testSkillArchiveOptions{
		Slug:           "popiskill-local-opencode-v1",
		DisplayName:    "Local OpenCode Skill",
		Description:    "Installable local skill for OpenCode.",
		RuntimeSkillID: "popiskill-remote-opencode-v1",
	})

	installResp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"skills", "install", archivePath,
		"--agent", "opencode",
	})
	data, ok := installResp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected install data object, got %#v", installResp["data"])
	}
	agentPath, _ := data["agent_skill_path"].(string)
	expectedPath := filepath.Join(xdgDir, "opencode", "skill", "popiskill-local-opencode-v1")
	if agentPath != expectedPath {
		t.Fatalf("expected native opencode skill path %q, got %q", expectedPath, agentPath)
	}
	if _, err := os.Stat(filepath.Join(expectedPath, "SKILL.md")); err != nil {
		t.Fatalf("expected linked native opencode skill to contain SKILL.md: %v", err)
	}
}

type testSkillArchiveOptions struct {
	Slug           string
	DisplayName    string
	Description    string
	RuntimeSkillID string
}

func writeTestSkillArchive(t *testing.T, opts testSkillArchiveOptions) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), opts.Slug+".zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer file.Close()

	archive := zip.NewWriter(file)
	root := opts.Slug + "/"
	addFileToZip(t, archive, root+"SKILL.md", strings.TrimSpace(`
---
slug: `+opts.Slug+`
display_name: `+opts.DisplayName+`
description: `+opts.Description+`
version: 1.0.0
archive: `+opts.Slug+`.zip
package_layout: rooted
package_root: `+opts.Slug+`
capabilities:
  - audio
  - video
requires_popiart_auth: true
input_schema_path: input_schema.json
output_schema_path: output_schema.json
execution:
  mode: remote-runtime
  runtime_skill_id: `+opts.RuntimeSkillID+`
  runner: popiart
---

# `+opts.DisplayName+`

`+opts.Description+`
`))
	addFileToZip(t, archive, root+"input_schema.json", `{"type":"object","properties":{"prompt":{"type":"string"}},"required":["prompt"]}`)
	addFileToZip(t, archive, root+"output_schema.json", `{"type":"object","properties":{"job_id":{"type":"string"}}}`)

	if err := archive.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
	return path
}

func addFileToZip(t *testing.T, archive *zip.Writer, name, contents string) {
	t.Helper()
	writer, err := archive.Create(name)
	if err != nil {
		t.Fatalf("create zip entry %s: %v", name, err)
	}
	if _, err := writer.Write([]byte(contents)); err != nil {
		t.Fatalf("write zip entry %s: %v", name, err)
	}
}

func executeRootJSON(t *testing.T, root *cobra.Command, args []string) map[string]any {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)
	root.SetContext(context.Background())

	if err := root.Execute(); err != nil {
		t.Fatalf("execute %v failed: %v stderr=%s", args, err, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal stdout for %v failed: %v output=%q", args, err, stdout.String())
	}
	if ok, _ := payload["ok"].(bool); !ok {
		t.Fatalf("expected ok response for %v, got %#v", args, payload)
	}
	return payload
}
