package cmd

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/popiart"
)

func newJobsCmd() *cobra.Command {
	jobsCmd := &cobra.Command{
		Use:   "jobs",
		Short: "管理和查询作业执行状态",
	}

	getCmd := &cobra.Command{
		Use:   "get <task-id>",
		Short: "获取任务的当前状态",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			task, err := popiart.GetTaskDetail(context.Background(), currentClient(), args[0])
			if err != nil {
				return err
			}
			if urls, err := popiart.GetTaskDownloadURLs(context.Background(), currentClient(), args[0]); err == nil && len(urls) > 0 {
				task.DownloadURLs = append([]string(nil), urls...)
				task.ResultURLs = append([]string(nil), urls...)
			}
			return writeOutput(cmd, taskDetailOutput(task))
		},
	}

	waitCmd := &cobra.Command{
		Use:   "wait <task-id>",
		Short: "阻塞当前进程直到任务达到终止状态",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			interval, err := intervalDuration(cmd, "interval")
			if err != nil {
				return err
			}
			task, err := popiart.WaitForTask(context.Background(), currentClient(), args[0], interval, 300)
			if err != nil {
				return err
			}
			return writeOutput(cmd, taskDetailOutput(task))
		},
	}
	waitCmd.Flags().String("interval", "2000", "轮询间隔（毫秒）")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "列出近期作业",
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, err := parseJobsListIntFlag(cmd, "limit", 20)
			if err != nil {
				return err
			}
			offset, err := parseJobsListIntFlag(cmd, "offset", 0)
			if err != nil {
				return err
			}
			page, err := popiart.ListTasks(
				context.Background(),
				currentClient(),
				flagString(cmd, "status"),
				strconv.Itoa(limit),
				strconv.Itoa(offset),
			)
			if err != nil {
				return err
			}

			items := make([]map[string]any, 0, len(page.Items))
			for _, task := range page.Items {
				items = append(items, taskDetailOutput(task))
			}
			return writeOutput(cmd, map[string]any{
				"items":  items,
				"total":  page.Total,
				"limit":  firstPositive(page.Limit, limit),
				"offset": page.Offset,
			})
		},
	}
	listCmd.Flags().String("status", "", "按状态过滤: pending|running|done|failed|cancelled")
	listCmd.Flags().String("skill", "", "按技能过滤")
	listCmd.Flags().String("project", "", "按项目过滤")
	listCmd.Flags().String("limit", "20", "最大结果数量")
	listCmd.Flags().String("offset", "0", "分页偏移量")

	cancelCmd := &cobra.Command{
		Use:   "cancel <task-id>",
		Short: "请求取消正在运行的任务",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return output.NewError("UNSUPPORTED_IN_POPI_ART_MODE", "当前模式暂不支持取消主站任务", map[string]any{
				"hint": "请在网站任务中心取消任务，或等待后续 task cancel 能力落地",
			})
		},
	}

	logsCmd := &cobra.Command{
		Use:   "logs <task-id>",
		Short: "流式获取任务的执行日志",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return output.NewError("UNSUPPORTED_IN_POPI_ART_MODE", "当前模式不支持主站任务日志流", map[string]any{
				"hint": "请改用 `popiart jobs get <task-id>` 查看任务状态，或在网站任务详情页查看执行信息",
			})
		},
	}
	logsCmd.Flags().Bool("follow", false, "跟踪日志流直到作业完成")

	jobsCmd.AddCommand(getCmd, waitCmd, listCmd, cancelCmd, logsCmd)
	return jobsCmd
}

// taskDetailOutput 把主站任务详情整理成兼容 jobs 输出的扁平结构。
func taskDetailOutput(task popiart.TaskDetail) map[string]any {
	result := map[string]any{
		"job_id":   task.Identifier(),
		"task_id":  task.Identifier(),
		"status":   task.Status,
		"type":     task.Type,
		"sub_type": task.SubType,
	}
	if task.AIModelCode != "" {
		result["model"] = task.AIModelCode
	}
	if len(task.DownloadURLs) > 0 {
		result["download_urls"] = append([]string(nil), task.DownloadURLs...)
	}
	if len(task.ResultURLs) > 0 {
		result["result_urls"] = append([]string(nil), task.ResultURLs...)
	}
	if task.OutputText != "" {
		result["output_text"] = task.OutputText
	}
	if task.Text != "" {
		result["text"] = task.Text
	}
	if task.UserErrorTipMsg != "" {
		result["user_error_tip_msg"] = task.UserErrorTipMsg
	}
	if task.ErrorMsg != "" {
		result["error_msg"] = task.ErrorMsg
	}
	return result
}

func parseJobsListIntFlag(cmd *cobra.Command, name string, fallback int) (int, error) {
	raw := flagString(cmd, name)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, output.NewError("VALIDATION_ERROR", "无效的 jobs list 分页参数", map[string]any{
			"flag":  name,
			"value": raw,
			"hint":  "请传入大于等于 0 的整数",
		})
	}
	return value, nil
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
