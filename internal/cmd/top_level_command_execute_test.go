package cmd

import (
	"bytes"
	"context"
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api_client/users/user/info":
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET /api_client/users/user/info, got %s", r.Method)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer pk-demo" && got != "Bearer sess_demo_auth_123456" {
				t.Fatalf("unexpected auth header for user info: %q", got)
			}
			fmt.Fprint(w, `{"data":{"user":{"id":10561,"email":"demo@popi.art","name":"Demo"}},"message":"ok","status":"0000"}`)
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
	userData := loginData["user"].(map[string]any)
	if userData["email"] != "demo@popi.art" {
		t.Fatalf("unexpected login user payload: %#v", loginData["user"])
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

	_, _, err := executeRootRaw(NewRootCmd("0.test"), []string{"auth", "key", "rotate"})
	if err == nil {
		t.Fatal("expected auth key rotate to fail")
	}
	cliErr, ok := err.(*output.CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %T", err)
	}
	if cliErr.Code != "UNSUPPORTED_IN_POPI_ART_MODE" {
		t.Fatalf("unexpected rotate error code: %q", cliErr.Code)
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
	nextSteps, ok := data["next_steps"].([]any)
	if !ok || len(nextSteps) == 0 {
		t.Fatalf("expected setup next_steps, got %#v", data["next_steps"])
	}
	var sawDoctorStep bool
	var sawRuntimeNote bool
	for _, item := range nextSteps {
		step, _ := item.(string)
		if strings.Contains(step, "popiart mcp doctor --agent codex") {
			sawDoctorStep = true
		}
		if strings.Contains(step, "discoverability") && strings.Contains(step, "runtime_status") {
			sawRuntimeNote = true
		}
	}
	if !sawDoctorStep || !sawRuntimeNote {
		t.Fatalf("expected setup next_steps to explain doctor/runtime distinction, got %#v", nextSteps)
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

	for _, args := range [][]string{
		{"budget", "status"},
		{"budget", "usage", "--group-by", "skill"},
		{"budget", "limits"},
	} {
		_, _, err := executeRootRaw(NewRootCmd("0.test"), args)
		if err == nil {
			t.Fatalf("expected %v to be unsupported", args)
		}
		cliErr, ok := err.(*output.CLIError)
		if !ok || cliErr.Code != "UNSUPPORTED_IN_POPI_ART_MODE" {
			t.Fatalf("unexpected budget command error for %v: %#v", args, err)
		}
	}
}

func TestJobsCommands(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-jobs")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api_client/anime/task/detail" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"id":"job_123","status":2,"type":1,"subType":102,"aiModelCode":"seedream-4-5-251128"}}`)
		case r.URL.Path == "/api_client/anime/task/downloadUrls" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"downloadUrls":["https://cdn.example.com/result.png"]}}`)
		case r.URL.Path == "/api_client/anime/task/list" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true,"data":{"items":[{"id":"job_123","status":2,"type":1,"subType":102,"aiModelCode":"seedream-4-5-251128"},{"id":"job_124","status":0,"type":2,"subType":203,"aiModelCode":"viduq2-pro-fast"}],"total":2,"limit":20,"offset":0}}`)
		default:
			t.Fatalf("unexpected jobs path: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	getResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"jobs", "get", "job_123"})
	if getResp["data"].(map[string]any)["status"] != float64(2) {
		t.Fatalf("unexpected jobs get payload: %#v", getResp["data"])
	}

	waitResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"jobs", "wait", "job_123", "--interval", "1"})
	if waitResp["data"].(map[string]any)["job_id"] != "job_123" {
		t.Fatalf("unexpected jobs wait payload: %#v", waitResp["data"])
	}

	listResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"jobs", "list", "--status", "done"})
	listData := listResp["data"].(map[string]any)
	if listData["total"] != float64(2) {
		t.Fatalf("unexpected jobs list payload: %#v", listData)
	}
	items := listData["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("unexpected jobs list items: %#v", items)
	}

	_, _, err := executeRootRaw(NewRootCmd("0.test"), []string{"jobs", "cancel", "job_123"})
	if err == nil {
		t.Fatal("expected jobs cancel to be unsupported")
	}
	cliErr, ok := err.(*output.CLIError)
	if !ok || cliErr.Code != "UNSUPPORTED_IN_POPI_ART_MODE" {
		t.Fatalf("unexpected jobs cancel error: %#v", err)
	}

	_, _, err = executeRootRaw(NewRootCmd("0.test"), []string{"jobs", "logs", "job_123"})
	if err == nil {
		t.Fatal("expected jobs logs to be unsupported")
	}
	cliErr, ok = err.(*output.CLIError)
	if !ok || cliErr.Code != "UNSUPPORTED_IN_POPI_ART_MODE" {
		t.Fatalf("unexpected jobs logs error: %#v", err)
	}
}

func TestProjectCommands(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)
	t.Setenv("POPIART_KEY", "pk-project")

	currentResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"project", "current"})
	currentData := currentResp["data"].(map[string]any)
	if currentData["project"] != nil {
		t.Fatalf("expected nil project before selection, got %#v", currentData["project"])
	}

	useResp := executeRootJSON(t, NewRootCmd("0.test"), []string{"project", "use", "proj_1"})
	if useResp["data"].(map[string]any)["project_set"] != "proj_1" {
		t.Fatalf("unexpected project use payload: %#v", useResp["data"])
	}

	for _, args := range [][]string{
		{"project", "current"},
		{"project", "list"},
		{"project", "get", "proj_1"},
		{"project", "get"},
		{"project", "context"},
	} {
		_, _, err := executeRootRaw(NewRootCmd("0.test"), args)
		if err == nil {
			t.Fatalf("expected %v to be unsupported", args)
		}
		cliErr, ok := err.(*output.CLIError)
		if !ok || cliErr.Code != "UNSUPPORTED_IN_POPI_ART_MODE" {
			t.Fatalf("unexpected project command error for %v: %#v", args, err)
		}
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
		case r.URL.Path == "/api_client/users/user/info":
			fmt.Fprint(w, `{"data":{"user":{"id":10561,"email":"mcp@popi.art","name":"MCP"}},"message":"ok","status":"0000"}`)
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
	doctorData := doctorResp["data"].(map[string]any)
	if doctorData["overall_status"] != "pass" {
		t.Fatalf("unexpected mcp doctor payload: %#v", doctorResp["data"])
	}
	if doctorData["discoverability_status"] != "pass" {
		t.Fatalf("unexpected discoverability_status: %#v", doctorResp["data"])
	}
	if doctorData["runtime_status"] != "pass" {
		t.Fatalf("unexpected runtime_status: %#v", doctorResp["data"])
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
