package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/output"
)

func TestAuthCommandFlow(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)

	var loginBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth/login":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST /auth/login, got %s", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&loginBody); err != nil {
				t.Fatalf("decode login body: %v", err)
			}
			fmt.Fprint(w, `{"ok":true,"data":{"token":"sess_demo_auth_123456","user":{"id":"user_1","email":"demo@popi.art","name":"Demo"}}}`)
		case "/auth/me":
			if got := r.Header.Get("Authorization"); got != "Bearer sess_demo_auth_123456" {
				t.Fatalf("unexpected auth header for whoami: %q", got)
			}
			fmt.Fprint(w, `{"ok":true,"data":{"id":"user_1","email":"demo@popi.art","name":"Demo","scopes":["creator"]}}`)
		case "/auth/logout":
			if got := r.Header.Get("Authorization"); got != "Bearer sess_demo_auth_123456" {
				t.Fatalf("unexpected auth header for logout: %q", got)
			}
			fmt.Fprint(w, `{"ok":true,"data":{"logged_out":true}}`)
		default:
			t.Fatalf("unexpected auth path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	loginResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"auth", "login", "--key", "pk-demo"})
	loginData := loginResp["data"].(map[string]any)
	if loginData["key_saved"] != true {
		t.Fatalf("expected key_saved=true, got %#v", loginData["key_saved"])
	}
	if loginBody["key"] != "pk-demo" {
		t.Fatalf("unexpected login key payload: %#v", loginBody["key"])
	}

	showResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"auth", "key", "show"})
	showData := showResp["data"].(map[string]any)
	if showData["config"] != filepath.Join(configDir, "config.json") {
		t.Fatalf("unexpected config path: %#v", showData["config"])
	}
	if masked := showData["key"].(string); !strings.Contains(masked, "••••") {
		t.Fatalf("expected masked key output, got %q", masked)
	}

	whoamiResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"auth", "whoami"})
	whoamiData := whoamiResp["data"].(map[string]any)
	if whoamiData["email"] != "demo@popi.art" {
		t.Fatalf("unexpected whoami email: %#v", whoamiData["email"])
	}

	logoutResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"auth", "logout"})
	logoutData := logoutResp["data"].(map[string]any)
	if logoutData["logged_out"] != true {
		t.Fatalf("expected logged_out=true, got %#v", logoutData["logged_out"])
	}
}

func TestBootstrapCommandDiscoverableFlow(t *testing.T) {
	configDir := t.TempDir()
	codexHome := filepath.Join(t.TempDir(), "codex-home")

	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("CODEX_HOME", codexHome)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"bootstrap",
		"--agent", "codex",
		"--completion", "zsh",
		"--discoverable",
		"--key", "pk-bootstrap",
	})

	data := resp["data"].(map[string]any)
	if data["key_saved"] != true {
		t.Fatalf("expected key_saved=true, got %#v", data["key_saved"])
	}
	if data["default_skills_profile"] != "default" {
		t.Fatalf("unexpected default_skills_profile: %#v", data["default_skills_profile"])
	}
	if data["runtime_baseline"] != "runtime-baseline" {
		t.Fatalf("unexpected runtime_baseline: %#v", data["runtime_baseline"])
	}

	for _, path := range []string{
		filepath.Join(configDir, "bootstrap.json"),
		filepath.Join(configDir, "config.json"),
		filepath.Join(configDir, "agents", "codex", "env.sh"),
		filepath.Join(configDir, "agents", "codex", "mcp.json"),
		filepath.Join(codexHome, "config.toml"),
		filepath.Join(codexHome, "skills", "popiart", "SKILL.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected bootstrap asset %s to exist: %v", path, err)
		}
	}
}

func TestSetupCommandFlow(t *testing.T) {
	configDir := t.TempDir()
	codexHome := filepath.Join(t.TempDir(), "codex-home")

	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("CODEX_HOME", codexHome)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"setup",
		"--agent", "codex",
		"--completion", "zsh",
		"--key", "pk-setup",
	})

	data := resp["data"].(map[string]any)
	if data["default_skills_profile"] != "default" {
		t.Fatalf("unexpected default_skills_profile: %#v", data["default_skills_profile"])
	}
	if data["runtime_baseline"] != "runtime-baseline" {
		t.Fatalf("unexpected runtime_baseline: %#v", data["runtime_baseline"])
	}
	for _, path := range []string{
		filepath.Join(configDir, "bootstrap.json"),
		filepath.Join(configDir, "agents", "codex", "env.sh"),
		filepath.Join(codexHome, "config.toml"),
		filepath.Join(codexHome, "skills", "popiart", "SKILL.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected setup asset %s to exist: %v", path, err)
		}
	}
}

