// src/commands/skills.js
import { Command } from 'commander';
import { get } from '../lib/client.js';
import { out } from '../lib/output.js';

export const skillsCmd = new Command('skills')
  .description('在注册表中发现可用技能');

// popiart skills list [--tag <tag>] [--search <query>]
skillsCmd
  .command('list')
  .description('列出所有可用技能')
  .option('-t, --tag <tag>',       '按标签过滤')
  .option('-s, --search <query>',  '全文搜索')
  .option('--limit <n>',           '最大结果数量', '50')
  .option('--offset <n>',          '分页偏移量', '0')
  .action(async (opts) => {
    const skills = await get('/skills', {
      query: {
        tag:    opts.tag,
        search: opts.search,
        limit:  opts.limit,
        offset: opts.offset,
      },
    });
    out(skills);
  });

// popiart skills get <skill-id>
skillsCmd
  .command('get <skill-id>')
  .description('获取技能的完整模式和描述')
  .action(async (skillId) => {
    const skill = await get(`/skills/${skillId}`);
    out(skill);
  });

// popiart skills schema <skill-id>
skillsCmd
  .command('schema <skill-id>')
  .description('打印某个技能的输入/输出 JSON 模式')
  .action(async (skillId) => {
    const { input_schema, output_schema } = await get(`/skills/${skillId}/schema`);
    out({ input_schema, output_schema });
  });
