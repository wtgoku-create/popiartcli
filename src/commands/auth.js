// src/commands/auth.js
import { Command } from 'commander';
import { post, get } from '../lib/client.js';
import { save, load, configPath } from '../lib/config.js';
import { out, err } from '../lib/output.js';
import { createInterface } from 'readline';

export const authCmd = new Command('auth')
  .description('管理身份验证和会话令牌');

// popiart auth login --token <token>
// popiart auth login --email <email> --password <password>  (交互式提示)
authCmd
  .command('login')
  .description('获取会话令牌')
  .option('-t, --token <token>',    '直接使用预发的 API 令牌')
  .option('-e, --email <email>',    '电子邮件地址（将提示输入密码）')
  .option('-p, --password <pw>',    '密码（建议使用交互式提示）')
  .option('--endpoint <url>',       '覆盖 API 端点')
  .action(async (opts) => {
    if (opts.endpoint) save({ endpoint: opts.endpoint });

    // 路径 A: 传入原始令牌
    if (opts.token) {
      const me = await get('/auth/me', { token: opts.token });
      save({ token: opts.token });
      out({ user: me, token_saved: true });
      return;
    }

    // 路径 B: 邮箱 + 密码 → 换取会话令牌
    const email = opts.email ?? await prompt('电子邮件: ');
    const password = opts.password ?? await promptPassword('密码: ');

    const { token, user } = await post('/auth/login', {
      body: { email, password },
    });
    save({ token });
    out({ user, token_saved: true });
  });

// popiart auth logout
authCmd
  .command('logout')
  .description('撤销当前会话令牌')
  .action(async () => {
    const { token } = load();
    if (!token) { out({ logged_out: true, was_authenticated: false }); return; }

    await post('/auth/logout', { body: {} }).catch(() => {});
    save({ token: null });
    out({ logged_out: true });
  });

// popiart auth whoami
authCmd
  .command('whoami')
  .description('显示当前已验证的用户')
  .action(async () => {
    const me = await get('/auth/me');
    out(me);
  });

// popiart auth token show | set | rotate
const tokenCmd = authCmd.command('token').description('管理 API 令牌');

tokenCmd
  .command('show')
  .description('打印存储的令牌（已脱敏）')
  .action(() => {
    const { token } = load();
    if (!token) err('UNAUTHENTICATED', '未存储令牌。请运行: popiart auth login');
    const masked = token.slice(0, 8) + '•'.repeat(Math.max(0, token.length - 12)) + token.slice(-4);
    out({ token: masked, config: configPath() });
  });

tokenCmd
  .command('set <token>')
  .description('存储令牌而不进行验证')
  .action((token) => {
    save({ token });
    out({ token_saved: true });
  });

tokenCmd
  .command('rotate')
  .description('签发新令牌并撤销旧令牌')
  .action(async () => {
    const { token: newToken } = await post('/auth/token/rotate', { body: {} });
    save({ token: newToken });
    out({ token_rotated: true });
  });

// --- 助手函数 ---
function prompt(question) {
  return new Promise((resolve) => {
    const rl = createInterface({ input: process.stdin, output: process.stderr });
    rl.question(question, (answer) => { rl.close(); resolve(answer); });
  });
}

function promptPassword(question) {
  return new Promise((resolve) => {
    process.stderr.write(question);
    const rl = createInterface({ input: process.stdin, output: process.stderr });
    // 禁用回显（在 TTY 上尽力而为）
    if (process.stdin.isTTY) process.stdin.setRawMode?.(true);
    let pw = '';
    process.stdin.once('data', (chunk) => {
      pw = chunk.toString().trim();
      if (process.stdin.isTTY) process.stdin.setRawMode?.(false);
      process.stderr.write('\n');
      rl.close();
      resolve(pw);
    });
  });
}
