package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestMCPServerInitializeAndToolsList(t *testing.T) {
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
	}, "\n")

	var out bytes.Buffer
	if err := runMCPServer(strings.NewReader(input), &out, &bytes.Buffer{}, "v0.test"); err != nil {
		t.Fatalf("runMCPServer returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 responses, got %d: %q", len(lines), out.String())
	}

	var initResp map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &initResp); err != nil {
		t.Fatalf("unmarshal init response: %v", err)
	}
	if initResp["jsonrpc"] != "2.0" {
		t.Fatalf("expected jsonrpc 2.0, got %#v", initResp["jsonrpc"])
	}
	result, ok := initResp["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected initialize result object, got %#v", initResp["result"])
	}
	if result["protocolVersion"] != mcpProtocolVersion {
		t.Fatalf("expected protocol version %q, got %#v", mcpProtocolVersion, result["protocolVersion"])
	}

	var toolsResp map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &toolsResp); err != nil {
		t.Fatalf("unmarshal tools response: %v", err)
	}
	toolsResult, ok := toolsResp["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected tools/list result object, got %#v", toolsResp["result"])
	}
	tools, ok := toolsResult["tools"].([]any)
	if !ok {
		t.Fatalf("expected tools array, got %#v", toolsResult["tools"])
	}
	if len(tools) == 0 {
		t.Fatal("expected at least one tool")
	}
	var foundArtifactUpload bool
	var foundMediaUpload bool
	for _, item := range tools {
		tool, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if tool["name"] == "upload_artifact" {
			foundArtifactUpload = true
		}
		if tool["name"] == "upload_media" {
			foundMediaUpload = true
		}
	}
	if !foundArtifactUpload || !foundMediaUpload {
		t.Fatalf("expected tools/list to include upload_artifact and upload_media, got %#v", tools)
	}
}

func TestMCPServerCurrentProjectToolCall(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("POPIART_ENDPOINT", "https://example.com/v1")
	t.Setenv("POPIART_PROJECT", "demo-project")
	t.Setenv("POPIART_KEY", "pk-demo")

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`,
		`{"jsonrpc":"2.0","id":"call","method":"tools/call","params":{"name":"current_project","arguments":{}}}`,
	}, "\n")

	var out bytes.Buffer
	if err := runMCPServer(strings.NewReader(input), &out, &bytes.Buffer{}, "v0.test"); err != nil {
		t.Fatalf("runMCPServer returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 responses, got %d: %q", len(lines), out.String())
	}

	var callResp map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &callResp); err != nil {
		t.Fatalf("unmarshal tool call response: %v", err)
	}
	result, ok := callResp["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result object, got %#v", callResp["result"])
	}
	if result["isError"] != nil {
		t.Fatalf("expected success result, got %#v", result)
	}
	structured, ok := result["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("expected structuredContent object, got %#v", result["structuredContent"])
	}
	data, ok := structured["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %#v", structured["data"])
	}
	if data["project"] != "demo-project" {
		t.Fatalf("expected project demo-project, got %#v", data["project"])
	}
}

func TestMCPServerSupportsHeaderFraming(t *testing.T) {
	payload := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`
	input := "Content-Length: " + strconv.Itoa(len(payload)) + "\r\n\r\n" + payload

	var out bytes.Buffer
	if err := runMCPServer(strings.NewReader(input), &out, &bytes.Buffer{}, "v0.test"); err != nil {
		t.Fatalf("runMCPServer returned error: %v", err)
	}
	if !strings.HasPrefix(out.String(), "Content-Length: ") {
		t.Fatalf("expected header-framed response, got %q", out.String())
	}
}

