// src/commands/project.js
import { Command } from 'commander';
import { get } from '../lib/client.js';
import { save, load } from '../lib/config.js';
import { out, err } from '../lib/output.js';

export const projectCmd = new Command('project')
  .description('读取并管理当前项目上下文');

// popiart project current
projectCmd
  .command('current')
  .description('显示当前活动项目')
  .action(async () => {
    const { project } = load();
    if (!project) {
      out({ project: null, hint: '使用以下命令设置: popiart project use <project-id>' });
      return;
    }
    const detail = await get(`/projects/${project}`);
    out(detail);
  });

// popiart project use <project-id>
projectCmd
  .command('use <project-id>')
  .description('设置活动项目（存储在配置中）')
  .action(async (projectId) => {
    // 验证项目是否存在
    const project = await get(`/projects/${projectId}`);
    save({ project: projectId });
    out({ project_set: projectId, name: project.name });
  });

// popiart project list
projectCmd
  .command('list')
  .description('列出可访问的项目')
  .option('--limit <n>', '最大结果数量', '20')
  .action(async (opts) => {
    const projects = await get('/projects', { query: { limit: opts.limit } });
    out(projects);
  });

// popiart project get <project-id>
projectCmd
  .command('get <project-id>')
  .description('获取项目的完整上下文')
  .action(async (projectId) => {
    const project = await get(`/projects/${projectId}`);
    out(project);
  });

// popiart project context
// 返回完整的上下文对象：活动技能、预算、元数据。
projectCmd
  .command('context')
  .description('获取活动项目的完整运行时上下文')
  .option('--project <id>', '覆盖活动项目')
  .action(async (opts) => {
    const { project: cfgProject } = load();
    const projectId = opts.project ?? cfgProject;
    if (!projectId) {
      err('NO_PROJECT', '未设置项目。请使用: popiart project use <id>');
    }
    const ctx = await get(`/projects/${projectId}/context`);
    out(ctx);
  });