func TestBudgetCommands(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-budget")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/budget":
			fmt.Fprint(w, `{"ok":true,"data":{"remaining":42,"currency":"credits"}}`)
		case "/budget/usage":
			fmt.Fprint(w, `{"ok":true,"data":{"items":[{"skill_id":"popiskill-image-text2image-basic-v1","total":8}]}}`)
		case "/budget/limits":
			fmt.Fprint(w, `{"ok":true,"data":{"daily_limit":100,"burst":10}}`)
		default:
			t.Fatalf("unexpected budget path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	statusResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"budget", "status"})
	if statusResp["data"].(map[string]any)["remaining"] != float64(42) {
		t.Fatalf("unexpected budget status payload: %#v", statusResp["data"])
	}

	usageResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"budget", "usage", "--group-by", "skill"})
	items := usageResp["data"].(map[string]any)["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("unexpected budget usage items: %#v", items)
	}

	limitsResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"budget", "limits"})
	if limitsResp["data"].(map[string]any)["daily_limit"] != float64(100) {
		t.Fatalf("unexpected budget limits payload: %#v", limitsResp["data"])
	}
}

func TestJobsCommands(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-jobs")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/jobs/job_123" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_123","status":"done","artifact_ids":["art_1"]}}`)
		case r.URL.Path == "/jobs" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"items":[{"job_id":"job_123","status":"done"}],"total":1,"limit":20,"offset":0}}`)
		case r.URL.Path == "/jobs/job_123/cancel" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"job_id":"job_123","cancelled":true}}`)
		case r.URL.Path == "/jobs/job_123/logs" && r.Method == http.MethodGet && strings.Contains(r.Header.Get("Accept"), "text/event-stream"):
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "event: log\ndata: streaming\n\n")
		case r.URL.Path == "/jobs/job_123/logs" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":[{"ts":"2026-04-10T10:00:00Z","level":"info","message":"hello"}]}`)
		default:
			t.Fatalf("unexpected jobs path: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	getResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"jobs", "get", "job_123"})
	if getResp["data"].(map[string]any)["status"] != "done" {
		t.Fatalf("unexpected jobs get payload: %#v", getResp["data"])
	}

	waitResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"jobs", "wait", "job_123", "--interval", "1"})
	if waitResp["data"].(map[string]any)["job_id"] != "job_123" {
		t.Fatalf("unexpected jobs wait payload: %#v", waitResp["data"])
	}

	listResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"jobs", "list", "--status", "done"})
	if listResp["data"].(map[string]any)["total"] != float64(1) {
		t.Fatalf("unexpected jobs list payload: %#v", listResp["data"])
	}

	cancelResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"jobs", "cancel", "job_123"})
	if cancelResp["data"].(map[string]any)["cancelled"] != true {
		t.Fatalf("unexpected jobs cancel payload: %#v", cancelResp["data"])
	}

	logsResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"jobs", "logs", "job_123"})
	if len(logsResp["data"].([]any)) != 1 {
		t.Fatalf("unexpected jobs logs payload: %#v", logsResp["data"])
	}

	stdout, _, err := executeRootRaw(NewRootCmd("0.test"), []string{"jobs", "logs", "job_123", "--follow"})
	if err != nil {
		t.Fatalf("jobs logs --follow failed: %v", err)
	}
	if !strings.Contains(stdout, "streaming") {
		t.Fatalf("expected streamed logs output, got %q", stdout)
	}
}

