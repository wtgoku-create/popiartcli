// src/commands/run.js
// popiart run <skill-id> [--input '{...}' | @file.json | -]
// 始终立即返回 { job_id }。
// 使用 --wait：阻塞直到作业达到终止状态。

import { Command } from 'commander';
import { post } from '../lib/client.js';
import { out } from '../lib/output.js';
import { resolveInput } from '../lib/input.js';
import { pollJob } from '../lib/poll.js';
import { load } from '../lib/config.js';

export const runCmd = new Command('run')
  .description('调用一个技能并返回一个 job_id')
  .argument('<skill-id>', '要调用的技能')
  .option('-i, --input <json>',       '输入 JSON 字符串、@file.json，或用 - 表示标准输入')
  .option('-w, --wait',               '阻塞进程直到作业完成')
  .option('--interval <ms>',          '轮询间隔（毫秒，默认：2000）', '2000')
  .option('--priority <level>',       '作业优先级: low | normal | high', 'normal')
  .option('--idempotency-key <key>',  '用于安全重试的幂等键')
  .option('--plain',                  '输出人类可读内容（而不是 JSON）')
  .action(async (skillId, opts) => {
    const { project } = load();
    const input = resolveInput(opts.input);

    const body = {
      skill_id: skillId,
      input,
      priority: opts.priority,
      ...(project  ? { project_id: project }           : {}),
      ...(opts.idempotencyKey ? { idempotency_key: opts.idempotencyKey } : {}),
    };

    const job = await post('/jobs', { body });

    await pollJob(job.job_id, {
      wait:       !!opts.wait,
      intervalMs: parseInt(opts.interval, 10),
      plain:      !!opts.plain,
    });
  });
