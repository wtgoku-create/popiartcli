// src/lib/output.js
// 所有响应共享一个信封：{ ok, data?, error? }
// --plain 标志切换到适合交互式 shell 的人类可读文本。

export function out(data, { plain = false } = {}) {
  if (plain) {
    printPlain(data);
  } else {
    process.stdout.write(JSON.stringify({ ok: true, data }) + '\n');
  }
}

export function err(code, message, details = {}) {
  process.stderr.write(
    JSON.stringify({ ok: false, error: { code, message, ...details } }) + '\n'
  );
  process.exit(1);
}

// 映射 HTTP 状态 → 规范的错误代码
export const HTTP_ERRORS = {
  400: 'BAD_REQUEST',
  401: 'UNAUTHENTICATED',
  403: 'FORBIDDEN',
  404: 'NOT_FOUND',
  409: 'CONFLICT',
  422: 'VALIDATION_ERROR',
  429: 'RATE_LIMITED',
  500: 'SERVER_ERROR',
  503: 'SERVICE_UNAVAILABLE',
};

export function httpErr(status, body = {}) {
  const code = HTTP_ERRORS[status] ?? 'HTTP_ERROR';
  err(code, body.message ?? `HTTP ${status}`, { status, ...body });
}

// 为 --plain 模式提供友好的打印
function printPlain(data) {
  if (data === null || data === undefined) return;
  if (typeof data === 'string') { console.log(data); return; }
  if (Array.isArray(data)) {
    for (const item of data) printPlain(item);
    return;
  }
  if (typeof data === 'object') {
    for (const [k, v] of Object.entries(data)) {
      if (typeof v === 'object' && v !== null) {
        console.log(`${k}:`);
        printPlain(v);
      } else {
        console.log(`  ${k}: ${v}`);
      }
    }
  }
}
