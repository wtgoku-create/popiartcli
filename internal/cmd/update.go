package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/output"
)

type updateSource string

const (
	updateSourceGitHub updateSource = "github"
	updateSourceGitee  updateSource = "gitee"

	defaultUpdateRepoGitHub = "wtgoku-create/popiartcli"
	defaultUpdateRepoGitee  = "wattx/popiartcli"
)

var updateHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

var updateLatestReleaseURL = func(source updateSource, repo string) string {
	switch source {
	case updateSourceGitee:
		return fmt.Sprintf("https://gitee.com/api/v5/repos/%s/releases/latest", repo)
	default:
		return fmt.Sprintf("https://github.com/%s/releases/latest", repo)
	}
}

var resolveUpdateTagFunc = resolveTargetReleaseTag
var resolveExecutablePathsFunc = resolveExecutablePaths
var selfUpdateRunner = runSelfUpdate

func newUpdateCmd() *cobra.Command {
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "从 GitHub 或 Gitee Releases 下载并安装最新版本",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			repoInput := strings.TrimSpace(flagString(cmd, "repo"))
			if repoInput == "" {
				repoInput = strings.TrimSpace(os.Getenv("POPIART_REPO"))
			}

			sourceInput := strings.TrimSpace(flagString(cmd, "source"))
			if sourceInput == "" {
				sourceInput = strings.TrimSpace(os.Getenv("POPIART_SOURCE"))
			}

			source, repo, requestedVersion, err := normalizeUpdateTargetInput(repoInput, sourceInput, flagString(cmd, "version"))
			if err != nil {
				return err
			}

			targetTag, err := resolveUpdateTagFunc(ctx, source, repo, requestedVersion)
			if err != nil {
				return err
			}

			currentVersion := normalizeInstalledVersion(cmd.Root().Version)
			_, executablePath, err := resolveExecutablePathsFunc()
			if err != nil {
				return output.NewError("CLI_ERROR", "解析当前可执行文件失败", map[string]any{
					"details": err.Error(),
				})
			}

			if isHomebrewManagedExecutable(executablePath) {
				return output.NewError("UNSUPPORTED_INSTALL", "当前安装由 Homebrew 管理，请改用 brew upgrade wtgoku-create/popi/popiart", map[string]any{
					"path": executablePath,
				})
			}

			expectedName := expectedExecutableName()
			if filepath.Base(executablePath) != expectedName {
				return output.NewError("UNSUPPORTED_INSTALL", "当前可执行文件名不是标准发布名，无法安全更新", map[string]any{
					"path":     executablePath,
					"expected": expectedName,
				})
			}

			if currentVersion != "" && currentVersion == targetTag {
				return writeOutput(cmd, map[string]any{
					"updated":          false,
					"current_version":  currentVersion,
					"target_version":   targetTag,
					"source":           string(source),
					"repo":             repo,
					"path":             executablePath,
					"config_unchanged": true,
				})
			}

			updateResult, err := selfUpdateRunner(ctx, updateRunOptions{
				Source:         source,
				Repo:           repo,
				Tag:            targetTag,
				ExecutablePath: executablePath,
				Stdout:         cmd.ErrOrStderr(),
				Stderr:         cmd.ErrOrStderr(),
			})
			if err != nil {
				return err
			}

			return writeOutput(cmd, map[string]any{
				"updated":              true,
				"current_version":      currentVersion,
				"target_version":       updateResult.Tag,
				"source":               string(source),
				"repo":                 repo,
				"path":                 updateResult.ExecutablePath,
				"restart_required":     updateResult.RestartRequired,
				"config_unchanged":     true,
				"installer_script_url": updateResult.ScriptURL,
			})
		},
	}

	updateCmd.Flags().String("version", "", "更新到指定版本，例如 v0.1.0；默认使用最新 release")
	updateCmd.Flags().String("repo", "", "覆盖仓库，支持 owner/name、GitHub/Gitee 仓库主页或 tag URL")
	updateCmd.Flags().String("source", "", "覆盖下载源：github 或 gitee")

	return updateCmd
}

type updateRunOptions struct {
	Source         updateSource
	Repo           string
	Tag            string
	ExecutablePath string
	Stdout         io.Writer
	Stderr         io.Writer
}

type updateRunResult struct {
	Tag             string
	ExecutablePath  string
	RestartRequired bool
	ScriptURL       string
}

func runSelfUpdate(ctx context.Context, opts updateRunOptions) (updateRunResult, error) {
	if runtime.GOOS == "windows" {
		return runWindowsSelfUpdate(ctx, opts)
	}
	return runUnixSelfUpdate(ctx, opts)
}

