#!/usr/bin/env node
// Browser driver for the RouteWise web app started by stack.sh (playwright-core + installed Chrome).
//
//   node drive.mjs shot <route> [out.png] [--auth] [--lang en|he]
//   node drive.mjs login [out.png] [--lang en|he]         real UI login with a freshly registered user
//   node drive.mjs flow <flow.mjs> [--auth] [--lang en|he] run `export default async ({page, shot, api, user}) => {}`
//
// --auth registers a fresh org user via the API and injects its token into localStorage (skips the login UI).
// Prints a JSON summary (url, title, headings, console errors, failed API calls, screenshot paths).
import { chromium } from 'playwright-core';
import path from 'node:path';
import fs from 'node:fs';
import { pathToFileURL } from 'node:url';

const WEB = process.env.RW_WEB_URL || `http://localhost:${process.env.RW_WEB_PORT || 3100}`;
const API = process.env.RW_API_URL || `http://localhost:${process.env.RW_API_PORT || 18080}`;
const OUT = process.env.RW_SHOTS || path.join(process.env.TMPDIR || '/tmp', 'routewise-run', 'shots');
fs.mkdirSync(OUT, { recursive: true });

const argv = process.argv.slice(2);
const flag = (name) => argv.includes(name);
const opt = (name, dflt) => { const i = argv.indexOf(name); return i >= 0 ? argv[i + 1] : dflt; };
const positional = argv.filter((a, i) => !a.startsWith('--') && !(argv[i - 1] || '').startsWith('--lang'));
const [cmd, ...rest] = positional;
const lang = opt('--lang', 'en');

async function api(method, p, body, token) {
  const res = await fetch(API + p, {
    method,
    headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
    body: body ? JSON.stringify(body) : undefined,
  });
  const text = await res.text();
  let json; try { json = JSON.parse(text); } catch { json = text; }
  if (!res.ok) throw new Error(`${method} ${p} -> ${res.status}: ${text}`);
  return json;
}

async function registerUser() {
  const n = `${Date.now()}${Math.floor(Math.random() * 1000)}`;
  const creds = { email: `agent${n}@example.com`, password: 'password123' };
  const r = await api('POST', '/api/v1/register', {
    ...creds, name: 'Agent Tester', phone: '+972500000000', company_name: `Agent Co ${n}`, industry: 'plumbing',
  });
  return { ...creds, token: r.token, user: r.user, organization: r.organization };
}

const resolveOut = (p, dflt) => (p ? path.resolve(p) : path.join(OUT, dflt));

async function main() {
  if (!cmd || !['shot', 'login', 'flow'].includes(cmd)) {
    console.error(fs.readFileSync(new URL(import.meta.url)).toString().split('\n').slice(1, 9).join('\n'));
    process.exit(2);
  }
  const user = (flag('--auth') || cmd === 'login') ? await registerUser() : null;

  const browser = await chromium.launch({ channel: process.env.RW_BROWSER_CHANNEL || 'chrome', headless: true });
  const ctx = await browser.newContext({ viewport: { width: 1400, height: 900 } });
  const consoleErrors = [], failedApi = [], shots = [];
  await ctx.addInitScript(([l, u, auth]) => {
    // Only on first load of a fresh context, so later logout/login in a flow is not undone.
    if (sessionStorage.getItem('__rw_init')) return;
    sessionStorage.setItem('__rw_init', '1');
    localStorage.setItem('language', l);
    if (auth && u) { localStorage.setItem('token', u.token); localStorage.setItem('user', JSON.stringify(u.user)); }
  }, [lang, user, flag('--auth')]);
  const page = await ctx.newPage();
  page.on('console', (m) => m.type() === 'error' && consoleErrors.push(m.text().slice(0, 300)));
  page.on('pageerror', (e) => consoleErrors.push(`pageerror: ${e.message.slice(0, 300)}`));
  page.on('response', (r) => r.url().startsWith(API) && r.status() >= 400 && failedApi.push(`${r.status()} ${r.request().method()} ${r.url()}`));

  const shot = async (file) => {
    await page.waitForLoadState('networkidle').catch(() => {});
    const p = resolveOut(file && (file.includes('/') ? file : path.join(OUT, file)), `shot-${Date.now()}.png`);
    await page.screenshot({ path: p, fullPage: true });
    shots.push(p);
    return p;
  };

  try {
    if (cmd === 'shot') {
      await page.goto(WEB + (rest[0] || '/'), { waitUntil: 'networkidle' });
      await shot(rest[1] || `${(rest[0] || 'root').replace(/[^a-z0-9]+/gi, '_').replace(/^_|_$/g, '') || 'root'}.png`);
    } else if (cmd === 'login') {
      await page.goto(WEB + '/login', { waitUntil: 'networkidle' });
      await page.fill('input[name="email"]', user.email);
      await page.fill('input[name="password"]', user.password);
      await page.click('form button[type="submit"]');
      await page.waitForURL('**/dashboard', { timeout: 15000 });
      await shot(rest[0] || 'dashboard-after-login.png');
    } else if (cmd === 'flow') {
      const mod = await import(pathToFileURL(path.resolve(rest[0])).href);
      await mod.default({ page, shot, user, web: WEB, api: (m, p, b) => api(m, p, b, user?.token) });
    }
  } finally {
    const headings = await page.locator('h1, h2').allInnerTexts().catch(() => []);
    console.log(JSON.stringify({
      url: page.url(), title: await page.title().catch(() => ''), headings: headings.slice(0, 8),
      user: user && { email: user.email, password: user.password, org_id: user.organization?.id },
      shots, consoleErrors, failedApi,
    }, null, 2));
    await browser.close();
  }
}

main().catch((e) => { console.error(e.stack || String(e)); process.exit(1); });
