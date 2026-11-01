// Renders the updates and backup/restore screens: the GL admin panel view
// (mock GL shell, as glinet-screens.mjs) and the agent's own dashboard
// (agent/web, API answered from fixtures). See updates-backup-screens.sh.
// Usage (inside the Playwright image): node updates-backup-screens.mjs <repo> <outdir>
import { createRequire } from 'node:module';
import { readFileSync, mkdirSync } from 'node:fs';
import { clock, glStates, glActions, glPhone, webStates, webActions, webPhone } from './glinet/ub-fixtures.mjs';
const require = createRequire(import.meta.url);
const { chromium } = require('playwright');

const [repo, out] = process.argv.slice(2);
for (const d of ['glinet', 'web']) mkdirSync(`${out}/${d}`, { recursive: true });
const glFiles = {
  '/shell.html': [readFileSync(`${repo}/scripts/ux/glinet/shell.html`), 'text/html'],
  '/views/gl-sdk4-ui-tailbond.common.js': [readFileSync(`${repo}/packaging/openwrt/gl-ui/www/views/gl-sdk4-ui-tailbond.common.js`), 'application/javascript'],
  '/vendor/vue.runtime.min.js': [readFileSync(require.resolve('vue/dist/vue.runtime.min.js')), 'application/javascript'],
};
const webDir = `${repo}/agent/web`;
const webTypes = { html: 'text/html', js: 'application/javascript', css: 'text/css', svg: 'image/svg+xml' };
const viewports = {
  desktop: { viewport: { width: 1280, height: 800 }, deviceScaleFactor: 1 },
  phone: { viewport: { width: 390, height: 844 }, deviceScaleFactor: 2, isMobile: true, hasTouch: true },
};
const fakeFile = ([name, size]) => ({ name, mimeType: 'application/octet-stream', buffer: Buffer.alloc(Math.round(size), 7) });
const fakeBackup = Buffer.from('TBBACKUP1\n{"v":1}\nAAAA\n');

async function steps(page, list) {
  for (const s of list || []) {
    if (s.text) await page.getByText(s.text).first().click();
    if (s.click) await page.click(s.click);
    if (s.button) await page.getByRole('button', { name: s.button, exact: true }).first().click();
    if (s.fill) await page.fill(s.fill[0], s.fill[1]);
    if (s.files) await page.setInputFiles(s.files[0], s.files[1].map(fakeFile));
    if (s.wait) await page.waitForTimeout(s.wait);
    await page.waitForTimeout(300);
  }
}

// answers an /api/... call from a fixture map {"METHOD /api/path": value}
function answer(route, apiMap, method, path) {
  const key = `${method} ${path}`;
  if (method === 'POST' && path === '/api/backup') {
    return route.fulfill({ status: 200, body: fakeBackup, headers: { 'Content-Type': 'application/octet-stream', 'Content-Disposition': 'attachment; filename="backup.tbbackup"' } });
  }
  let v = apiMap[key];
  if (v === undefined && method !== 'GET') v = { ok: true };
  if (v === undefined) return route.fulfill({ status: 404, contentType: 'application/json', body: '{"ok":false,"error":"not found"}' });
  return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(v) });
}

const browser = await chromium.launch();

async function gl(name, fx, vp, act) {
  const ctx = await browser.newContext({ ...viewports[vp], locale: 'en-US', timezoneId: 'UTC', acceptDownloads: true });
  const page = await ctx.newPage();
  await page.route('http://gl.test/**', (route) => {
    const req = route.request();
    const u = new URL(req.url());
    if (u.pathname.startsWith('/tailbond-api/')) return answer(route, fx.api || {}, req.method(), '/api/' + u.pathname.slice('/tailbond-api/'.length));
    const f = glFiles[u.pathname];
    if (!f) return route.fulfill({ status: 404, body: '' });
    return route.fulfill({ status: 200, body: f[0], contentType: f[1] });
  });
  await page.addInitScript((f) => { window.__TB = f; }, fx);
  if (fx.clock) await page.clock.setFixedTime(new Date(fx.clock));
  await page.goto('http://gl.test/shell.html');
  await page.waitForFunction(() => window.__mounted === true);
  await page.waitForTimeout(900);
  await steps(page, act && act.steps);
  // a dialog is shot at window size, a page whole; the cards sit at the bottom
  const full = act ? !!act.full : true;
  if (!act) await page.evaluate(() => { const c = document.querySelector('.tb-upd') || document.querySelector('.tb-bk'); if (c) c.scrollIntoView(); });
  await page.screenshot({ path: `${out}/glinet/${name}-${vp}.png`, fullPage: full });
  await ctx.close();
}

