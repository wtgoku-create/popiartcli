#!/usr/bin/env node
// Legacy Node shim kept only for reference during the Go migration.
// The authoritative CLI lives in ./cmd/popiart and package.json no longer
// publishes this file as the repo's active entrypoint.
import { program } from '../src/cli.js';
program.parseAsync(process.argv).catch((err) => {
  process.stderr.write(JSON.stringify({ ok: false, error: { code: 'FATAL', message: err.message } }) + '\n');
  process.exit(1);
});
