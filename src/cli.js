// Legacy Node CLI surface kept only as a migration reference.
// The active product surface is the Go implementation under ./cmd/popiart.
// This file is intentionally no longer the repository's default distribution path.

import { Command } from 'commander';
import { authCmd }      from './commands/auth.js';
import { skillsCmd }    from './commands/skills.js';
import { runCmd }       from './commands/run.js';
import { jobsCmd }      from './commands/jobs.js';
import { artifactsCmd } from './commands/artifacts.js';
import { budgetCmd }    from './commands/budget.js';
import { projectCmd }   from './commands/project.js';
import { save }         from './lib/config.js';
import { err }          from './lib/output.js';

export const program = new Command();

program
  .name('popiart')
  .description('为 Coding Agent 提供创作者技能入口与多模态授权计费的 CLI')
  .version('0.1.0', '-v, --version')
  // 在子命令处理程序之前应用的全局标志
  .option('--endpoint <url>',   '覆盖本次调用的 API 端点')
  .option('--project <id>',     '覆盖本次调用的活动项目')
  .option('--plain',            '人类可读的输出（默认：JSON）')
  .option('--no-color',         '在纯文本输出中禁用 ANSI 颜色')
  // 错误输出格式
  .configureOutput({
    writeErr: (str) => process.stderr.write(str),
    outputError: (str, write) => write(
      JSON.stringify({ ok: false, error: { code: 'CLI_ERROR', message: str.trim() } }) + '\n'
    ),
  });

// 在每个命令运行前应用全局覆盖
program.hook('preAction', (thisCommand, actionCommand) => {
  const opts = program.opts();
  if (opts.endpoint) save({ endpoint: opts.endpoint });
  if (opts.project)  save({ project: opts.project });
});

// 注册所有命令树
program.addCommand(authCmd);
program.addCommand(skillsCmd);
program.addCommand(runCmd);
program.addCommand(jobsCmd);
program.addCommand(artifactsCmd);
program.addCommand(budgetCmd);
program.addCommand(projectCmd);

// 捕获未知命令
program.on('command:*', (operands) => {
  err('UNKNOWN_COMMAND', `未知命令: ${operands[0]}`, {
    hint: '运行 `popiart --help` 以查看可用命令',
  });
});
