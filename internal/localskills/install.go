package localskills

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/wtgoku-create/popiartcli/internal/agentpaths"
	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/output"
)

type PullResult struct {
	Slug        string `json:"slug"`
	Version     string `json:"version,omitempty"`
	ArchivePath string `json:"archive_path"`
	SourceURL   string `json:"source_url,omitempty"`
}

type InstallOptions struct {
	Force         bool
	Agent         string
	AgentSkillDir string
}

type InstallResult struct {
	Slug           string `json:"slug"`
	Version        string `json:"version,omitempty"`
	InstalledDir   string `json:"installed_dir"`
	ManifestPath   string `json:"manifest_path"`
	SkillDocPath   string `json:"skill_doc_path"`
	AgentSkillPath string `json:"agent_skill_path,omitempty"`
}

var DownloadHTTPClient = &http.Client{
	Timeout: 60 * time.Second,
}

func DownloadArchive(ctx context.Context, archiveURL string) (PullResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, archiveURL, nil)
	if err != nil {
		return PullResult{}, output.NewError("BAD_REQUEST", "创建下载请求失败", map[string]any{
			"url":     archiveURL,
			"details": err.Error(),
		})
	}
	req.Header.Set("User-Agent", "popiart-cli/0.2.0")

	res, err := DownloadHTTPClient.Do(req)
	if err != nil {
		return PullResult{}, output.NewError("NETWORK_ERROR", fmt.Sprintf("下载 skill 失败: %v", err), map[string]any{
			"url": archiveURL,
		})
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return PullResult{}, output.NewError("NETWORK_ERROR", "下载 skill 失败", map[string]any{
			"url":    archiveURL,
			"status": res.Status,
		})
	}

	tempFile, err := os.CreateTemp("", "popiart-skill-*.zip")
	if err != nil {
		return PullResult{}, output.NewError("CLI_ERROR", "创建临时归档失败", map[string]any{
			"details": err.Error(),
		})
	}
	defer os.Remove(tempFile.Name())

	if _, err := io.Copy(tempFile, res.Body); err != nil {
		tempFile.Close()
		return PullResult{}, output.NewError("NETWORK_ERROR", "写入临时归档失败", map[string]any{
			"details": err.Error(),
		})
	}
	if err := tempFile.Close(); err != nil {
		return PullResult{}, output.NewError("CLI_ERROR", "关闭临时归档失败", map[string]any{
			"details": err.Error(),
		})
	}

	manifest, err := inspectArchive(tempFile.Name())
	if err != nil {
		return PullResult{}, err
	}

	archiveName := archiveFilename(archiveURL, manifest)
	targetDir := filepath.Join(config.SkillDownloadsDir(), manifest.Slug)
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return PullResult{}, output.NewError("CLI_ERROR", "创建 skill 下载目录失败", map[string]any{
			"path":    targetDir,
			"details": err.Error(),
		})
	}

	targetPath := filepath.Join(targetDir, archiveName)
	if err := copyFile(tempFile.Name(), targetPath); err != nil {
		return PullResult{}, err
	}

	return PullResult{
		Slug:        manifest.Slug,
		Version:     manifest.Version,
		ArchivePath: targetPath,
		SourceURL:   archiveURL,
	}, nil
}

func InstallArchive(archivePath string, opts InstallOptions) (InstallResult, error) {
	archivePath = filepath.Clean(strings.TrimSpace(archivePath))
	if archivePath == "" {
		return InstallResult{}, output.NewError("VALIDATION_ERROR", "缺少 skill 归档路径", nil)
	}

	manifest, rootDir, cleanup, err := extractArchiveToTemp(archivePath)
	if err != nil {
		return InstallResult{}, err
	}
	defer cleanup()

	targetDir := filepath.Join(config.InstalledSkillsDir(), manifest.Slug)
	if err := os.MkdirAll(filepath.Dir(targetDir), 0o700); err != nil {
		return InstallResult{}, output.NewError("CLI_ERROR", "创建本地 skill 安装目录失败", map[string]any{
			"path":    targetDir,
			"details": err.Error(),
		})
	}

	if _, err := os.Stat(targetDir); err == nil {
		if !opts.Force {
			return InstallResult{}, output.NewError("CONFLICT", "本地 skill 已存在；如需覆盖请传 --force", map[string]any{
				"slug":        manifest.Slug,
				"install_dir": targetDir,
			})
		}
		if err := os.RemoveAll(targetDir); err != nil {
			return InstallResult{}, output.NewError("CLI_ERROR", "删除旧的本地 skill 失败", map[string]any{
				"path":    targetDir,
				"details": err.Error(),
			})
		}
	}

	if err := copyDir(rootDir, targetDir); err != nil {
		return InstallResult{}, err
	}

	installedSkill, err := LoadInstalledSkill(targetDir)
	if err != nil {
		return InstallResult{}, err
	}

	result := InstallResult{
		Slug:         installedSkill.Manifest.Slug,
		Version:      installedSkill.Manifest.Version,
		InstalledDir: targetDir,
		ManifestPath: installedSkill.ManifestPath,
		SkillDocPath: installedSkill.SkillDocPath,
	}

	if opts.Agent != "" || opts.AgentSkillDir != "" {
		agentPath, err := LinkToAgent(installedSkill, opts.Agent, opts.AgentSkillDir)
		if err != nil {
			return InstallResult{}, err
		}
		result.AgentSkillPath = agentPath
	}

	return result, nil
}

