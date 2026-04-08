package cmd

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wtgoku-create/popiartcli/internal/config"
	"github.com/wtgoku-create/popiartcli/internal/input"
	"github.com/wtgoku-create/popiartcli/internal/localskills"
	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/poll"
	"github.com/wtgoku-create/popiartcli/internal/seed"
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
			resolvedSkillID, err := resolveRunnableSkillID(context.Background(), args[0])
			if err != nil {
				return err
			}
			if job, handled, err := maybeRunOfficialRuntimeDirectFallbackJob(context.Background(), resolvedSkillID, payload, flagString(cmd, "priority"), "", flagString(cmd, "idempotency-key")); handled {
				if err != nil {
					return err
				}
				return writeJobResultOrWait(cmd, job)
			}

			cfg := config.Load()
			body := map[string]any{
				"skill_id": resolvedSkillID,
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

			interval, err := intervalDuration(cmd, "interval")
			if err != nil {
				return err
			}
			done, err := poll.WaitForJob(context.Background(), currentClient(), jobID, interval, 300)
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

func resolveRunnableSkillID(ctx context.Context, skillID string) (string, error) {
	skillID = strings.TrimSpace(skillID)

	if installed, shouldUseLocal, err := resolveRunnableInstalledSkill(ctx, skillID); err != nil {
		return "", err
	} else if shouldUseLocal {
		return localskillsEffectiveRuntimeSkillID(installed), nil
	}

	if err := validateBundledSkillRun(ctx, skillID); err != nil {
		return "", err
	}
	return skillID, nil
}

func resolveRunnableInstalledSkill(ctx context.Context, skillID string) (localskills.InstalledSkill, bool, error) {
	skill, ok, err := localskills.FindInstalled(skillID)
	if err != nil {
		return localskills.InstalledSkill{}, false, err
	}
	if !ok {
		return localskills.InstalledSkill{}, false, nil
	}

	active, err := localskills.IsActive(skill.Manifest.Slug)
	if err != nil {
		return localskills.InstalledSkill{}, false, err
	}
	if !active {
		exists, err := remoteSkillExists(ctx, skillID)
		if err != nil {
			return localskills.InstalledSkill{}, false, err
		}
		if exists {
			return localskills.InstalledSkill{}, false, nil
		}
	}

	if skill.Manifest.RequiresPopiartAuth {
		if _, err := config.RequireToken(); err != nil {
			return localskills.InstalledSkill{}, false, requireTokenError()
		}
	}

	if skill.Manifest.Execution.Runner != "" && skill.Manifest.Execution.Runner != "popiart" {
		return localskills.InstalledSkill{}, false, output.NewError("LOCAL_SKILL_UNSUPPORTED", "当前仅支持由 popiart 执行的本地 skill", map[string]any{
			"skill_id": skillID,
			"runner":   skill.Manifest.Execution.Runner,
		})
	}
	if skill.Manifest.Execution.Mode != "remote-runtime" {
		return localskills.InstalledSkill{}, false, output.NewError("LOCAL_SKILL_UNSUPPORTED", "当前仅支持 execution.mode=remote-runtime 的本地 skill", map[string]any{
			"skill_id": skillID,
			"mode":     skill.Manifest.Execution.Mode,
		})
	}

	return skill, true, nil
}

func validateBundledSkillRun(ctx context.Context, skillID string) error {
	if _, ok := seed.FindBundledSkill(skillID); !ok {
		return nil
	}
	if _, ok := officialRuntimeSkillForID(skillID); ok {
		return nil
	}

	exists, err := remoteSkillExists(ctx, skillID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return output.NewError("LOCAL_ONLY_SKILL", "该 skill 仅是 CLI 内置 seed helper，不能直接提交到远端执行", map[string]any{
		"skill_id": skillID,
		"hint":     "使用 `popiart skills get " + skillID + "` 查看说明，或选择对应的远程 runtime skill 再执行 `popiart run`",
	})
}
