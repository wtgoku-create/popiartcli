package cmd

import (
	"context"
	"io"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/poll"
	"github.com/wtgoku-create/popiartcli/internal/types"
)

func newJobsCmd() *cobra.Command {
	jobsCmd := &cobra.Command{
		Use:   "jobs",
		Short: "管理和查询作业执行状态",
	}

	getCmd := &cobra.Command{
		Use:   "get <job-id>",
		Short: "获取作业的当前状态",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var job types.Job
			if err := currentClient().GetJSON(context.Background(), "/jobs/"+args[0], nil, &job); err != nil {
				return err
			}
			return writeOutput(cmd, job)
		},
	}

	waitCmd := &cobra.Command{
		Use:   "wait <job-id>",
		Short: "阻塞当前进程直到作业达到终止状态",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			intervalMs, _ := strconv.Atoi(flagString(cmd, "interval"))
			job, err := poll.WaitForJob(context.Background(), currentClient(), args[0], time.Duration(intervalMs)*time.Millisecond, 300)
			if err != nil {
				return err
			}
			return writeOutput(cmd, job)
		},
	}
	waitCmd.Flags().String("interval", "2000", "轮询间隔（毫秒）")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "列出近期作业",
		RunE: func(cmd *cobra.Command, args []string) error {
			var jobs types.JobListResponse
			if err := currentClient().GetJSON(context.Background(), "/jobs", map[string]string{
				"status":     flagString(cmd, "status"),
				"skill_id":   flagString(cmd, "skill"),
				"project_id": flagString(cmd, "project"),
				"limit":      flagString(cmd, "limit"),
				"offset":     flagString(cmd, "offset"),
			}, &jobs); err != nil {
				return err
			}
			return writeOutput(cmd, jobs)
		},
	}
	listCmd.Flags().String("status", "", "按状态过滤: pending|running|done|failed|cancelled")
	listCmd.Flags().String("skill", "", "按技能过滤")
	listCmd.Flags().String("project", "", "按项目过滤")
	listCmd.Flags().String("limit", "20", "最大结果数量")
	listCmd.Flags().String("offset", "0", "分页偏移量")

	cancelCmd := &cobra.Command{
		Use:   "cancel <job-id>",
		Short: "请求取消正在运行的作业",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp any
			if err := currentClient().PostJSON(context.Background(), "/jobs/"+args[0]+"/cancel", map[string]any{}, &resp); err != nil {
				return err
			}
			return writeOutput(cmd, resp)
		},
	}

	logsCmd := &cobra.Command{
		Use:   "logs <job-id>",
		Short: "流式获取作业的执行日志",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagBool(cmd, "follow") {
				res, err := currentClient().Stream(context.Background(), "GET", "/jobs/"+args[0]+"/logs", apiOpts("", "text/event-stream"))
				if err != nil {
					return err
				}
				defer res.Body.Close()
				_, err = io.Copy(cmd.OutOrStdout(), res.Body)
				return err
			}

			var logs []types.LogEntry
			if err := currentClient().GetJSON(context.Background(), "/jobs/"+args[0]+"/logs", nil, &logs); err != nil {
				return err
			}
			return writeOutput(cmd, logs)
		},
	}
	logsCmd.Flags().Bool("follow", false, "跟踪日志流直到作业完成")

	jobsCmd.AddCommand(getCmd, waitCmd, listCmd, cancelCmd, logsCmd)
	return jobsCmd
}