func LatestDownloadedArchive(slug string) (string, error) {
	dir := filepath.Join(config.SkillDownloadsDir(), strings.TrimSpace(slug))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", output.NewError("NOT_FOUND", "本地没有可安装的已下载归档", map[string]any{
			"slug": slug,
			"path": dir,
		})
	}

	type candidate struct {
		path    string
		modTime time.Time
	}
	items := make([]candidate, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".zip") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		items = append(items, candidate{
			path:    filepath.Join(dir, entry.Name()),
			modTime: info.ModTime(),
		})
	}
	if len(items) == 0 {
		return "", output.NewError("NOT_FOUND", "本地没有可安装的已下载归档", map[string]any{
			"slug": slug,
			"path": dir,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].modTime.After(items[j].modTime)
	})
	return items[0].path, nil
}

func LinkToAgent(skill InstalledSkill, agent, explicitDir string) (string, error) {
	targetDir, err := resolveAgentSkillDir(agent, explicitDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return "", output.NewError("CLI_ERROR", "创建 agent skills 目录失败", map[string]any{
			"path":    targetDir,
			"details": err.Error(),
		})
	}

	linkPath := filepath.Join(targetDir, skill.Manifest.Slug)
	if err := os.RemoveAll(linkPath); err != nil {
		return "", output.NewError("CLI_ERROR", "清理旧的 agent skill 失败", map[string]any{
			"path":    linkPath,
			"details": err.Error(),
		})
	}

	if err := os.Symlink(skill.RootDir, linkPath); err == nil {
		return linkPath, nil
	}
	if err := copyDir(skill.RootDir, linkPath); err != nil {
		return "", err
	}
	return linkPath, nil
}

func resolveAgentSkillDir(agent, explicitDir string) (string, error) {
	if strings.TrimSpace(explicitDir) != "" {
		return filepath.Clean(explicitDir), nil
	}

	if strings.TrimSpace(agent) == "" {
		return "", output.NewError("VALIDATION_ERROR", "需要显式指定 --agent 或 --agent-skill-dir", nil)
	}

	paths, err := agentpaths.Resolve(agent)
	if err != nil {
		return "", err
	}
	return paths.SkillDir, nil
}

func archiveFilename(sourceURL string, manifest Manifest) string {
	if manifest.Archive != "" {
		return manifest.Archive
	}
	if parsed, err := url.Parse(sourceURL); err == nil {
		name := path.Base(parsed.Path)
		if strings.TrimSpace(name) != "" && name != "." && name != "/" {
			return name
		}
	}
	return manifest.Slug + ".zip"
}

func inspectArchive(archivePath string) (Manifest, error) {
	manifest, _, cleanup, err := extractArchiveToTemp(archivePath)
	if cleanup != nil {
		defer cleanup()
	}
	return manifest, err
}

func extractArchiveToTemp(archivePath string) (Manifest, string, func(), error) {
	tempDir, err := os.MkdirTemp("", "popiart-skill-install-*")
	if err != nil {
		return Manifest{}, "", nil, output.NewError("CLI_ERROR", "创建临时解压目录失败", map[string]any{
			"details": err.Error(),
		})
	}

	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	if err := unzip(archivePath, tempDir); err != nil {
		cleanup()
		return Manifest{}, "", nil, err
	}

	rootDir, err := determinePackageRoot(tempDir)
	if err != nil {
		cleanup()
		return Manifest{}, "", nil, err
	}

	skill, err := LoadInstalledSkill(rootDir)
	if err != nil {
		cleanup()
		return Manifest{}, "", nil, err
	}

	if skill.Manifest.PackageLayout == "rooted" && skill.Manifest.PackageRoot != "" {
		expectedRoot := filepath.Join(tempDir, skill.Manifest.PackageRoot)
		if filepath.Clean(rootDir) != filepath.Clean(expectedRoot) {
			cleanup()
			return Manifest{}, "", nil, output.NewError("LOCAL_SKILL_INVALID", "skill 包的顶层目录与 package_root 不匹配", map[string]any{
				"package_root": skill.Manifest.PackageRoot,
			})
		}
	}

	return skill.Manifest, rootDir, cleanup, nil
}

