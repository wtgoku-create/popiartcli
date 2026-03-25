// src/commands/jobs.js
import { Command } from 'commander';
import { get, post, del } from '../lib/client.js';
import { out } from '../lib/output.js';
import { pollJob } from '../lib/poll.js';

export const jobsCmd = new Command('jobs')
  .description('管理和查询作业执行状态');

// popiart jobs get <job-id>
jobsCmd
  .command('get <job-id>')
  .description('获取作业的当前状态')
  .option('--plain', '人类可读的输出')
  .action(async (jobId, opts) => {
    const job = await get(`/jobs/${jobId}`);
    out(job, { plain: !!opts.plain });
  });

// popiart jobs wait <job-id>
jobsCmd
  .command('wait <job-id>')
  .description('阻塞当前进程直到作业达到终止状态')
  .option('--interval <ms>', '轮询间隔（毫秒）', '2000')
  .option('--plain', '人类可读的输出')
  .action(async (jobId, opts) => {
    await pollJob(jobId, {
      wait: true,
      intervalMs: parseInt(opts.interval, 10),
      plain: !!opts.plain,
    });
  });

// popiart jobs list [--status <status>] [--skill <skill-id>]
jobsCmd
  .command('list')
  .description('列出近期作业')
  .option('--status <status>',  '按状态过滤: pending|running|done|failed|cancelled')
  .option('--skill <skill-id>', '按技能过滤')
  .option('--limit <n>',        '最大结果数量', '20')
  .option('--offset <n>',       '分页偏移量', '0')
  .option('--plain', '人类可读的输出')
  .action(async (opts) => {
    const jobs = await get('/jobs', {
      query: {
        status: opts.status,
        skill_id: opts.skill,
        limit: opts.limit,
        offset: opts.offset,
      },
    });
    out(jobs, { plain: !!opts.plain });
  });

// popiart jobs cancel <job-id>
jobsCmd
  .command('cancel <job-id>')
  .description('请求取消正在运行的作业')
  .action(async (jobId) => {
    const result = await post(`/jobs/${jobId}/cancel`, { body: {} });
    out(result);
  });

// popiart jobs logs <job-id>
jobsCmd
  .command('logs <job-id>')
  .description('流式获取作业的执行日志')
  .option('--follow', '跟踪日志流直到作业完成')
  .action(async (jobId, opts) => {
    if (opts.follow) {
      const res = await get(`/jobs/${jobId}/logs`, { stream: true });
      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        process.stdout.write(decoder.decode(value));
      }
    } else {
      const logs = await get(`/jobs/${jobId}/logs`);
      out(logs);
    }
  });
