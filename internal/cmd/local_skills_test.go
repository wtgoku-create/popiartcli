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

	listResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"skills", "list"})
	listData, ok := listResp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected list data object, got %#v", listResp["data"])
	}
	items, ok := listData["items"].([]any)
	if !ok {
		t.Fatalf("expected list items array, got %#v", listData["items"])
	}
	var found bool
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if entry["id"] == "popiskill-local-audio-avatar-v1" && entry["source"] == "installed" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected installed local skill in list, got %#v", items)
	}

	getResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"skills", "get", "popiskill-local-audio-avatar-v1"})
	getData, ok := getResp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected get data object, got %#v", getResp["data"])
	}
	if getData["source"] != "installed" {
		t.Fatalf("expected installed source, got %#v", getData["source"])
	}
	if getData["runtime_skill_id"] != "popiskill-remote-audio-avatar-v1" {
		t.Fatalf("expected runtime skill id, got %#v", getData["runtime_skill_id"])
	}

	schemaResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"skills", "schema", "popiskill-local-audio-avatar-v1"})
	schemaData, ok := schemaResp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected schema data object, got %#v", schemaResp["data"])
	}
	inputSchema, ok := schemaData["input_schema"].(map[string]any)
	if !ok {
		t.Fatalf("expected input_schema object, got %#v", schemaData["input_schema"])
	}
	required, ok := inputSchema["required"].([]any)
	if !ok || len(required) != 1 || required[0] != "prompt" {
		t.Fatalf("unexpected input_schema.required: %#v", inputSchema["required"])
	}
}

func TestUseLocalOverridesRemoteSkillAtRunTime(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/skills/popiskill-shared-runtime-v1":
			_, _ = w.Write([]byte(`{"ok":true,"data":{"id":"popiskill-shared-runtime-v1","name":"Remote Shared Runtime","description":"Remote","tags":["remote"],"version":"1.0.0","input_schema":{"type":"object"},"output_schema":{"type":"object"},"model_type":"video","estimated_duration_s":30}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/jobs":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode job body: %v", err)
			}
			if body["skill_id"] != "popiskill-remote-audio-avatar-v1" {
				t.Fatalf("expected runtime skill id override, got %#v", body["skill_id"])
			}
			_, _ = w.Write([]byte(`{"ok":true,"data":{"job_id":"job_local_override","status":"pending"}}`))
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
		Description:    "Installed local wrapper that should override remote when activated.",
		RuntimeSkillID: "popiskill-remote-audio-avatar-v1",
	})

	executeRootJSON(t, NewRootCmd("0.test"), []string{"skills", "install", archivePath})
	executeRootJSON(t, NewRootCmd("0.test"), []string{"skills", "use-local", "popiskill-shared-runtime-v1"})

	getResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"skills", "get", "popiskill-shared-runtime-v1"})
	getData, ok := getResp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected get data object, got %#v", getResp["data"])
	}
	if getData["source"] != "installed" || getData["local_active"] != true {
		t.Fatalf("expected active installed skill, got %#v", getData)
	}

	runResp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"run", "popiskill-shared-runtime-v1",
		"--input", `{"prompt":"hello"}`,
	})
	runData, ok := runResp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected run data object, got %#v", runResp["data"])
	}
	if runData["job_id"] != "job_local_override" {
		t.Fatalf("expected overridden job id, got %#v", runData["job_id"])
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
