package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wtgoku-create/popiartcli/internal/output"
)

func TestArtifactsUploadCommand(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("POPIART_KEY", "pk-demo")
	t.Setenv("POPIART_PROJECT", "proj_agent_chat")

	sourcePath := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(sourcePath, []byte("png-body"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/artifacts/upload" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer pk-demo" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "multipart/form-data; boundary=") {
			t.Fatalf("expected multipart content type, got %q", got)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart form: %v", err)
		}
		if r.FormValue("project_id") != "proj_agent_chat" {
			t.Fatalf("unexpected project_id: %q", r.FormValue("project_id"))
		}
		if r.FormValue("role") != "source" {
			t.Fatalf("unexpected role: %q", r.FormValue("role"))
		}
		if r.FormValue("metadata_json") != `{"origin":"agent-chat"}` {
			t.Fatalf("unexpected metadata_json: %q", r.FormValue("metadata_json"))
		}
		if r.FormValue("content_type") != "image/png" {
			t.Fatalf("unexpected content_type field: %q", r.FormValue("content_type"))
		}
		if r.FormValue("visibility") != "unlisted" {
			t.Fatalf("unexpected visibility field: %q", r.FormValue("visibility"))
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("read multipart file: %v", err)
		}
		defer file.Close()

		if header.Filename != "chat-source.png" {
			t.Fatalf("unexpected upload filename: %q", header.Filename)
		}
		if header.Header.Get("Content-Type") != "image/png" {
			t.Fatalf("unexpected upload part content type: %q", header.Header.Get("Content-Type"))
		}
		body, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read upload body: %v", err)
		}
		if string(body) != "png-body" {
			t.Fatalf("unexpected upload body: %q", string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"data":{"id":"art_agent_upload_1","filename":"chat-source.png","content_type":"image/png","size_bytes":8,"created_at":"2026-03-28T04:00:00Z","expires_at":"2026-04-27T04:00:00Z","media_id":"med_agent_upload_1","url":"https://media.popi.test/a/art_agent_upload_1/chat-source.png","visibility":"unlisted","sha256":"demo-sha256","storage_status":"ready"}}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"artifacts", "upload", sourcePath,
		"--filename", "chat-source.png",
		"--role", "source",
		"--metadata-json", `{"origin":"agent-chat"}`,
		"--visibility", "unlisted",
	})

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected upload data object, got %#v", resp["data"])
	}
	if data["artifact_id"] != "art_agent_upload_1" {
		t.Fatalf("unexpected artifact_id: %#v", data["artifact_id"])
	}
	if data["project_id"] != "proj_agent_chat" {
		t.Fatalf("unexpected project_id: %#v", data["project_id"])
	}
	if data["role"] != "source" {
		t.Fatalf("unexpected role: %#v", data["role"])
	}
	if data["content_type"] != "image/png" {
		t.Fatalf("unexpected content_type: %#v", data["content_type"])
	}
	if data["media_id"] != "med_agent_upload_1" {
		t.Fatalf("unexpected media_id: %#v", data["media_id"])
	}
	if data["url"] != "https://media.popi.test/a/art_agent_upload_1/chat-source.png" {
		t.Fatalf("unexpected url: %#v", data["url"])
	}
	if data["visibility"] != "unlisted" {
		t.Fatalf("unexpected visibility: %#v", data["visibility"])
	}
	if data["storage_status"] != "ready" {
		t.Fatalf("unexpected storage_status: %#v", data["storage_status"])
	}
}

func TestArtifactsListRequiresJobIDHint(t *testing.T) {
	root := NewRootCmd("0.test")

	var stdout strings.Builder
	var stderr strings.Builder
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"artifacts", "list"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected artifacts list without job id to fail")
	}

	cliErr, ok := err.(*output.CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %T", err)
	}
	if cliErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %q", cliErr.Code)
	}
	if cliErr.Details["argument"] != "job-id" {
		t.Fatalf("unexpected argument detail: %#v", cliErr.Details["argument"])
	}
	hint, _ := cliErr.Details["hint"].(string)
	if !strings.Contains(hint, "artifacts get <artifact-id>") {
		t.Fatalf("unexpected hint: %q", hint)
	}
}
