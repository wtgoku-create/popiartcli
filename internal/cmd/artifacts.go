package cmd

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/api"
	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/types"
)

type artifactUploadOptions struct {
	Filename     string
	ContentType  string
	Role         string
	MetadataJSON string
	ProjectID    string
	Visibility   string
}

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

	uploadCmd := &cobra.Command{
		Use:   "upload <path>",
		Short: "上传本地文件并创建一个带稳定 URL 的可复用工件",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := uploadArtifact(context.Background(), args[0], artifactUploadOptions{
				Filename:     flagString(cmd, "filename"),
				ContentType:  flagString(cmd, "content-type"),
				Role:         flagString(cmd, "role"),
				MetadataJSON: flagString(cmd, "metadata-json"),
				Visibility:   flagString(cmd, "visibility"),
			})
			if err != nil {
				return err
			}
			return writeOutput(cmd, result)
		},
	}
	uploadCmd.Flags().String("filename", "", "上传时覆盖文件名（默认：使用本地文件名）")
	uploadCmd.Flags().String("content-type", "", "上传内容类型（默认：按扩展名或文件头推断）")
	uploadCmd.Flags().String("role", "", "上传工件角色，例如 source | mask | reference")
	uploadCmd.Flags().String("metadata-json", "", "附带的 JSON 元数据字符串")
	uploadCmd.Flags().String("visibility", "", "工件稳定 URL 的可见性，例如 private | unlisted | public")

	artifactsCmd.AddCommand(listCmd, getCmd, pullCmd, pullAllCmd, uploadCmd)
	return artifactsCmd
}

func uploadArtifact(ctx context.Context, path string, opts artifactUploadOptions) (map[string]any, error) {
	if err := validateUploadInput(path, opts.MetadataJSON); err != nil {
		return nil, err
	}

	filename := resolveUploadFilename(path, opts.Filename)
	contentType := resolveUploadContentType(path, opts.ContentType)
	projectID := resolveUploadProjectID(opts.ProjectID)

	fields := map[string]string{
		"filename":      filename,
		"content_type":  contentType,
		"role":          opts.Role,
		"metadata_json": opts.MetadataJSON,
		"project_id":    projectID,
		"visibility":    opts.Visibility,
	}

	var artifact types.Artifact
	if err := currentClient().UploadFile(ctx, "/artifacts/upload", path, api.UploadFileOptions{
		Filename:    filename,
		ContentType: contentType,
		Fields:      fields,
	}, &artifact); err != nil {
		return nil, err
	}

	result := map[string]any{
		"artifact_id":   artifact.ID,
		"filename":      artifact.Filename,
		"content_type":  artifact.ContentType,
		"size_bytes":    artifact.SizeBytes,
		"created_at":    artifact.CreatedAt,
		"expires_at":    artifact.ExpiresAt,
		"uploaded_from": path,
	}
	if artifact.JobID != "" {
		result["job_id"] = artifact.JobID
	}
	if artifact.MediaID != "" {
		result["media_id"] = artifact.MediaID
	}
	if artifact.URL != "" {
		result["url"] = artifact.URL
	}
	if artifact.Visibility != "" {
		result["visibility"] = artifact.Visibility
	}
	if artifact.SHA256 != "" {
		result["sha256"] = artifact.SHA256
	}
	if artifact.StorageStatus != "" {
		result["storage_status"] = artifact.StorageStatus
	}
	if opts.Role != "" {
		result["role"] = opts.Role
	}
	if projectID != "" {
		result["project_id"] = projectID
	}
	return result, nil
}

func validateUploadInput(path, metadataJSON string) error {
	info, err := os.Stat(path)
	if err != nil {
		return output.NewError("CLI_ERROR", "读取上传文件失败", map[string]any{
			"path":    path,
			"details": err.Error(),
		})
	}
	if info.IsDir() {
		return output.NewError("VALIDATION_ERROR", "上传路径必须是文件，不能是目录", map[string]any{
			"path": path,
		})
	}
	if metadataJSON != "" && !json.Valid([]byte(metadataJSON)) {
		return output.NewError("INPUT_PARSE_ERROR", "metadata-json 不是合法 JSON", map[string]any{
			"metadata_json": metadataJSON,
		})
	}
	return nil
}

func resolveUploadFilename(path, filename string) string {
	if filename != "" {
		return filename
	}
	return filepath.Base(path)
}

func resolveUploadContentType(path, contentType string) string {
	if contentType != "" {
		return contentType
	}
	return detectUploadContentType(path)
}

func resolveUploadProjectID(projectID string) string {
	if projectID != "" {
		return projectID
	}
	return config.Load().Project
}

func detectUploadContentType(path string) string {
	if contentType := mime.TypeByExtension(filepath.Ext(path)); contentType != "" {
		return contentType
	}

	file, err := os.Open(path)
	if err != nil {
		return "application/octet-stream"
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "application/octet-stream"
	}
	return http.DetectContentType(buffer[:n])
}
