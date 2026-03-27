package cmd

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewRootCmdIncludesUpdate(t *testing.T) {
	root := NewRootCmd("0.1.0")

	found, _, err := root.Find([]string{"update"})
	if err != nil {
		t.Fatalf("Find(update) returned error: %v", err)
	}
	if found == nil || found.Name() != "update" {
		t.Fatalf("expected update command, got %#v", found)
	}
}

func TestShouldPersistGlobalOverridesSkipsUpdate(t *testing.T) {
	if shouldPersistGlobalOverrides(newUpdateCmd()) {
		t.Fatal("expected update command to skip config persistence")
	}
	if !shouldPersistGlobalOverrides(newRunCmd()) {
		t.Fatal("expected non-update command to persist config overrides")
	}
}

func TestUpdateCommandDoesNotPersistGlobalOverrides(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("POPIART_CONFIG_DIR", configDir)

	previousResolver := resolveUpdateTagFunc
	previousPathResolver := resolveExecutablePathsFunc
	previousRunner := selfUpdateRunner
	resolveUpdateTagFunc = func(ctx context.Context, repo, requested string) (string, error) {
		return "v0.2.0", nil
	}
	resolveExecutablePathsFunc = func() (string, string, error) {
		return "/tmp/popiart", "/tmp/popiart", nil
	}
	selfUpdateRunner = func(ctx context.Context, opts updateRunOptions) (updateRunResult, error) {
		return updateRunResult{
			Tag:            opts.Tag,
			ExecutablePath: opts.ExecutablePath,
		}, nil
	}
	defer func() {
		resolveUpdateTagFunc = previousResolver
		resolveExecutablePathsFunc = previousPathResolver
		selfUpdateRunner = previousRunner
	}()

	root := NewRootCmd("0.1.0")
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"--project", "demo-project", "update"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(configDir, "config.json")); !os.IsNotExist(err) {
		t.Fatalf("expected update to leave config untouched, got err=%v", err)
	}
}

func TestNormalizeReleaseTag(t *testing.T) {
	if got := normalizeReleaseTag("0.2.0"); got != "v0.2.0" {
		t.Fatalf("expected v0.2.0, got %q", got)
	}
	if got := normalizeReleaseTag("v0.2.0"); got != "v0.2.0" {
		t.Fatalf("expected v0.2.0, got %q", got)
	}
}

func TestNormalizeInstalledVersion(t *testing.T) {
	if got := normalizeInstalledVersion("0.2.0 (abc123) built 2026-03-27"); got != "v0.2.0" {
		t.Fatalf("expected v0.2.0, got %q", got)
	}
	if got := normalizeInstalledVersion("dev"); got != "dev" {
		t.Fatalf("expected dev, got %q", got)
	}
}

func TestIsHomebrewManagedExecutable(t *testing.T) {
	if !isHomebrewManagedExecutable("/opt/homebrew/Cellar/popiart/0.2.0/bin/popiart") {
		t.Fatal("expected Homebrew Cellar path to be detected")
	}
	if isHomebrewManagedExecutable("/usr/local/bin/popiart") {
		t.Fatal("expected regular path to not be detected as Homebrew-managed")
	}
}

func TestResolveTargetReleaseTagFollowsLatestRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/latest":
			http.Redirect(w, r, "/releases/tag/v1.2.3", http.StatusFound)
		case "/releases/tag/v1.2.3":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	previousClient := updateHTTPClient
	previousLatestURL := updateLatestReleaseURL
	updateHTTPClient = server.Client()
	updateLatestReleaseURL = func(repo string) string {
		return server.URL + "/releases/latest"
	}
	defer func() {
		updateHTTPClient = previousClient
		updateLatestReleaseURL = previousLatestURL
	}()

	tag, err := resolveTargetReleaseTag(context.Background(), "ignored/repo", "")
	if err != nil {
		t.Fatalf("resolveTargetReleaseTag returned error: %v", err)
	}
	if tag != "v1.2.3" {
		t.Fatalf("expected v1.2.3, got %q", tag)
	}
}

func TestDownloadTemporaryScript(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("#!/bin/sh\necho ok\n"))
	}))
	defer server.Close()

	path, cleanup, err := downloadTemporaryScript(context.Background(), server.URL, t.TempDir(), "install-*.sh", 0o700)
	if err != nil {
		t.Fatalf("downloadTemporaryScript returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(data) != "#!/bin/sh\necho ok\n" {
		t.Fatalf("unexpected script contents: %q", string(data))
	}

	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected cleanup to remove %q, got err=%v", path, err)
	}
}