func TestMCPServerUploadArtifactToolCall(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("POPIART_KEY", "pk-demo")

	uploadPath := filepath.Join(t.TempDir(), "chat-upload.png")
	if err := os.WriteFile(uploadPath, []byte("png-body"), 0o644); err != nil {
		t.Fatalf("write upload file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api_client/media/upload" {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"ok":false,"error":{"code":"NOT_FOUND","message":"not found"}}`)
			return
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart form: %v", err)
		}
		if r.FormValue("role") != "source" {
			t.Fatalf("unexpected role: %q", r.FormValue("role"))
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("read form file: %v", err)
		}
		defer file.Close()

		if header.Filename != "agent-chat.png" {
			t.Fatalf("unexpected filename: %q", header.Filename)
		}
		if header.Header.Get("Content-Type") != "image/png" {
			t.Fatalf("unexpected part content type: %q", header.Header.Get("Content-Type"))
		}
		body, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read upload body: %v", err)
		}
		if string(body) != "png-body" {
			t.Fatalf("unexpected upload body: %q", string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		// 兼容语义说明：MCP 的 upload_artifact 仍保留旧工具名，但底层已经统一走主站 media/upload。
		fmt.Fprint(w, `{"data":{"id":105722,"type":1,"name":"agent-chat.png","url":"https://static.popi.art/media/2026/0609/105722.png","createTime":"2026-06-09T18:28:37.262079707+08:00"},"message":"ok","status":"0000"}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	argsJSON, err := json.Marshal(map[string]any{
		"path":          uploadPath,
		"filename":      "agent-chat.png",
		"role":          "source",
		"metadata_json": `{"origin":"chat-upload"}`,
		"visibility":    "unlisted",
	})
	if err != nil {
		t.Fatalf("marshal arguments: %v", err)
	}

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`,
		fmt.Sprintf(`{"jsonrpc":"2.0","id":"call","method":"tools/call","params":{"name":"upload_artifact","arguments":%s}}`, argsJSON),
	}, "\n")

	var out bytes.Buffer
	if err := runMCPServer(strings.NewReader(input), &out, &bytes.Buffer{}, "v0.test"); err != nil {
		t.Fatalf("runMCPServer returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 responses, got %d: %q", len(lines), out.String())
	}

	var callResp map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &callResp); err != nil {
		t.Fatalf("unmarshal tool call response: %v", err)
	}
	result, ok := callResp["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result object, got %#v", callResp["result"])
	}
	if result["isError"] != nil {
		t.Fatalf("expected success result, got %#v", result)
	}
	structured, ok := result["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("expected structuredContent object, got %#v", result["structuredContent"])
	}
	data, ok := structured["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %#v", structured["data"])
	}
	if data["artifact_id"] != "105722" {
		t.Fatalf("unexpected artifact_id: %#v", data["artifact_id"])
	}
	if data["content_type"] != "image/png" {
		t.Fatalf("unexpected content_type: %#v", data["content_type"])
	}
	if data["media_id"] != "105722" {
		t.Fatalf("unexpected media_id: %#v", data["media_id"])
	}
	if data["url"] != "https://static.popi.art/media/2026/0609/105722.png" {
		t.Fatalf("unexpected url: %#v", data["url"])
	}
}

func TestMCPServerUploadMediaToolCall(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("POPIART_KEY", "pk-demo")

	uploadPath := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(uploadPath, []byte("png-body"), 0o644); err != nil {
		t.Fatalf("write upload file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api_client/media/upload" {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"ok":false,"error":{"code":"NOT_FOUND","message":"not found"}}`)
			return
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart form: %v", err)
		}
		if r.FormValue("visibility") != "public" {
			t.Fatalf("unexpected visibility: %q", r.FormValue("visibility"))
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("read form file: %v", err)
		}
		defer file.Close()

		if header.Filename != "source.png" {
			t.Fatalf("unexpected filename: %q", header.Filename)
		}
		body, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read upload body: %v", err)
		}
		if string(body) != "png-body" {
			t.Fatalf("unexpected upload body: %q", string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"id":105723,"type":1,"name":"source.png","url":"https://static.popi.art/media/2026/0609/105723.png","createTime":"2026-06-09T18:29:37.262079707+08:00"},"message":"ok","status":"0000"}`)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)
	t.Setenv("POPIART_PROJECT", "proj_demo")

	argsJSON, err := json.Marshal(map[string]any{
		"path":          uploadPath,
		"filename":      "source.png",
		"metadata_json": `{"origin":"agent-upload"}`,
		"visibility":    "public",
	})
	if err != nil {
		t.Fatalf("marshal arguments: %v", err)
	}

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`,
		fmt.Sprintf(`{"jsonrpc":"2.0","id":"call","method":"tools/call","params":{"name":"upload_media","arguments":%s}}`, argsJSON),
	}, "\n")

	var out bytes.Buffer
	if err := runMCPServer(strings.NewReader(input), &out, &bytes.Buffer{}, "v0.test"); err != nil {
		t.Fatalf("runMCPServer returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 responses, got %d: %q", len(lines), out.String())
	}

	var callResp map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &callResp); err != nil {
		t.Fatalf("unmarshal tool call response: %v", err)
	}
	result, ok := callResp["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result object, got %#v", callResp["result"])
	}
	if result["isError"] != nil {
		t.Fatalf("expected success result, got %#v", result)
	}
	structured, ok := result["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("expected structuredContent object, got %#v", result["structuredContent"])
	}
	data, ok := structured["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %#v", structured["data"])
	}
	if data["media_id"] != "105723" {
		t.Fatalf("unexpected media_id: %#v", data["media_id"])
	}
	if data["url"] != "https://static.popi.art/media/2026/0609/105723.png" {
		t.Fatalf("unexpected url: %#v", data["url"])
	}
}

