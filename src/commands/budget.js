// src/commands/budget.js
import { Command } from 'commander';
import { get } from '../lib/client.js';
import { out } from '../lib/output.js';

export const budgetCmd = new Command('budget')
  .description('查看令牌使用情况和剩余预算');

// popiart budget status
budgetCmd
  .command('status')
  .description('显示当前周期的预算和使用情况摘要')
  .option('--project <id>', '限定到特定项目')
  .action(async (opts) => {
    const budget = await get('/budget', {
      query: opts.project ? { project_id: opts.project } : {},
    });
    out(budget);
  });

// popiart budget usage [--since <iso-date>] [--until <iso-date>]
budgetCmd
  .command('usage')
  .description('按技能和时间段进行详细的使用情况细分')
  .option('--since <date>',    '开始日期 (ISO 8601)')
  .option('--until <date>',    '结束日期 (ISO 8601，默认：当前时间)')
  .option('--group-by <field>','分组方式: skill|day|project', 'skill')
  .option('--project <id>',   '限定到特定项目')
  .action(async (opts) => {
    const usage = await get('/budget/usage', {
      query: {
        since:      opts.since,
        until:      opts.until,
        group_by:   opts.groupBy,
        project_id: opts.project,
      },
    });
    out(usage);
  });

// popiart budget limits
budgetCmd
  .command('limits')
  .description('显示速率限制和配额配置')
  .action(async () => {
    const limits = await get('/budget/limits');
    out(limits);
  });
