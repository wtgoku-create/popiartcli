package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
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

const defaultUpdateRepo = "wtgoku-create/popiartcli"

var updateHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

var updateLatestReleaseURL = func(repo string) string {
	return fmt.Sprintf("https://github.com/%s/releases/latest", repo)
}

var resolveUpdateTagFunc = resolveTargetReleaseTag
var resolveExecutablePathsFunc = resolveExecutablePaths
var selfUpdateRunner = runSelfUpdate

func newUpdateCmd() *cobra.Command {
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "从 GitHub Releases 下载并安装最新版本",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			repo := strings.TrimSpace(flagString(cmd, "repo"))
			if repo == "" {
				repo = strings.TrimSpace(os.Getenv("POPIART_REPO"))
			}
			if repo == "" {
				repo = defaultUpdateRepo
			}

			targetTag, err := resolveUpdateTagFunc(ctx, repo, flagString(cmd, "version"))
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
					"path":             executablePath,
					"config_unchanged": true,
				})
			}

			updateResult, err := selfUpdateRunner(ctx, updateRunOptions{
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
				"path":                 updateResult.ExecutablePath,
				"restart_required":     updateResult.RestartRequired,
				"config_unchanged":     true,
				"installer_script_url": updateResult.ScriptURL,
			})
		},
	}

	updateCmd.Flags().String("version", "", "更新到指定版本，例如 v0.1.0；默认使用最新 release")
	updateCmd.Flags().String("repo", "", "覆盖 GitHub 仓库，格式 owner/name")
	_ = updateCmd.Flags().MarkHidden("repo")

	return updateCmd
}

type updateRunOptions struct {
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
	scriptURL := rawGitHubContentURL(opts.Repo, opts.Tag, "install.sh")
	scriptPath, cleanup, err := downloadTemporaryScript(ctx, scriptURL, "", "popiart-install-*.sh", 0o700)
	if err != nil {
		return updateRunResult{}, err
	}
	defer cleanup()

	command := exec.CommandContext(ctx, "sh", scriptPath, "--cli-only", "--version", opts.Tag)
	command.Env = append(os.Environ(),
		"BINDIR="+filepath.Dir(opts.ExecutablePath),
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

	scriptURL := rawGitHubContentURL(opts.Repo, opts.Tag, "install.ps1")
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
  [string]$Repo,
  [string]$Version
)

$ErrorActionPreference = "Stop"

while (Get-Process -Id $CurrentPid -ErrorAction SilentlyContinue) {
  Start-Sleep -Milliseconds 500
}

$env:POPIART_REPO = $Repo

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

func resolveTargetReleaseTag(ctx context.Context, repo, requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested != "" {
		return normalizeReleaseTag(requested), nil
	}

	url := updateLatestReleaseURL(repo)
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
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", output.NewError("NETWORK_ERROR", "获取最新 release 失败", map[string]any{
			"repo":   repo,
			"status": resp.Status,
		})
	}

	tag := path.Base(resp.Request.URL.Path)
	if tag == "" || tag == "latest" || tag == "." || tag == "/" {
		return "", output.NewError("NETWORK_ERROR", "解析最新 release 版本失败", map[string]any{
			"repo": repo,
			"url":  resp.Request.URL.String(),
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

func rawGitHubContentURL(repo, tag, name string) string {
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", repo, tag, name)
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
