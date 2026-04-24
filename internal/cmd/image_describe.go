package cmd

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/output"
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
	modelID := strings.TrimSpace(flagString(cmd, "model"))
	if modelID == "" {
		return invalidFlagValueError("--model", "", "请传入一个支持图片理解的多模态模型 ID")
	}

	input := hydratePromptEnhancerImageInput(context.Background(), payload)
	if dryRunMode(cmd) {
		return writeDryRunPreview(cmd, "image.describe", map[string]any{
			"model_id": modelID,
			"source":   preview["source"],
			"request": map[string]any{
				"method": "POST",
				"path":   "/models/infer",
				"body":   buildModelInferBody(modelID, input, flagString(cmd, "priority"), flagString(cmd, "idempotency-key")),
			},
		})
	}

	job, err := submitModelInferJob(context.Background(), modelID, input, flagString(cmd, "priority"), "", flagString(cmd, "idempotency-key"))
	if err != nil {
		return err
	}

	jobID := stringValue(job["job_id"])
	if jobID == "" {
		return output.NewError("CLI_ERROR", "图片描述响应缺少 job_id", map[string]any{
			"model_id": modelID,
		})
	}

	interval, err := intervalDuration(cmd, "interval")
	if err != nil {
		return err
	}
	completedJob, err := waitForDynamicJob(context.Background(), jobID, interval, videoPromptEnhancerMaxPolls)
	if err != nil {
		return err
	}
	descriptionPrompt, err := extractTextFromJob(context.Background(), completedJob, "图片描述结果")
	if err != nil {
		return err
	}

	result := map[string]any{
		"job_id":             jobID,
		"model_id":           modelID,
		"description_prompt": descriptionPrompt,
	}
	if source := preview["source"]; source != nil {
		result["source"] = source
	}
	return writeOutput(cmd, result)
}
