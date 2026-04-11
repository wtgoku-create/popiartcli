package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/api"
	"github.com/wtgoku-create/popiartcli/internal/types"
)

type mediaUploadOptions struct {
	Filename     string
	ContentType  string
	MetadataJSON string
	ProjectID    string
	Visibility   string
}

func newMediaCmd() *cobra.Command {
	mediaCmd := &cobra.Command{
		Use:   "media",
		Short: "上传并查询稳定媒体 URL",
	}

	getCmd := &cobra.Command{
		Use:   "get <media-id>",
		Short: "获取媒体元数据和稳定 URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var media types.Media
			if err := currentClient().GetJSON(context.Background(), "/media/"+args[0], nil, &media); err != nil {
				return err
			}
			return writeOutput(cmd, media)
		},
	}

	uploadCmd := &cobra.Command{
		Use:   "upload <path>",
		Short: "上传本地文件并生成稳定媒体 URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunMode(cmd) {
				return writeDryRunPreview(cmd, "media.upload", map[string]any{
					"path": args[0],
					"request": map[string]any{
						"method": "POST",
						"path":   "/media/upload",
						"body": map[string]any{
							"path":          args[0],
							"filename":      flagString(cmd, "filename"),
							"content_type":  flagString(cmd, "content-type"),
							"metadata_json": flagString(cmd, "metadata-json"),
							"project_id":    flagString(cmd, "project-id"),
							"visibility":    flagString(cmd, "visibility"),
						},
					},
				})
			}
			result, err := uploadMedia(context.Background(), args[0], mediaUploadOptions{
				Filename:     flagString(cmd, "filename"),
				ContentType:  flagString(cmd, "content-type"),
				MetadataJSON: flagString(cmd, "metadata-json"),
				ProjectID:    flagString(cmd, "project-id"),
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
	uploadCmd.Flags().String("metadata-json", "", "附带的 JSON 元数据字符串")
	uploadCmd.Flags().String("project-id", "", "覆盖默认项目 ID（默认：当前活动项目）")
	uploadCmd.Flags().String("visibility", "", "稳定 URL 的可见性，例如 private | unlisted | public")

	mediaCmd.AddCommand(getCmd, uploadCmd)
	return mediaCmd
}

func uploadMedia(ctx context.Context, path string, opts mediaUploadOptions) (map[string]any, error) {
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

	var media types.Media
	if err := currentClient().UploadFile(ctx, "/media/upload", path, api.UploadFileOptions{
		Filename:    filename,
		ContentType: contentType,
		Fields:      fields,
	}, &media); err != nil {
		return nil, err
	}

	result := map[string]any{
		"media_id":      media.ID,
		"filename":      media.Filename,
		"content_type":  media.ContentType,
		"size_bytes":    media.SizeBytes,
		"created_at":    media.CreatedAt,
		"uploaded_from": path,
	}
	if media.ArtifactID != "" {
		result["artifact_id"] = media.ArtifactID
	}
	if media.ProjectID != "" {
		result["project_id"] = media.ProjectID
	}
	if media.URL != "" {
		result["url"] = media.URL
	}
	if media.Visibility != "" {
		result["visibility"] = media.Visibility
	}
	if media.SHA256 != "" {
		result["sha256"] = media.SHA256
	}
	return result, nil
}
