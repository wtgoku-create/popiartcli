package cmd

import (
	"context"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/input"
	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/poll"
	"github.com/wtgoku-create/popiartcli/internal/types"
)

func newRunCmd() *cobra.Command {
	runCmd := &cobra.Command{
		Use:   "run <skill-id>",
		Short: "调用一个技能并返回一个 job_id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := input.Resolve(flagString(cmd, "input"))
			if err != nil {
				return err
			}

			cfg := config.Load()
			body := map[string]any{
				"skill_id": args[0],
				"input":    payload,
				"priority": flagString(cmd, "priority"),
			}
			if cfg.Project != "" {
				body["project_id"] = cfg.Project
			}
			if value := flagString(cmd, "idempotency-key"); value != "" {
				body["idempotency_key"] = value
			}

			var job types.Job
			if err := currentClient().PostJSON(context.Background(), "/jobs", body, &job); err != nil {
				return err
			}

			if !flagBool(cmd, "wait") {
				return writeOutput(cmd, job)
			}

			jobID := job.JobID
			if jobID == "" {
				return output.NewError("CLI_ERROR", "作业响应中缺少 job_id", nil)
			}

			intervalMs, _ := strconv.Atoi(flagString(cmd, "interval"))
			done, err := poll.WaitForJob(context.Background(), currentClient(), jobID, time.Duration(intervalMs)*time.Millisecond, 300)
			if err != nil {
				return err
			}
			return writeOutput(cmd, done)
		},
	}

	runCmd.Flags().StringP("input", "i", "", "输入 JSON 字符串、@file.json，或用 - 表示标准输入")
	runCmd.Flags().BoolP("wait", "w", false, "阻塞进程直到作业完成")
	runCmd.Flags().String("interval", "2000", "轮询间隔（毫秒，默认：2000）")
	runCmd.Flags().String("priority", "normal", "作业优先级: low | normal | high")
	runCmd.Flags().String("idempotency-key", "", "用于安全重试的幂等键")
	return runCmd
}