func determinePackageRoot(tempDir string) (string, error) {
	if fileExists(filepath.Join(tempDir, "SKILL.md")) || fileExists(filepath.Join(tempDir, "popiart-skill.yaml")) || fileExists(filepath.Join(tempDir, "popiart-skill.yml")) || fileExists(filepath.Join(tempDir, "popiart-skill.json")) {
		return tempDir, nil
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return "", output.NewError("CLI_ERROR", "读取解压目录失败", map[string]any{
			"path":    tempDir,
			"details": err.Error(),
		})
	}

	var dirs []string
	for _, entry := range entries {
		if entry.Name() == "__MACOSX" {
			continue
		}
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(tempDir, entry.Name()))
		}
	}
	if len(dirs) == 1 {
		return dirs[0], nil
	}

	return "", output.NewError("LOCAL_SKILL_INVALID", "无法识别 skill 包根目录", map[string]any{
		"archive_layout": "expected rooted package or manifest files at archive root",
	})
}

func unzip(archivePath, targetDir string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return output.NewError("LOCAL_SKILL_INVALID", "打开 skill 压缩包失败", map[string]any{
			"path":    archivePath,
			"details": err.Error(),
		})
	}
	defer reader.Close()

	for _, file := range reader.File {
		cleanName := filepath.Clean(file.Name)
		if cleanName == "." {
			continue
		}
		if strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
			return output.NewError("LOCAL_SKILL_INVALID", "skill 压缩包包含不安全路径", map[string]any{
				"name": file.Name,
			})
		}

		targetPath := filepath.Join(targetDir, cleanName)
		if !strings.HasPrefix(targetPath, filepath.Clean(targetDir)+string(os.PathSeparator)) && filepath.Clean(targetPath) != filepath.Clean(targetDir) {
			return output.NewError("LOCAL_SKILL_INVALID", "skill 压缩包目标路径越界", map[string]any{
				"name": file.Name,
			})
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return output.NewError("CLI_ERROR", "创建解压目录失败", map[string]any{
					"path":    targetPath,
					"details": err.Error(),
				})
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return output.NewError("CLI_ERROR", "创建解压目录失败", map[string]any{
				"path":    filepath.Dir(targetPath),
				"details": err.Error(),
			})
		}

		in, err := file.Open()
		if err != nil {
			return output.NewError("LOCAL_SKILL_INVALID", "读取压缩包内容失败", map[string]any{
				"name":    file.Name,
				"details": err.Error(),
			})
		}

		out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fileMode(file.Mode()))
		if err != nil {
			in.Close()
			return output.NewError("CLI_ERROR", "写入解压文件失败", map[string]any{
				"path":    targetPath,
				"details": err.Error(),
			})
		}

		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		in.Close()
		if copyErr != nil {
			return output.NewError("CLI_ERROR", "解压 skill 文件失败", map[string]any{
				"path":    targetPath,
				"details": copyErr.Error(),
			})
		}
		if closeErr != nil {
			return output.NewError("CLI_ERROR", "关闭解压文件失败", map[string]any{
				"path":    targetPath,
				"details": closeErr.Error(),
			})
		}
	}

	return nil
}

func fileMode(mode os.FileMode) os.FileMode {
	if runtime.GOOS == "windows" {
		return 0o644
	}
	if mode == 0 {
		return 0o644
	}
	return mode
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return output.NewError("CLI_ERROR", "打开源文件失败", map[string]any{
			"path":    src,
			"details": err.Error(),
		})
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return output.NewError("CLI_ERROR", "创建目标目录失败", map[string]any{
			"path":    filepath.Dir(dst),
			"details": err.Error(),
		})
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return output.NewError("CLI_ERROR", "创建目标文件失败", map[string]any{
			"path":    dst,
			"details": err.Error(),
		})
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return output.NewError("CLI_ERROR", "复制文件失败", map[string]any{
			"src":     src,
			"dst":     dst,
			"details": err.Error(),
		})
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(current string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, current)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		return copyFile(current, targetPath)
	})
}
