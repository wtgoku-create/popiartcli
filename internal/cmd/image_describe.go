package cmd

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/popiart"
)

func newImageDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe [prompt]",
		Short: "识别一张图片并返回描述性 prompt",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, preview, err := resolveImageDescribeInput(cmd, args)
			if err != nil {
				return err
			}
			return executeImageDescribeCommand(cmd, payload, preview)
		},
	}

	cmd.Flags().String("model", "", "用于图片理解的多模态模型 ID")
	cmd.Flags().String("image", "", "源图 URL 或本地文件路径")
	cmd.Flags().String("from", "", "源图路径或 URL（等同于 --image）")
	cmd.Flags().String("source-artifact-id", "", "已上传源图的 artifact_id")
	cmd.Flags().String("prompt", "", "可选附加说明，告诉模型描述重点或输出风格")
	cmd.Flags().String("notes", "", "额外约束说明")
	cmd.Flags().String("interval", "2000", "轮询间隔（毫秒，默认：2000）")
	cmd.Flags().String("priority", "normal", "作业优先级: low | normal | high")
	cmd.Flags().String("idempotency-key", "", "用于安全重试的幂等键")
	_ = cmd.MarkFlagRequired("model")
	return cmd
}

func resolveImageDescribeInput(cmd *cobra.Command, args []string) (map[string]any, map[string]any, error) {
	modelID := strings.TrimSpace(flagString(cmd, "model"))
	if modelID == "" {
		return nil, nil, invalidFlagValueError("--model", "", "请传入一个支持图片理解的多模态模型 ID")
	}

	payload, preview, err := resolveImageSourceInput(cmd)
	if err != nil {
		return nil, nil, err
	}

	instruction := strings.TrimSpace(flagString(cmd, "prompt"))
	if instruction == "" && len(args) > 0 {
		instruction = strings.TrimSpace(args[0])
	}

	input := map[string]any{
		"prompt": buildImageDescribeInstruction(instruction, strings.TrimSpace(flagString(cmd, "notes"))),
	}
	if sourceURL := stringValue(payload["image_url"]); sourceURL != "" {
		input["image_url"] = sourceURL
		input["reference_image_url"] = sourceURL
	}
	if sourceArtifactID := stringValue(payload["source_artifact_id"]); sourceArtifactID != "" {
		input["source_artifact_id"] = sourceArtifactID
	}

	return input, preview, nil
}

func buildImageDescribeInstruction(instruction, notes string) string {
	parts := []string{
		"请准确理解输入图片，并输出一段适合直接复用的中文描述性 prompt。",
		"描述需要覆盖主体、场景、构图、镜头视角、光线、材质、风格和可见细节。",
		"只返回最终 prompt 本身，不要解释、不要 Markdown、不要 JSON、不要分点。",
	}
	if instruction = strings.TrimSpace(instruction); instruction != "" {
		parts = append(parts, "补充要求："+instruction)
	}
	if notes = strings.TrimSpace(notes); notes != "" {
		parts = append(parts, "额外约束："+notes)
	}
	return strings.Join(parts, "\n")
}

func executeImageDescribeCommand(cmd *cobra.Command, payload, preview map[string]any) error {
	modelCode := strings.TrimSpace(flagString(cmd, "model"))
	if modelCode == "" {
		return invalidFlagValueError("--model", "", "请传入一个支持图片理解的多模态模型 ID")
	}

	input := hydratePromptEnhancerImageInput(context.Background(), payload)
	model, err := popiart.ResolveModelForCommand(context.Background(), currentClient(), "image.describe", modelCode, popiart.ModelValidationSpec{
		SubType:        501,
		RequiresImages: true,
		ImageCount:     imageCountForDescribeInput(input),
		VideoCount:     0,
		AudioCount:     0,
	})
	if err != nil {
		return err
	}
	req := popiart.TaskRequest{
		Type:        5,
		SubType:     501,
		AIModelCode: model.Code,
		AIModelID:   model.ID,
		ChatPrompt:  stringValue(input["prompt"]),
		Metadata:    map[string]any{},
	}
	if imageURL := stringValue(input["image_url"]); imageURL != "" {
		req.Images = []string{imageURL}
	}
	if notes := strings.TrimSpace(flagString(cmd, "notes")); notes != "" {
		req.Metadata["notes"] = notes
	}
	if len(req.Metadata) == 0 {
		req.Metadata = nil
	}
	if dryRunMode(cmd) {
		return writeDryRunPreview(cmd, "image.describe", map[string]any{
			"model_id": model.ID,
			"source":   preview["source"],
			"request": map[string]any{
				"method": "POST",
				"path":   "/api_client/anime/task/create",
				"body":   popiart.NormalizeTaskRequest(req),
			},
		})
	}

	task, err := popiart.CreateTask(context.Background(), currentClient(), req)
	if err != nil {
		return err
	}

	taskID := task.Identifier()
	if taskID == "" {
		return output.NewError("CLI_ERROR", "图片描述响应缺少 task_id", map[string]any{
			"model_id": model.ID,
		})
	}

	interval, err := intervalDuration(cmd, "interval")
	if err != nil {
		return err
	}
	completedTask, err := popiart.WaitForTask(context.Background(), currentClient(), taskID, interval, 300)
	if err != nil {
		return err
	}
	descriptionPrompt := firstTaskResultURL(completedTask)
	if descriptionPrompt == "" {
		descriptionPrompt = firstTaskDownloadURL(completedTask)
	}
	if descriptionPrompt == "" {
		descriptionPrompt = strings.TrimSpace(completedTask.OutputText)
	}
	if descriptionPrompt == "" {
		descriptionPrompt = strings.TrimSpace(completedTask.Text)
	}

	result := map[string]any{
		"job_id":             taskID,
		"task_id":            taskID,
		"model_id":           model.ID,
		"description_prompt": descriptionPrompt,
	}
	if source := preview["source"]; source != nil {
		result["source"] = source
	}
	return writeOutput(cmd, result)
}

// imageCountForDescribeInput 为图片理解任务生成最小图片数量约束。
func imageCountForDescribeInput(input map[string]any) int {
	if imageURL := stringValue(input["image_url"]); imageURL != "" {
		return 1
	}
	return 0
}

// firstTaskResultURL 优先读取任务结果里的主结果地址。
func firstTaskResultURL(task popiart.TaskDetail) string {
	if len(task.ResultURLs) > 0 {
		return strings.TrimSpace(task.ResultURLs[0])
	}
	return ""
}

// firstTaskDownloadURL 在结果地址为空时回退到下载地址。
func firstTaskDownloadURL(task popiart.TaskDetail) string {
	if len(task.DownloadURLs) > 0 {
		return strings.TrimSpace(task.DownloadURLs[0])
	}
	return ""
}
