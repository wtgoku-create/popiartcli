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
		if r.URL.Path != "/api_client/media/upload" {
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
		// 兼容语义说明：artifacts upload 底层已切换到主站 media/upload，artifact_id 对应 media.id。
		fmt.Fprint(w, `{"data":{"id":105721,"type":1,"name":"chat-source.png","url":"https://static.popi.art/media/2026/0609/105721.png","createTime":"2026-06-09T18:27:37.262079707+08:00"},"message":"ok","status":"0000"}`)
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
	if data["artifact_id"] != "105721" {
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
	if data["media_id"] != "105721" {
		t.Fatalf("unexpected media_id: %#v", data["media_id"])
	}
	if data["url"] != "https://static.popi.art/media/2026/0609/105721.png" {
		t.Fatalf("unexpected url: %#v", data["url"])
	}
	if data["created_at"] != "2026-06-09T18:27:37.262079707+08:00" {
		t.Fatalf("unexpected created_at: %#v", data["created_at"])
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
	if !strings.Contains(hint, "artifacts list <task-id>") {
		t.Fatalf("unexpected hint: %q", hint)
	}
}

func TestArtifactsListUsesTaskDownloadURLs(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-artifacts")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/task/downloadUrls":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"downloadUrls":["https://cdn.example.com/result-1.png","https://cdn.example.com/result-2.mp4"]}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{"artifacts", "list", "task_123"})
	data := resp["data"].(map[string]any)
	if data["task_id"] != "task_123" {
		t.Fatalf("unexpected task_id: %#v", data["task_id"])
	}
	if data["total"] != float64(2) {
		t.Fatalf("unexpected total: %#v", data["total"])
	}
	items := data["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("unexpected items: %#v", data["items"])
	}
}

func TestArtifactsPullAllDownloadsTaskResults(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-artifacts")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/task/downloadUrls":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"downloadUrls":["`+serverURLForHost(r.Host)+`/download/result-1.png","`+serverURLForHost(r.Host)+`/download/result-2.txt"]}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/download/result-1.png":
			_, _ = w.Write([]byte("png-body"))
		case r.Method == http.MethodGet && r.URL.Path == "/download/result-2.txt":
			_, _ = w.Write([]byte("text-body"))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	dir := filepath.Join(t.TempDir(), "task-downloads")
	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{"artifacts", "pull-all", "task_456", "--dir", dir})
	data := resp["data"].(map[string]any)
	if data["artifacts_downloaded"] != float64(2) {
		t.Fatalf("unexpected artifacts_downloaded: %#v", data["artifacts_downloaded"])
	}
	for _, name := range []string{"result-1.png", "result-2.txt"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected downloaded file %s: %v", name, err)
		}
	}
}

func TestArtifactsGetUsesMediaDetailInPopiArtMode(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-artifacts")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api_client/media/detail" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("id") != "art_123" {
			t.Fatalf("unexpected artifact id query: %q", r.URL.Query().Get("id"))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"id":"art_123","project_id":"proj_demo","name":"result.png","content_type":"image/png","size_bytes":12,"createTime":"2026-06-11T16:00:00+08:00","fileUrl":"http://127.0.0.1:18080/v1/media/art_123/content","visibility":"unlisted","sha256":"sha-demo"},"message":"ok","status":"0000"}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{"artifacts", "get", "art_123"})
	data := resp["data"].(map[string]any)
	if data["artifact_id"] != "art_123" {
		t.Fatalf("unexpected artifact_id: %#v", data["artifact_id"])
	}
	if data["filename"] != "result.png" {
		t.Fatalf("unexpected filename: %#v", data["filename"])
	}
	if data["url"] != "https://server.popi.art/v1/media/art_123/content" {
		t.Fatalf("unexpected url: %#v", data["url"])
	}
	if data["original_url"] != "http://127.0.0.1:18080/v1/media/art_123/content" {
		t.Fatalf("unexpected original_url: %#v", data["original_url"])
	}
}

func TestArtifactsPullIsUnsupportedInPopiArtMode(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())

	_, _, err := executeRootRaw(NewRootCmd("0.test"), []string{"artifacts", "pull", "art_123"})
	if err == nil {
		t.Fatal("expected artifacts pull to fail")
	}
	cliErr, ok := err.(*output.CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %T", err)
	}
	if cliErr.Code != "UNSUPPORTED_IN_POPI_ART_MODE" {
		t.Fatalf("unexpected error code: %q", cliErr.Code)
	}
}

func serverURLForHost(host string) string {
	return "http://" + host
}
