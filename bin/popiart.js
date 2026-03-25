#!/usr/bin/env node
import { program } from '../src/cli.js';
program.parseAsync(process.argv).catch((err) => {
  process.stderr.write(JSON.stringify({ ok: false, error: { code: 'FATAL', message: err.message } }) + '\n');
  process.exit(1);
});
