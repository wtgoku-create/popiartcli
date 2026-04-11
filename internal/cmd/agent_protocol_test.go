package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wtgoku-create/popiartcli/internal/output"
)

func TestMCPDescribeSupportsOutputPlain(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())

	stdout, stderr, err := executeRootRaw(NewRootCmd("0.test"), []string{
		"mcp", "serve", "--describe", "--output", "plain",
	})
	if err != nil {
		t.Fatalf("mcp serve --describe --output plain failed: %v stderr=%s", err, stderr)
	}
	if strings.Contains(stdout, `"ok":`) {
		t.Fatalf("expected plain output, got %q", stdout)
	}
	if !strings.Contains(stdout, "server_name:") {
		t.Fatalf("expected plain describe output, got %q", stdout)
	}
}

func TestAuthLoginNonInteractiveRequiresKey(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())

	_, _, err := executeRootRaw(NewRootCmd("0.test"), []string{"auth", "login", "--non-interactive"})
	if err == nil {
		t.Fatal("expected auth login --non-interactive without key to fail")
	}
	cliErr, ok := err.(*output.CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %T", err)
	}
	if cliErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %#v", cliErr.Code)
	}
}

func TestRunDryRunPreviewsRequestWithoutSubmittingJob(t *testing.T) {
	t.Setenv("POPIART_CONFIG_DIR", t.TempDir())
	t.Setenv("POPIART_KEY", "pk-demo")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("dry-run should not hit server, got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()
	t.Setenv("POPIART_ENDPOINT", server.URL)

	resp := executeRootJSON(t, NewRootCmd("0.test"), []string{
		"run", officialText2ImageSkillID,
		"--input", `{"prompt":"agent dry-run"}`,
		"--dry-run",
	})

	data := resp["data"].(map[string]any)
	if data["dry_run"] != true {
		t.Fatalf("expected dry_run=true, got %#v", data["dry_run"])
	}
	request := data["request"].(map[string]any)
	if request["path"] != "/jobs" {
		t.Fatalf("unexpected preview path: %#v", request["path"])
	}
	body := request["body"].(map[string]any)
	if body["skill_id"] != officialText2ImageSkillID {
		t.Fatalf("unexpected skill_id: %#v", body["skill_id"])
	}
	agentProtocol := data["agent_protocol"].(map[string]any)
	if agentProtocol["output"] != "json" {
		t.Fatalf("unexpected output mode: %#v", agentProtocol["output"])
	}
	if agentProtocol["async"] != true {
		t.Fatalf("expected async preview by default, got %#v", agentProtocol["async"])
	}
}