async function web(name, fx, vp, act) {
  const ctx = await browser.newContext({ ...viewports[vp], locale: 'en-US', timezoneId: 'UTC', acceptDownloads: true });
  const page = await ctx.newPage();
  if (process.env.TB_UX_DEBUG) { page.on('console', (m) => console.log('console:', m.text())); page.on('pageerror', (e) => console.log('pageerror:', e.message)); }
  await page.route('http://homebox.test/**', (route) => {
    const req = route.request();
    const p = new URL(req.url()).pathname;
    if (p === '/api/status') return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(fx.status) });
    if (p === '/api/events') return route.fulfill({ status: 200, contentType: 'text/event-stream', body: `data: ${JSON.stringify(fx.status)}\n\n` });
    if (p === '/api/settings') return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(fx.settings) });
    if (p === '/api/health') return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ all_good: true, checks: [] }) });
    if (p === '/api/logs') return route.fulfill({ status: 200, contentType: 'application/json', body: '{"entries":[]}' });
    if (p.startsWith('/api/')) return answer(route, fx.api || {}, req.method(), p);
    const f = p === '/' ? 'index.html' : p.slice(1);
    try {
      return route.fulfill({ status: 200, body: readFileSync(`${webDir}/${f}`), contentType: webTypes[f.split('.').pop()] || 'application/octet-stream' });
    } catch { return route.fulfill({ status: 404, body: '' }); }
  });
  await page.clock.setFixedTime(new Date(clock));
  await page.goto('http://homebox.test/');
  await page.waitForSelector('#tabs', { state: 'attached' });
  await page.waitForTimeout(800);
  // a phone opens on "Add this phone"; the owner reveals the dashboard
  if (vp === 'phone') { const b = await page.$('#show-dashboard'); if (b && await b.isVisible()) await b.click(); }
  await page.waitForSelector('#tabs');
  await page.click('#tabs button[data-tab="maint"]');
  await page.waitForSelector('#upd-card .ub-head');
  await page.waitForTimeout(500);
  await steps(page, act && act.steps);
  const full = act ? !!act.full : true;
  await page.screenshot({ path: `${out}/web/${name}-${vp}.png`, fullPage: full });
  await ctx.close();
}

// TB_UX_ONLY=<regex>: only the screens whose name matches (debugging)
const only = process.env.TB_UX_ONLY ? new RegExp(process.env.TB_UX_ONLY) : null;
const want = (name, vp, phoneList) => (!only || only.test(name)) && (vp === 'desktop' || phoneList.includes(name));
const withActApi = (fx, act) => ({ ...fx, api: { ...fx.api, ...(act.api || {}) } });
for (const vp of Object.keys(viewports)) {
  for (const [name, fx] of Object.entries(glStates)) if (want(name, vp, glPhone)) await gl(name, fx, vp);
  for (const [name, act] of Object.entries(glActions)) if (want(name, vp, glPhone)) await gl(name, withActApi(glStates[act.base], act), vp, act);
  for (const [name, fx] of Object.entries(webStates)) if (want(name, vp, webPhone)) await web(name, fx, vp);
  for (const [name, act] of Object.entries(webActions)) if (want(name, vp, webPhone)) await web(name, withActApi(webStates[act.base], act), vp, act);
}
await browser.close();
console.log('rendered to', out);