func TestProjectCommands(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("POPIART_KEY", "pk-project")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/projects" && r.Method == http.MethodGet:
			fmt.Fprint(w, `{"ok":true,"data":{"items":[{"id":"proj_1","name":"Project One"}],"total":1}}`)
		case r.URL.Path == "/projects/proj_1" && r.Method == http.MethodGet:
			fmt.Fprint(w, `{"ok":true,"data":{"id":"proj_1","name":"Project One"}}`)
		case r.URL.Path == "/projects/proj_1/context" && r.Method == http.MethodGet:
			fmt.Fprint(w, `{"ok":true,"data":{"project_id":"proj_1","runtime":"ready"}}`)
		default:
			t.Fatalf("unexpected project path: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	currentResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"project", "current"})
	currentData := currentResp["data"].(map[string]any)
	if currentData["project"] != nil {
		t.Fatalf("expected nil project before selection, got %#v", currentData["project"])
	}

	useResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"project", "use", "proj_1"})
	if useResp["data"].(map[string]any)["project_set"] != "proj_1" {
		t.Fatalf("unexpected project use payload: %#v", useResp["data"])
	}

	currentResp = executeRootJSON(t, NewRootCmd("0.test"), []string{"project", "current"})
	if currentResp["data"].(map[string]any)["id"] != "proj_1" {
		t.Fatalf("unexpected project current payload: %#v", currentResp["data"])
	}

	listResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"project", "list"})
	if listResp["data"].(map[string]any)["total"] != float64(1) {
		t.Fatalf("unexpected project list payload: %#v", listResp["data"])
	}

	getResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"project", "get", "proj_1"})
	if getResp["data"].(map[string]any)["name"] != "Project One" {
		t.Fatalf("unexpected project get payload: %#v", getResp["data"])
	}

	getCurrentResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"project", "get"})
	if getCurrentResp["data"].(map[string]any)["id"] != "proj_1" {
		t.Fatalf("unexpected project get current payload: %#v", getCurrentResp["data"])
	}

	contextResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"project", "context"})
	if contextResp["data"].(map[string]any)["runtime"] != "ready" {
		t.Fatalf("unexpected project context payload: %#v", contextResp["data"])
	}
}

func TestProjectGetRequiresCurrentProjectWhenArgOmitted(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())

	root := NewRootCmd("0.test")
	_, _, err := executeRootRaw(root, []string{"project", "get"})
	if err == nil {
		t.Fatal("expected project get without current project to fail")
	}

	cliErr, ok := err.(*output.CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %T", err)
	}
	if cliErr.Code != "NO_PROJECT" {
		t.Fatalf("expected NO_PROJECT, got %q", cliErr.Code)
	}
}

func TestCompletionCommandGeneratesScript(t *testing.T) {
	stdout, stderr, err := executeRootRaw(NewRootCmd("0.test"), []string{"completion", "bash"})
	if err != nil {
		t.Fatalf("completion bash failed: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "popiart") {
		t.Fatalf("expected completion output to mention popiart, got %q", stdout)
	}
}

func TestMCPCommands(t *testing.T) {
	configDir := t.TempDir()
	codexHome := filepath.Join(t.TempDir(), "codex-home")
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("CODEX_HOME", codexHome)
	t.Setenv("POPIART_KEY", "pk-mcp")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/auth/me":
			fmt.Fprint(w, `{"ok":true,"data":{"id":"user_1","email":"mcp@popi.art","name":"MCP"}}`)
		case r.URL.Path == "/skills":
			fmt.Fprint(w, `{"ok":true,"data":{"items":[{"id":"popiskill-image-text2image-basic-v1","name":"Basic Text2Image"}],"total":1,"limit":1,"offset":0}}`)
		case strings.HasPrefix(r.URL.Path, "/skills/"):
			skillID := strings.TrimPrefix(r.URL.Path, "/skills/")
			fmt.Fprintf(w, `{"ok":true,"data":{"id":"%s","name":"%s","description":"runtime ready","input_schema":{"type":"object"},"output_schema":{"type":"object"}}}`, skillID, skillID)
		case r.URL.Path == "/models/routes":
			fmt.Fprint(w, `{"ok":true,"data":{"items":[{"route":"image.text2image","model":"demo-model"}]}}`)
		default:
			t.Fatalf("unexpected mcp doctor path: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	executeRootJSON(t, NewRootCmd("0.test"), []string{
		"bootstrap",
		"--agent", "codex",
		"--discoverable",
		"--key", "pk-mcp",
	})

	printConfigResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"mcp", "print-config", "--agent", "codex"})
	if printConfigResp["data"].(map[string]any)["server_name"] != popiartMCPServerName {
		t.Fatalf("unexpected mcp print-config payload: %#v", printConfigResp["data"])
	}

	describeResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"mcp", "serve", "--describe"})
	describeData := describeResp["data"].(map[string]any)
	if describeData["server_id"] != popiartMCPServerID {
		t.Fatalf("unexpected mcp describe payload: %#v", describeResp["data"])
	}

	doctorResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"mcp", "doctor", "--agent", "codex"})
	if doctorResp["data"].(map[string]any)["overall_status"] != "pass" {
		t.Fatalf("unexpected mcp doctor payload: %#v", doctorResp["data"])
	}
}

func executeRootRaw(root *cobra.Command, args []string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)
	root.SetContext(context.Background())
	err := root.Execute()
	return stdout.String(), stderr.String(), err
}
