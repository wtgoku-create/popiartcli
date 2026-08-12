package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/popiart"
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
		Use:   "list [job-id]",
		Short: "列出作业生成的工件",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return output.NewError("VALIDATION_ERROR", "缺少作业 ID", map[string]any{
					"argument": "job-id",
					"hint":     "请使用: popiart artifacts list <task-id>。当前模式下单个 artifact 对象不再作为主查询入口。",
				})
			}

			urls, err := popiart.GetTaskDownloadURLs(context.Background(), currentClient(), args[0])
			if err != nil {
				return err
			}
			items := make([]map[string]any, 0, len(urls))
			for index, item := range urls {
				items = append(items, map[string]any{
					"id":       fmt.Sprintf("%s#%d", args[0], index+1),
					"url":      item,
					"filename": filenameFromDownloadURL(item, index+1),
				})
			}
			return writeOutput(cmd, map[string]any{
				"job_id":  args[0],
				"task_id": args[0],
				"items":   items,
				"total":   len(items),
			})
		},
	}

	getCmd := &cobra.Command{
		Use:   "get <artifact-id>",
		Short: "获取工件元数据",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			media, err := popiart.FetchMediaDetail(context.Background(), currentClient(), args[0])
			if err != nil {
				return err
			}
			result := map[string]any{
				"id":           media.ID,
				"artifact_id":  media.ID,
				"media_id":     media.ID,
				"project_id":   media.ProjectID,
				"filename":     media.Filename,
				"content_type": media.ContentType,
				"size_bytes":   media.SizeBytes,
				"created_at":   media.CreatedAt,
				"visibility":   media.Visibility,
				"sha256":       media.SHA256,
			}
			addStableURLFields(result, media.URL)
			return writeOutput(cmd, result)
		},
	}

	pullCmd := &cobra.Command{
		Use:   "pull <artifact-id>",
		Short: "将工件下载到磁盘",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return output.NewError("UNSUPPORTED_IN_POPI_ART_MODE", "当前模式不支持按 artifact_id 下载单个任务结果", map[string]any{
				"hint": "请改用 `popiart artifacts pull-all <task-id>` 下载任务结果中的全部文件",
			})
		},
	}
	pullCmd.Flags().StringP("out", "o", "", "输出文件路径（默认：./<artifact-name>）")
	pullCmd.Flags().Bool("stdout", false, "将内容写入 stdout 而不是文件")

	pullAllCmd := &cobra.Command{
		Use:   "pull-all <task-id>",
		Short: "将任务中的所有结果文件下载到一个目录中",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			urls, err := popiart.GetTaskDownloadURLs(context.Background(), currentClient(), args[0])
			if err != nil {
				return err
			}

			if len(urls) == 0 {
				return writeOutput(cmd, map[string]any{
					"job_id":               args[0],
					"task_id":              args[0],
					"artifacts_downloaded": 0,
				})
			}

			dir := flagString(cmd, "dir")
			if dir == "" {
				dir = filepath.Join(".", args[0])
			}
			if dryRunMode(cmd) {
				return writeDryRunPreview(cmd, "artifacts.pull-all", map[string]any{
					"job_id":               args[0],
					"task_id":              args[0],
					"artifacts_downloaded": len(urls),
					"files":                plannedDownloadFiles(urls, dir, true),
				})
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return output.NewError("CLI_ERROR", "创建输出目录失败", map[string]any{"details": err.Error()})
			}

			files, err := downloadResultURLs(context.Background(), urls, dir, true)
			if err != nil {
				return err
			}

			return writeOutput(cmd, map[string]any{
				"job_id":               args[0],
				"task_id":              args[0],
				"artifacts_downloaded": len(files),
				"files":                files,
			})
		},
	}
	pullAllCmd.Flags().StringP("dir", "d", "", "输出目录（默认：./<task-id>）")

	uploadCmd := &cobra.Command{
		Use:   "upload <path>",
		Short: "上传本地文件并创建一个带稳定 URL 的可复用工件",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunMode(cmd) {
				return writeDryRunPreview(cmd, "artifacts.upload", map[string]any{
					"path": args[0],
					"request": map[string]any{
						"method": "POST",
						"path":   "/api_client/media/upload",
						"body": map[string]any{
							"path":          args[0],
							"filename":      flagString(cmd, "filename"),
							"content_type":  flagString(cmd, "content-type"),
							"role":          flagString(cmd, "role"),
							"metadata_json": flagString(cmd, "metadata-json"),
							"visibility":    flagString(cmd, "visibility"),
						},
					},
				})
			}
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

// filenameFromDownloadURL 为 task 下载地址生成稳定文件名。
func filenameFromDownloadURL(raw string, index int) string {
	parsed, err := neturl.Parse(strings.TrimSpace(raw))
	if err == nil {
		base := filepath.Base(parsed.Path)
		base = strings.TrimSpace(base)
		if base != "" && base != "." && base != "/" {
			return base
		}
	}
	return fmt.Sprintf("artifact-%d", index)
}

func plannedDownloadFiles(urls []string, dir string, includeURL bool) []map[string]any {
	files := make([]map[string]any, 0, len(urls))
	for index, item := range urls {
		fileResult := map[string]any{
			"would_save_to": filepath.Join(dir, filenameFromDownloadURL(item, index+1)),
		}
		if includeURL {
			fileResult["url"] = item
		}
		files = append(files, fileResult)
	}
	return files
}

func downloadResultURLs(ctx context.Context, urls []string, dir string, includeURL bool) ([]map[string]any, error) {
	files := make([]map[string]any, 0, len(urls))
	for index, item := range urls {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, item, nil)
		if err != nil {
			return nil, output.NewError("BAD_REQUEST", "无效的任务结果下载地址", map[string]any{
				"url":     item,
				"details": err.Error(),
			})
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, output.NewError("NETWORK_ERROR", "下载任务结果失败", map[string]any{
				"url":     item,
				"details": err.Error(),
			})
		}
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			res.Body.Close()
			return nil, output.NewError("NETWORK_ERROR", "下载任务结果失败", map[string]any{
				"url":    item,
				"status": res.StatusCode,
			})
		}

		outPath := filepath.Join(dir, filenameFromDownloadURL(item, index+1))
		file, err := os.Create(outPath)
		if err != nil {
			res.Body.Close()
			return nil, output.NewError("CLI_ERROR", "创建输出文件失败", map[string]any{"details": err.Error()})
		}

		n, copyErr := io.Copy(file, res.Body)
		file.Close()
		res.Body.Close()
		if copyErr != nil {
			return nil, output.NewError("NETWORK_ERROR", "写入工件失败", map[string]any{"details": copyErr.Error()})
		}

		fileResult := map[string]any{
			"saved_to": outPath,
			"bytes":    n,
		}
		if includeURL {
			fileResult["url"] = item
		}
		files = append(files, fileResult)
	}
	return files, nil
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
		"metadata_json": opts.MetadataJSON,
		"project_id":    projectID,
		"visibility":    opts.Visibility,
	}
	if opts.Role != "" {
		fields["role"] = opts.Role
	}

	// 复用主站 media 上传归一化逻辑，兼容 numeric id / name / createTime 等真实返回结构。
	media, err := popiart.UploadMedia(ctx, currentClient(), path, popiart.UploadOptions{
		Filename:    filename,
		ContentType: contentType,
		Fields:      fields,
		MaxRetries:  3,
	})
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"artifact_id":   media.ID,
		"media_id":      media.ID,
		"filename":      firstNonEmptyString(media.Filename, filename),
		"content_type":  firstNonEmptyString(media.ContentType, contentType),
		"size_bytes":    media.SizeBytes,
		"created_at":    media.CreatedAt,
		"uploaded_from": path,
	}
	addStableURLFields(result, media.URL)
	if media.Visibility != "" {
		result["visibility"] = media.Visibility
	}
	if media.SHA256 != "" {
		result["sha256"] = media.SHA256
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