func runUnixSelfUpdate(ctx context.Context, opts updateRunOptions) (updateRunResult, error) {
	scriptURL := rawContentURL(opts.Source, opts.Repo, opts.Tag, "install.sh")
	scriptPath, cleanup, err := downloadTemporaryScript(ctx, scriptURL, "", "popiart-install-*.sh", 0o700)
	if err != nil {
		return updateRunResult{}, err
	}
	defer cleanup()

	command := exec.CommandContext(ctx, "sh", scriptPath, "--cli-only", "--version", opts.Tag)
	command.Env = append(os.Environ(),
		"BINDIR="+filepath.Dir(opts.ExecutablePath),
		"POPIART_SOURCE="+string(opts.Source),
		"POPIART_REPO="+opts.Repo,
	)
	command.Stdout = opts.Stdout
	command.Stderr = opts.Stderr
	command.Stdin = os.Stdin

	if err := command.Run(); err != nil {
		return updateRunResult{}, output.NewError("UPDATE_FAILED", "安装最新版本失败", map[string]any{
			"details": err.Error(),
			"version": opts.Tag,
		})
	}

	return updateRunResult{
		Tag:            opts.Tag,
		ExecutablePath: opts.ExecutablePath,
		ScriptURL:      scriptURL,
	}, nil
}

func runWindowsSelfUpdate(ctx context.Context, opts updateRunOptions) (updateRunResult, error) {
	_ = ctx

	tempDir, err := os.MkdirTemp("", "popiart-update-*")
	if err != nil {
		return updateRunResult{}, output.NewError("CLI_ERROR", "创建临时目录失败", map[string]any{
			"details": err.Error(),
		})
	}
	cleanupTempDir := func() {
		_ = os.RemoveAll(tempDir)
	}

	scriptURL := rawContentURL(opts.Source, opts.Repo, opts.Tag, "install.ps1")
	installerPath, _, err := downloadTemporaryScript(context.Background(), scriptURL, tempDir, "install-*.ps1", 0o600)
	if err != nil {
		cleanupTempDir()
		return updateRunResult{}, err
	}

	wrapperPath := filepath.Join(tempDir, "run-update.ps1")
	wrapper := `param(
  [int]$CurrentPid,
  [string]$InstallerPath,
  [string]$InstallDir,
  [string]$Source,
  [string]$Repo,
  [string]$Version
)

$ErrorActionPreference = "Stop"

while (Get-Process -Id $CurrentPid -ErrorAction SilentlyContinue) {
  Start-Sleep -Milliseconds 500
}

$env:POPIART_REPO = $Repo
$env:POPIART_SOURCE = $Source

$installerArgs = @(
  "-NoProfile",
  "-ExecutionPolicy", "Bypass",
  "-File", $InstallerPath,
  "-CliOnly",
  "-InstallDir", $InstallDir
)

if ($Version) {
  $installerArgs += @("-Version", $Version)
}

& powershell.exe @installerArgs

Remove-Item -Path $InstallerPath -Force -ErrorAction SilentlyContinue
Remove-Item -Path $PSCommandPath -Force -ErrorAction SilentlyContinue
`
	if err := os.WriteFile(wrapperPath, []byte(wrapper), 0o600); err != nil {
		cleanupTempDir()
		return updateRunResult{}, output.NewError("CLI_ERROR", "写入 Windows 更新脚本失败", map[string]any{
			"details": err.Error(),
		})
	}

	command := exec.Command(
		"powershell.exe",
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-File", wrapperPath,
		"-CurrentPid", fmt.Sprintf("%d", os.Getpid()),
		"-InstallerPath", installerPath,
		"-InstallDir", filepath.Dir(opts.ExecutablePath),
		"-Source", string(opts.Source),
		"-Repo", opts.Repo,
		"-Version", opts.Tag,
	)
	command.Stdout = opts.Stdout
	command.Stderr = opts.Stderr
	command.Stdin = nil

	if err := command.Start(); err != nil {
		cleanupTempDir()
		return updateRunResult{}, output.NewError("UPDATE_FAILED", "启动后台更新进程失败", map[string]any{
			"details": err.Error(),
			"version": opts.Tag,
		})
	}
	_ = command.Process.Release()

	return updateRunResult{
		Tag:             opts.Tag,
		ExecutablePath:  opts.ExecutablePath,
		RestartRequired: true,
		ScriptURL:       scriptURL,
	}, nil
}

