package cmd

import (
	"bytes"
	"encoding/json"
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
