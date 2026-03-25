// src/lib/config.js
// 将会话令牌和 API 端点存储在 ~/.popiart/config.json
// 可用时使用 XDG_CONFIG_HOME。

import { readFileSync, writeFileSync, mkdirSync, existsSync } from 'fs';
import { homedir } from 'os';
import { join } from 'path';

const CONFIG_DIR = process.env.POPIART_CONFIG_DIR
  ?? join(process.env.XDG_CONFIG_HOME ?? homedir(), '.popiart');
const CONFIG_FILE = join(CONFIG_DIR, 'config.json');

const DEFAULTS = {
  endpoint: process.env.POPIART_ENDPOINT ?? 'https://api.creatoragentos.io/v1',
  token: process.env.POPIART_TOKEN ?? null,
  project: process.env.POPIART_PROJECT ?? null,
};

let _cache = null;

export function load() {
  if (_cache) return _cache;
  if (!existsSync(CONFIG_FILE)) {
    _cache = { ...DEFAULTS };
    return _cache;
  }
  try {
    _cache = { ...DEFAULTS, ...JSON.parse(readFileSync(CONFIG_FILE, 'utf8')) };
    return _cache;
  } catch {
    _cache = { ...DEFAULTS };
    return _cache;
  }
}

export function save(patch) {
  const current = load();
  const next = { ...current, ...patch };
  mkdirSync(CONFIG_DIR, { recursive: true });
  writeFileSync(CONFIG_FILE, JSON.stringify(next, null, 2) + '\n', { mode: 0o600 });
  _cache = next;
  return next;
}

export function requireToken() {
  const { token } = load();
  if (!token) {
    // 同步路径 — 内联错误以便不需要顶层 await
    process.stderr.write(JSON.stringify({
      ok: false,
      error: {
        code: 'UNAUTHENTICATED',
        message: '没有会话令牌。请运行: popiart auth login',
      }
    }) + '\n');
    process.exit(1);
  }
  return token;
}

export function configPath() {
  return CONFIG_FILE;
}
