package cmd

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/localskills"
	"github.com/wtgoku-create/popiartcli/internal/output"
)

func newSkillsPullCmd() *cobra.Command {
	var sourceURL string

	pullCmd := &cobra.Command{
		Use:   "pull <skill-id-or-url>",
		Short: "下载 skill 归档到本地缓存目录",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			archiveURL, err := resolveSkillArchiveURL(ctx, args[0], sourceURL)
			if err != nil {
				return err
			}

			result, err := localskills.DownloadArchive(ctx, archiveURL)
			if err != nil {
				return err
			}
			return writeOutput(cmd, result)
		},
	}
	pullCmd.Flags().StringVar(&sourceURL, "url", "", "显式指定 skill 压缩包 URL")
	return pullCmd
}

func newSkillsInstallCmd() *cobra.Command {
	var sourceURL string
	var force bool
	var agent string
	var agentSkillDir string

	installCmd := &cobra.Command{
		Use:   "install <skill-id|archive-path|url>",
		Short: "安装一个本地 skill，并可选同步到 agent skills 目录",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			archivePath, err := resolveInstallArchive(ctx, args[0], sourceURL)
			if err != nil {
				return err
			}

			result, err := localskills.InstallArchive(archivePath, localskills.InstallOptions{
				Force:         force,
				Agent:         agent,
				AgentSkillDir: agentSkillDir,
			})
			if err != nil {
				return err
			}
			return writeOutput(cmd, result)
		},
	}
	installCmd.Flags().StringVar(&sourceURL, "url", "", "显式指定 skill 压缩包 URL")
	installCmd.Flags().BoolVar(&force, "force", false, "覆盖已有的本地安装")
	installCmd.Flags().StringVar(&agent, "agent", "", "将 skill 同步到指定 agent 的 skills 目录，例如 codex")
	installCmd.Flags().StringVar(&agentSkillDir, "agent-skill-dir", "", "覆盖 agent 的 skills 目录路径")
	return installCmd
}

func newSkillsUseLocalCmd() *cobra.Command {
	var agent string
	var agentSkillDir string

	useLocalCmd := &cobra.Command{
		Use:   "use-local <skill-id>",
		Short: "将已安装的本地 skill 标记为优先使用",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			skill, ok, err := localskills.FindInstalled(args[0])
			if err != nil {
				return err
			}
			if !ok {
				return output.NewError("NOT_FOUND", "未找到已安装的本地 skill", map[string]any{
					"skill_id": args[0],
				})
			}

			if err := localskills.Activate(skill.Manifest.Slug); err != nil {
				return err
			}

			result := map[string]any{
				"skill_id":       skill.Manifest.Slug,
				"local_active":   true,
				"install_dir":    skill.RootDir,
				"execution_mode": skill.Manifest.Execution.Mode,
			}

			if agent != "" || agentSkillDir != "" {
				linkPath, err := localskills.LinkToAgent(skill, agent, agentSkillDir)
				if err != nil {
					return err
				}
				result["agent_skill_path"] = linkPath
			}

			return writeOutput(cmd, result)
		},
	}
	useLocalCmd.Flags().StringVar(&agent, "agent", "", "将已安装 skill 同步到指定 agent 的 skills 目录")
	useLocalCmd.Flags().StringVar(&agentSkillDir, "agent-skill-dir", "", "覆盖 agent 的 skills 目录路径")
	return useLocalCmd
}

func resolveInstallArchive(ctx context.Context, ref, sourceURL string) (string, error) {
	ref = strings.TrimSpace(ref)
	if isHTTPURL(ref) || strings.TrimSpace(sourceURL) != "" {
		archiveURL, err := resolveSkillArchiveURL(ctx, ref, sourceURL)
		if err != nil {
			return "", err
		}
		result, err := localskills.DownloadArchive(ctx, archiveURL)
		if err != nil {
			return "", err
		}
		return result.ArchivePath, nil
	}

	if path, ok := existingArchivePath(ref); ok {
		return path, nil
	}

	path, err := localskills.LatestDownloadedArchive(ref)
	if err == nil {
		return path, nil
	}

	archiveURL, urlErr := resolveSkillArchiveURL(ctx, ref, sourceURL)
	if urlErr != nil {
		return "", output.NewError("NOT_FOUND", "本地没有可安装的下载包，且远端未暴露归档地址", map[string]any{
			"skill_id":            ref,
			"local_lookup_error":  err.Error(),
			"remote_lookup_error": urlErr.Error(),
		})
	}
	result, pullErr := localskills.DownloadArchive(ctx, archiveURL)
	if pullErr != nil {
		return "", pullErr
	}
	return result.ArchivePath, nil
}

func resolveSkillArchiveURL(ctx context.Context, ref, sourceURL string) (string, error) {
	if strings.TrimSpace(sourceURL) != "" {
		if !isHTTPURL(sourceURL) {
			return "", output.NewError("VALIDATION_ERROR", "--url 必须是 http(s) 地址", map[string]any{
				"url": sourceURL,
			})
		}
		return sourceURL, nil
	}

	if isHTTPURL(ref) {
		return ref, nil
	}

	packageURL, err := remoteSkillArchiveURL(ctx, ref)
	if err != nil {
		return "", err
	}
	return packageURL, nil
}

func remoteSkillArchiveURL(ctx context.Context, skillID string) (string, error) {
	var payload map[string]any
	if err := currentClient().GetJSON(ctx, "/skills/"+skillID, nil, &payload); err != nil {
		return "", err
	}

	candidates := []string{"package_url", "archive_url", "download_url"}
	for _, key := range candidates {
		value, _ := payload[key].(string)
		if isHTTPURL(value) {
			return value, nil
		}
	}

	return "", output.NewError("NOT_FOUND", "远端 skill 未暴露可下载归档；请改用 --url", map[string]any{
		"skill_id": skillID,
	})
}

func existingArchivePath(ref string) (string, bool) {
	if strings.TrimSpace(ref) == "" {
		return "", false
	}
	info, err := os.Stat(ref)
	if err != nil || info.IsDir() {
		return "", false
	}
	if strings.HasSuffix(strings.ToLower(ref), ".zip") {
		absolute, err := filepath.Abs(ref)
		if err != nil {
			return ref, true
		}
		return absolute, true
	}
	return "", false
}

func isHTTPURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}