func TestMCPServerWhoamiAndJobToolsUseMainSiteTaskEndpoints(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/users/user/info":
			fmt.Fprint(w, `{"data":{"user":{"id":10561,"email":"demo@popi.art","name":"Demo"}},"message":"ok","status":"0000"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/task/detail":
			fmt.Fprint(w, `{"ok":true,"data":{"id":"task_1","status":2,"type":2,"subType":203,"aiModelCode":"viduq2-pro-fast"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api_client/anime/task/downloadUrls":
			fmt.Fprint(w, `{"ok":true,"data":{"downloadUrls":["https://cdn.example.com/result.mp4"]}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	cases := []struct {
		name      string
		arguments map[string]any
		assert    func(t *testing.T, data map[string]any)
	}{
		{
			name:      "whoami",
			arguments: map[string]any{},
			assert: func(t *testing.T, data map[string]any) {
				if data["email"] != "demo@popi.art" {
					t.Fatalf("unexpected whoami payload: %#v", data)
				}
			},
		},
		{
			name:      "get_job",
			arguments: map[string]any{"job_id": "task_1"},
			assert: func(t *testing.T, data map[string]any) {
				if data["job_id"] != "task_1" {
					t.Fatalf("unexpected get_job payload: %#v", data)
				}
				urls := data["download_urls"].([]any)
				if len(urls) != 1 || urls[0] != "https://cdn.example.com/result.mp4" {
					t.Fatalf("unexpected get_job urls: %#v", data)
				}
			},
		},
		{
			name:      "wait_job",
			arguments: map[string]any{"job_id": "task_1", "interval_millis": 1},
			assert: func(t *testing.T, data map[string]any) {
				if data["task_id"] != "task_1" {
					t.Fatalf("unexpected wait_job payload: %#v", data)
				}
			},
		},
		{
			name:      "list_artifacts",
			arguments: map[string]any{"job_id": "task_1"},
			assert: func(t *testing.T, data map[string]any) {
				if data["total"] != float64(1) {
					t.Fatalf("unexpected list_artifacts payload: %#v", data)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			argsJSON, err := json.Marshal(tc.arguments)
			if err != nil {
				t.Fatalf("marshal arguments: %v", err)
			}

			input := strings.Join([]string{
				`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`,
				fmt.Sprintf(`{"jsonrpc":"2.0","id":"call","method":"tools/call","params":{"name":"%s","arguments":%s}}`, tc.name, argsJSON),
			}, "\n")

			var out bytes.Buffer
			if err := runMCPServer(strings.NewReader(input), &out, &bytes.Buffer{}, "v0.test"); err != nil {
				t.Fatalf("runMCPServer returned error: %v", err)
			}

			lines := strings.Split(strings.TrimSpace(out.String()), "\n")
			if len(lines) != 2 {
				t.Fatalf("expected 2 responses, got %d: %q", len(lines), out.String())
			}

			var callResp map[string]any
			if err := json.Unmarshal([]byte(lines[1]), &callResp); err != nil {
				t.Fatalf("unmarshal tool call response: %v", err)
			}
			result, ok := callResp["result"].(map[string]any)
			if !ok {
				t.Fatalf("expected result object, got %#v", callResp["result"])
			}
			if result["isError"] != nil {
				t.Fatalf("expected success result, got %#v", result)
			}
			structured, ok := result["structuredContent"].(map[string]any)
			if !ok {
				t.Fatalf("expected structuredContent object, got %#v", result["structuredContent"])
			}
			data, ok := structured["data"].(map[string]any)
			if !ok {
				t.Fatalf("expected data object, got %#v", structured["data"])
			}
			tc.assert(t, data)
		})
	}
}

func TestMCPServerGetJobLogsReturnsUnsupportedInPopiArtMode(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	argsJSON, err := json.Marshal(map[string]any{"job_id": "task_1"})
	if err != nil {
		t.Fatalf("marshal arguments: %v", err)
	}

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`,
		fmt.Sprintf(`{"jsonrpc":"2.0","id":"call","method":"tools/call","params":{"name":"get_job_logs","arguments":%s}}`, argsJSON),
	}, "\n")

	var out bytes.Buffer
	if err := runMCPServer(strings.NewReader(input), &out, &bytes.Buffer{}, "v0.test"); err != nil {
		t.Fatalf("runMCPServer returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 responses, got %d: %q", len(lines), out.String())
	}

	var callResp map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &callResp); err != nil {
		t.Fatalf("unmarshal tool call response: %v", err)
	}
	result, ok := callResp["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result object, got %#v", callResp["result"])
	}
	if result["isError"] != true {
		t.Fatalf("expected MCP error result, got %#v", result)
	}
	structured, ok := result["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("expected structuredContent object, got %#v", result["structuredContent"])
	}
	errPayload := structured["error"].(map[string]any)
	if errPayload["code"] != "UNSUPPORTED_IN_POPI_ART_MODE" {
		t.Fatalf("unexpected error payload: %#v", errPayload)
	}
}