func resolveTargetReleaseTag(ctx context.Context, source updateSource, repo, requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested != "" {
		return normalizeReleaseTag(requested), nil
	}

	url := updateLatestReleaseURL(source, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", output.NewError("CLI_ERROR", "创建更新请求失败", map[string]any{
			"details": err.Error(),
		})
	}
	req.Header.Set("User-Agent", "popiart-cli-updater")

	resp, err := updateHTTPClient.Do(req)
	if err != nil {
		return "", output.NewError("NETWORK_ERROR", "获取最新 release 失败", map[string]any{
			"details": err.Error(),
			"repo":    repo,
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", output.NewError("NETWORK_ERROR", "获取最新 release 失败", map[string]any{
			"source": string(source),
			"repo":   repo,
			"status": resp.Status,
		})
	}

	if source == updateSourceGitee {
		var payload struct {
			TagName string `json:"tag_name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return "", output.NewError("NETWORK_ERROR", "解析最新 release 版本失败", map[string]any{
				"source":  string(source),
				"repo":    repo,
				"url":     url,
				"details": err.Error(),
			})
		}
		tag := normalizeReleaseTag(payload.TagName)
		if tag == "" {
			return "", output.NewError("NETWORK_ERROR", "解析最新 release 版本失败", map[string]any{
				"source": string(source),
				"repo":   repo,
				"url":    url,
			})
		}
		return tag, nil
	}

	tag := path.Base(resp.Request.URL.Path)
	if tag == "" || tag == "latest" || tag == "." || tag == "/" {
		return "", output.NewError("NETWORK_ERROR", "解析最新 release 版本失败", map[string]any{
			"source": string(source),
			"repo":   repo,
			"url":    resp.Request.URL.String(),
		})
	}

	return normalizeReleaseTag(tag), nil
}

func normalizeReleaseTag(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return ""
	}
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

func normalizeUpdateTargetInput(repoInput, sourceInput, requestedVersion string) (updateSource, string, string, error) {
	requestedVersion = strings.TrimSpace(requestedVersion)
	source, err := normalizeUpdateSource(sourceInput)
	if err != nil {
		return "", "", "", output.NewError("VALIDATION_ERROR", "无效的下载源", map[string]any{
			"source":  sourceInput,
			"details": err.Error(),
		})
	}

	if strings.TrimSpace(repoInput) == "" {
		return source, defaultUpdateRepoForSource(source), requestedVersion, nil
	}

	resolvedSource, repo, inferredTag, err := parseRepoReference(repoInput, source)
	if err != nil {
		return "", "", "", output.NewError("VALIDATION_ERROR", "无效的 GitHub 或 Gitee 仓库 / tag URL", map[string]any{
			"repo":    repoInput,
			"details": err.Error(),
		})
	}
	if requestedVersion == "" && inferredTag != "" {
		requestedVersion = inferredTag
	}
	return resolvedSource, repo, requestedVersion, nil
}

func parseRepoReference(value string, defaultSource updateSource) (updateSource, string, string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", "", fmt.Errorf("empty repository reference")
	}

	lowerValue := strings.ToLower(value)
	if !strings.Contains(value, "://") &&
		!strings.HasPrefix(lowerValue, "github.com/") &&
		!strings.HasPrefix(lowerValue, "gitee.com/") {
		repo := strings.Trim(strings.TrimSuffix(value, ".git"), "/")
		parts := strings.Split(repo, "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", "", fmt.Errorf("expected owner/name")
		}
		return defaultSource, parts[0] + "/" + parts[1], "", nil
	}

	if strings.HasPrefix(lowerValue, "github.com/") || strings.HasPrefix(lowerValue, "gitee.com/") {
		value = "https://" + value
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", "", "", err
	}
	host := strings.ToLower(parsed.Hostname())
	source, err := sourceFromHostname(host)
	if err != nil {
		return "", "", "", err
	}

	pathParts := splitNonEmpty(strings.Trim(parsed.Path, "/"), "/")
	if len(pathParts) < 2 {
		return "", "", "", fmt.Errorf("expected owner/repo path")
	}

	repo := pathParts[0] + "/" + strings.TrimSuffix(pathParts[1], ".git")
	inferredTag := ""

	switch {
	case len(pathParts) >= 5 && pathParts[2] == "releases" && pathParts[3] == "tag":
		inferredTag = pathParts[4]
	case len(pathParts) >= 6 && pathParts[2] == "archive" && pathParts[3] == "refs" && pathParts[4] == "tags":
		inferredTag = trimArchiveTag(pathParts[5])
	case len(pathParts) >= 4 && pathParts[2] == "archive":
		inferredTag = trimArchiveTag(pathParts[3])
	}

	return source, repo, normalizeReleaseTag(inferredTag), nil
}

func trimArchiveTag(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, ".tar.gz")
	name = strings.TrimSuffix(name, ".tgz")
	name = strings.TrimSuffix(name, ".zip")
	return name
}

func normalizeUpdateSource(value string) (updateSource, error) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", string(updateSourceGitHub):
		return updateSourceGitHub, nil
	case string(updateSourceGitee):
		return updateSourceGitee, nil
	default:
		return "", fmt.Errorf("expected github or gitee")
	}
}

func defaultUpdateRepoForSource(source updateSource) string {
	if source == updateSourceGitee {
		return defaultUpdateRepoGitee
	}
	return defaultUpdateRepoGitHub
}

func sourceFromHostname(host string) (updateSource, error) {
	switch host {
	case "github.com", "www.github.com":
		return updateSourceGitHub, nil
	case "gitee.com", "www.gitee.com":
		return updateSourceGitee, nil
	default:
		return "", fmt.Errorf("unsupported host %q", host)
	}
}

func splitNonEmpty(value, sep string) []string {
	if value == "" {
		return nil
	}
	raw := strings.Split(value, sep)
	parts := make([]string, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item != "" {
			parts = append(parts, item)
		}
	}
	return parts
}

func normalizeInstalledVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return ""
	}

	parts := strings.Fields(version)
	if len(parts) == 0 {
		return ""
	}

	current := parts[0]
	if current == "dev" {
		return current
	}
	return normalizeReleaseTag(current)
}

func resolveExecutablePaths() (string, string, error) {
	var candidate string
	if os.Args[0] != "" {
		if lookedUp, err := exec.LookPath(os.Args[0]); err == nil {
			candidate = lookedUp
		}
	}

	if candidate == "" {
		executablePath, err := os.Executable()
		if err != nil {
			return "", "", err
		}
		candidate = executablePath
	}

	absolutePath, err := filepath.Abs(candidate)
	if err != nil {
		return "", "", err
	}

	resolvedPath := absolutePath
	if realPath, err := filepath.EvalSymlinks(absolutePath); err == nil {
		resolvedPath = realPath
	}

	return absolutePath, resolvedPath, nil
}

func expectedExecutableName() string {
	if runtime.GOOS == "windows" {
		return "popiart.exe"
	}
	return "popiart"
}

func isHomebrewManagedExecutable(executablePath string) bool {
	normalized := filepath.ToSlash(executablePath)
	return strings.Contains(normalized, "/Cellar/") || strings.Contains(normalized, "/Homebrew/Cellar/")
}

func rawContentURL(source updateSource, repo, tag, name string) string {
	switch source {
	case updateSourceGitee:
		return fmt.Sprintf("https://gitee.com/%s/raw/%s/%s", repo, tag, name)
	default:
		return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", repo, tag, name)
	}
}

func downloadTemporaryScript(ctx context.Context, url, dir, pattern string, mode os.FileMode) (string, func(), error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", nil, output.NewError("CLI_ERROR", "创建更新请求失败", map[string]any{
			"details": err.Error(),
			"url":     url,
		})
	}
	req.Header.Set("User-Agent", "popiart-cli-updater")

	resp, err := updateHTTPClient.Do(req)
	if err != nil {
		return "", nil, output.NewError("NETWORK_ERROR", "下载更新脚本失败", map[string]any{
			"details": err.Error(),
			"url":     url,
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, output.NewError("NETWORK_ERROR", "下载更新脚本失败", map[string]any{
			"url":    url,
			"status": resp.Status,
		})
	}

	file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", nil, output.NewError("CLI_ERROR", "创建临时更新脚本失败", map[string]any{
			"details": err.Error(),
		})
	}

	cleanup := func() {
		_ = os.Remove(file.Name())
	}

	if _, err := io.Copy(file, resp.Body); err != nil {
		file.Close()
		cleanup()
		return "", nil, output.NewError("NETWORK_ERROR", "写入更新脚本失败", map[string]any{
			"details": err.Error(),
			"url":     url,
		})
	}

	if err := file.Close(); err != nil {
		cleanup()
		return "", nil, output.NewError("CLI_ERROR", "关闭更新脚本失败", map[string]any{
			"details": err.Error(),
		})
	}

	if err := os.Chmod(file.Name(), mode); err != nil {
		cleanup()
		return "", nil, output.NewError("CLI_ERROR", "设置更新脚本权限失败", map[string]any{
			"details": err.Error(),
			"path":    file.Name(),
		})
	}

	return file.Name(), cleanup, nil
}
