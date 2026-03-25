// src/lib/client.js
// 轻量级的 fetch 包装器。附加 Bearer 令牌，验证错误。

import { load } from './config.js';
import { httpErr, err } from './output.js';

export async function request(method, path, { body, query, token: overrideToken, stream = false } = {}) {
  const { endpoint, token: cfgToken } = load();
  const token = overrideToken ?? cfgToken;

  const url = new URL(path, endpoint + '/');
  if (query) {
    for (const [k, v] of Object.entries(query)) {
      if (v !== undefined && v !== null) url.searchParams.set(k, String(v));
    }
  }

  const headers = {
    'Accept': 'application/json',
    'User-Agent': `popiart-cli/${process.env.npm_package_version ?? '0.1.0'}`,
  };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  if (body !== undefined) headers['Content-Type'] = 'application/json';

  let res;
  try {
    res = await fetch(url.toString(), {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  } catch (e) {
    err('NETWORK_ERROR', `Request failed: ${e.message}`, { url: url.toString() });
  }

  if (stream) return res; // 调用者以流的格式读取响应体

  let json;
  const text = await res.text();
  try { json = JSON.parse(text); } catch { json = { message: text }; }

  if (!res.ok) httpErr(res.status, json);
  return json;
}

export const get    = (path, opts) => request('GET',    path, opts);
export const post   = (path, opts) => request('POST',   path, opts);
export const del    = (path, opts) => request('DELETE', path, opts);
export const patch  = (path, opts) => request('PATCH',  path, opts);
