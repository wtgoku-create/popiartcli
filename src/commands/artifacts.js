// src/commands/artifacts.js
import { Command } from 'commander';
import { get } from '../lib/client.js';
import { out, err } from '../lib/output.js';
import { writeFileSync, mkdirSync } from 'fs';
import { dirname, join, basename } from 'path';

export const artifactsCmd = new Command('artifacts')
  .description('拉取并管理作业生成的工件');

// popiart artifacts list <job-id>
artifactsCmd
  .command('list <job-id>')
  .description('列出作业生成的工件')
  .action(async (jobId) => {
    const artifacts = await get(`/jobs/${jobId}/artifacts`);
    out(artifacts);
  });

// popiart artifacts get <artifact-id>
artifactsCmd
  .command('get <artifact-id>')
  .description('获取工件元数据')
  .action(async (artifactId) => {
    const artifact = await get(`/artifacts/${artifactId}`);
    out(artifact);
  });

// popiart artifacts pull <artifact-id> [--out <path>]
// 将工件二进制文件/内容下载到磁盘。
artifactsCmd
  .command('pull <artifact-id>')
  .description('将工件下载到磁盘')
  .option('-o, --out <path>',   '输出文件路径（默认：./<artifact-name>）')
  .option('--stdout',            '将内容写入 stdout 而不是文件')
  .action(async (artifactId, opts) => {
    // 首先获取元数据以知道文件名和大小
    const meta = await get(`/artifacts/${artifactId}`);

    const res = await get(`/artifacts/${artifactId}/content`, { stream: true });
    if (opts.stdout) {
      const reader = res.body.getReader();
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        process.stdout.write(value);
      }
      return;
    }

    const outPath = opts.out ?? join('.', meta.filename ?? `artifact-${artifactId}`);
    mkdirSync(dirname(outPath), { recursive: true });

    const chunks = [];
    const reader = res.body.getReader();
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      chunks.push(value);
    }
    const buf = Buffer.concat(chunks.map((c) => Buffer.from(c)));
    writeFileSync(outPath, buf);

    out({
      artifact_id: artifactId,
      saved_to: outPath,
      bytes: buf.length,
      content_type: meta.content_type,
    });
  });

// popiart artifacts pull-all <job-id> [--dir <dir>]
// 便捷功能：将作业中的所有工件拉取到一个目录中。
artifactsCmd
  .command('pull-all <job-id>')
  .description('将作业中的所有工件下载到一个目录中')
  .option('-d, --dir <dir>', '输出目录（默认：./<job-id>）')
  .action(async (jobId, opts) => {
    const { items: artifacts } = await get(`/jobs/${jobId}/artifacts`);
    if (!artifacts?.length) { out({ job_id: jobId, artifacts_downloaded: 0 }); return; }

    const dir = opts.dir ?? join('.', jobId);
    mkdirSync(dir, { recursive: true });

    const results = [];
    for (const art of artifacts) {
      const outPath = join(dir, art.filename ?? `artifact-${art.id}`);
      const res = await get(`/artifacts/${art.id}/content`, { stream: true });

      const chunks = [];
      const reader = res.body.getReader();
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        chunks.push(value);
      }
      const buf = Buffer.concat(chunks.map((c) => Buffer.from(c)));
      writeFileSync(outPath, buf);
      results.push({ artifact_id: art.id, saved_to: outPath, bytes: buf.length });
    }

    out({ job_id: jobId, artifacts_downloaded: results.length, files: results });
  });
