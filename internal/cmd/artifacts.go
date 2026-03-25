package cmd

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/types"
)

func newArtifactsCmd() *cobra.Command {
	artifactsCmd := &cobra.Command{
		Use:   "artifacts",
		Short: "拉取并管理作业生成的工件",
	}

	listCmd := &cobra.Command{
		Use:   "list <job-id>",
		Short: "列出作业生成的工件",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var artifacts types.ArtifactListResponse
			if err := currentClient().GetJSON(context.Background(), "/jobs/"+args[0]+"/artifacts", nil, &artifacts); err != nil {
				return err
			}
			return writeOutput(cmd, artifacts)
		},
	}

	getCmd := &cobra.Command{
		Use:   "get <artifact-id>",
		Short: "获取工件元数据",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var artifact types.Artifact
			if err := currentClient().GetJSON(context.Background(), "/artifacts/"+args[0], nil, &artifact); err != nil {
				return err
			}
			return writeOutput(cmd, artifact)
		},
	}

	pullCmd := &cobra.Command{
		Use:   "pull <artifact-id>",
		Short: "将工件下载到磁盘",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var meta types.Artifact
			if err := currentClient().GetJSON(context.Background(), "/artifacts/"+args[0], nil, &meta); err != nil {
				return err
			}

			res, err := currentClient().Stream(context.Background(), "GET", "/artifacts/"+args[0]+"/content", apiOpts("", "application/octet-stream"))
			if err != nil {
				return err
			}
			defer res.Body.Close()

			if flagBool(cmd, "stdout") {
				_, err = io.Copy(cmd.OutOrStdout(), res.Body)
				return err
			}

			outPath := flagString(cmd, "out")
			if outPath == "" {
				filename := meta.Filename
				if filename == "" {
					filename = "artifact-" + args[0]
				}
				outPath = filepath.Join(".", filename)
			}

			if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
				return output.NewError("CLI_ERROR", "创建输出目录失败", map[string]any{"details": err.Error()})
			}

			file, err := os.Create(outPath)
			if err != nil {
				return output.NewError("CLI_ERROR", "创建输出文件失败", map[string]any{"details": err.Error()})
			}
			defer file.Close()

			n, err := io.Copy(file, res.Body)
			if err != nil {
				return output.NewError("NETWORK_ERROR", "写入工件失败", map[string]any{"details": err.Error()})
			}

			return writeOutput(cmd, map[string]any{
				"artifact_id":  args[0],
				"saved_to":     outPath,
				"bytes":        n,
				"content_type": meta.ContentType,
			})
		},
	}
	pullCmd.Flags().StringP("out", "o", "", "输出文件路径（默认：./<artifact-name>）")
	pullCmd.Flags().Bool("stdout", false, "将内容写入 stdout 而不是文件")

	pullAllCmd := &cobra.Command{
		Use:   "pull-all <job-id>",
		Short: "将作业中的所有工件下载到一个目录中",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var list types.ArtifactListResponse
			if err := currentClient().GetJSON(context.Background(), "/jobs/"+args[0]+"/artifacts", nil, &list); err != nil {
				return err
			}

			if len(list.Items) == 0 {
				return writeOutput(cmd, map[string]any{
					"job_id":               args[0],
					"artifacts_downloaded": 0,
				})
			}

			dir := flagString(cmd, "dir")
			if dir == "" {
				dir = filepath.Join(".", args[0])
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return output.NewError("CLI_ERROR", "创建输出目录失败", map[string]any{"details": err.Error()})
			}

			files := make([]map[string]any, 0, len(list.Items))
			for _, item := range list.Items {
				artifactID := item.ID
				filename := item.Filename
				if filename == "" {
					filename = "artifact-" + artifactID
				}

				res, err := currentClient().Stream(context.Background(), "GET", "/artifacts/"+artifactID+"/content", apiOpts("", "application/octet-stream"))
				if err != nil {
					return err
				}

				outPath := filepath.Join(dir, filename)
				file, err := os.Create(outPath)
				if err != nil {
					res.Body.Close()
					return output.NewError("CLI_ERROR", "创建输出文件失败", map[string]any{"details": err.Error()})
				}

				n, copyErr := io.Copy(file, res.Body)
				file.Close()
				res.Body.Close()
				if copyErr != nil {
					return output.NewError("NETWORK_ERROR", "写入工件失败", map[string]any{"details": copyErr.Error()})
				}

				files = append(files, map[string]any{
					"artifact_id": artifactID,
					"saved_to":    outPath,
					"bytes":       n,
				})
			}

			return writeOutput(cmd, map[string]any{
				"job_id":               args[0],
				"artifacts_downloaded": len(files),
				"files":                files,
			})
		},
	}
	pullAllCmd.Flags().StringP("dir", "d", "", "输出目录（默认：./<job-id>）")

	artifactsCmd.AddCommand(listCmd, getCmd, pullCmd, pullAllCmd)
	return artifactsCmd
}
