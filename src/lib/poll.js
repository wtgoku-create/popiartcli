// src/lib/poll.js
// 通用作业轮询器。当作业达到终止状态时退出。
// 终止状态：done（完成）、failed（失败）、cancelled（取消）
// 支持 --wait 标志：阻塞直到终止，在 stderr 上打印状态。
// 不带 --wait：立即返回 job_id。

import { get } from './client.js';
import { out, err } from './output.js';

const TERMINAL = new Set(['done', 'failed', 'cancelled']);
const DEFAULT_INTERVAL_MS = 2000;
const DEFAULT_MAX_POLLS = 300; // 在2秒间隔下为10分钟

export async function pollJob(jobId, {
  wait = false,
  intervalMs = DEFAULT_INTERVAL_MS,
  maxPolls = DEFAULT_MAX_POLLS,
  plain = false,
} = {}) {
  if (!wait) {
    return out({ job_id: jobId, status: 'pending', polling: false }, { plain });
  }

  let polls = 0;
  while (polls < maxPolls) {
    const job = await get(`/jobs/${jobId}`);
    const status = job.status;

    if (TERMINAL.has(status)) {
      if (status === 'failed') {
        err('JOB_FAILED', job.error?.message ?? 'Job failed', {
          job_id: jobId,
          status,
          error: job.error,
        });
      }
      return out(job, { plain });
    }

    // 在 stderr 上打印进度提示 (避免污染 stdout 中的 JSON)
    process.stderr.write(`\r⏳ ${jobId} — ${status} (${polls * intervalMs / 1000}s)   `);
    await sleep(intervalMs);
    polls++;
  }

  err('POLL_TIMEOUT', `Job ${jobId} did not complete within the timeout`, {
    job_id: jobId,
    polls,
    timeout_ms: polls * intervalMs,
  });
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
