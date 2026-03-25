// src/lib/input.js
// 解析 --input 标志的输入。
// 接收:
//   - @path/to/file.json   → 读取文件
//   - -                    → 读取 stdin
//   - '{"key":"val"}'      → 解析内联 JSON
//   - undefined            → 返回 {}

import { readFileSync } from 'fs';
import { err } from './output.js';

export function resolveInput(raw) {
  if (!raw) return {};

  // 文件引用: @/path 或 @./path 或 @filename.json
  if (raw.startsWith('@')) {
    const filePath = raw.slice(1);
    return readJsonFile(filePath, `输入文件: ${filePath}`);
  }

  // 标准输入 (stdin)
  if (raw === '-') {
    try {
      const text = readFileSync('/dev/stdin', 'utf8');
      return parseJson(text, '标准输入');
    } catch {
      err('INPUT_ERROR', '从标准输入读取失败');
    }
  }

  // 内联 JSON
  return parseJson(raw, '内联输入');
}

function readJsonFile(filePath, label) {
  try {
    const text = readFileSync(filePath, 'utf8');
    return parseJson(text, label);
  } catch (e) {
    if (e.code === 'ENOENT') err('INPUT_NOT_FOUND', `未找到文件: ${filePath}`);
    throw e;
  }
}

function parseJson(text, label) {
  try {
    return JSON.parse(text);
  } catch (e) {
    err('INPUT_PARSE_ERROR', `${label} 中存在无效的 JSON: ${e.message}`, {
      hint: '使用 @file.json 进行文件输入，或者传入有效的 JSON 字符串',
    });
  }
}
