package poll

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/wtgoku-create/popiartcli/internal/api"
	"github.com/wtgoku-create/popiartcli/internal/output"
	"github.com/wtgoku-create/popiartcli/internal/types"
)

var terminalStates = map[string]bool{
	"done":      true,
	"failed":    true,
	"cancelled": true,
}

func WaitForJob(ctx context.Context, client *api.Client, jobID string, interval time.Duration, maxPolls int) (*types.Job, error) {
	for pollIndex := 0; pollIndex < maxPolls; pollIndex++ {
		var job types.Job
		if err := client.GetJSON(ctx, "/jobs/"+jobID, nil, &job); err != nil {
			return nil, err
		}

		status := job.Status
		if terminalStates[status] {
			if status == "failed" {
				return nil, output.NewError("JOB_FAILED", messageFromJob(&job), map[string]any{
					"job_id": jobID,
					"status": status,
					"error":  job.Error,
				})
			}
			return &job, nil
		}

		fmt.Fprintf(os.Stderr, "\r⏳ %s - %s (%ds)   ", jobID, status, int(interval.Seconds())*pollIndex)
		time.Sleep(interval)
	}

	return nil, output.NewError("POLL_TIMEOUT", fmt.Sprintf("Job %s did not complete within the timeout", jobID), map[string]any{
		"job_id":          jobID,
		"timeout_seconds": int(interval.Seconds()) * maxPolls,
	})
}

func messageFromJob(job *types.Job) string {
	if job == nil || job.Error == nil {
		return "Job failed"
	}
	if job.Error.Message != "" {
		return job.Error.Message
	}
	return "Job failed"
}
