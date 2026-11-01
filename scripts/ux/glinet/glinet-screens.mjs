// Renders the tailbond GL admin panel in a mock GL 4.x shell, every state
// in fixtures.mjs, desktop (1280x800) and phone (390x844), full page.
// Usage (inside the Playwright image): node glinet-screens.mjs <repo> <outdir> [set]
// set "bypass-failover": only the bypass rules / more servers screens
// (bfStates, bfActions; desktop, plus phone for bfPhone).
import { createRequire } from 'node:module';
import { readFileSync, mkdirSync } from 'node:fs';
import { states, actions, bfStates, bfActions, bfPhone, landing, fedoraStatus, fedoraStatusApproved, fedoraStatusFresh, fedoraStatusPhone, fedoraStatusPhoneDone, fedoraHealth, fedoraHealthGood, fedoraHealthRoute } from './fixtures.mjs';
const require = createRequire(import.meta.url);
const { chromium } = require('playwright');

const [repo, out, set] = process.argv.slice(2);
mkdirSync(out, { recursive: true });
const files = {
  '/shell.html': [readFileSync(`${repo}/scripts/ux/glinet/shell.html`), 'text/html'],
  '/luci.html': [readFileSync(`${repo}/scripts/ux/glinet/luci.html`), 'text/html'],
  '/gl-advanced.html': [readFileSync(`${repo}/scripts/ux/glinet/gl-advanced.html`), 'text/html'],
  '/views/gl-sdk4-ui-tailbond.common.js': [readFileSync(`${repo}/packaging/openwrt/gl-ui/www/views/gl-sdk4-ui-tailbond.common.js`), 'application/javascript'],
  '/vendor/vue.runtime.min.js': [readFileSync(require.resolve('vue/dist/vue.runtime.min.js')), 'application/javascript'],
};
const viewports = {
  desktop: { viewport: { width: 1280, height: 800 }, deviceScaleFactor: 1 },
  phone: { viewport: { width: 390, height: 844 }, deviceScaleFactor: 2, isMobile: true, hasTouch: true },
};

const browser = await chromium.launch();
async function shot(name, fixture, vp, act, page0) {
  if (process.env.TB_UX_DEBUG) console.log('shot', name, vp);
  const ctx = await browser.newContext({ ...viewports[vp], locale: 'en-US', timezoneId: 'UTC' });
  const page = await ctx.newPage();
  await page.route('http://gl.test/**', (route) => {
    const p = new URL(route.request().url()).pathname;
    const f = files[p];
    if (!f) return route.fulfill({ status: 404, body: '' });
    return route.fulfill({ status: 200, body: f[0], contentType: f[1] });
  });
  await page.addInitScript((fx) => { window.__TB = fx; }, fixture);
  await page.goto(`http://gl.test/${page0 || 'shell.html'}`);
  if (!page0) {
    await page.waitForFunction(() => window.__mounted === true);
    await page.waitForTimeout(700);
    if (fixture.settle) await page.waitForTimeout(fixture.settle); // let a status sequence play
    if (act && act.click) { await page.click(act.click); await page.waitForTimeout(300); }
    if (act && act.clickText) { await page.getByText(act.clickText).first().click(); await page.waitForTimeout(300); }
    for (const s of (act && act.steps) || []) {
      if (s.text) await page.getByText(s.text).first().click();
      if (s.click) await page.click(s.click);
      if (s.button) await page.getByRole('button', { name: s.button, exact: true }).first().click();
      if (s.fill) await page.fill(s.fill[0], s.fill[1]);
      if (s.wait) await page.waitForTimeout(s.wait);
      await page.waitForTimeout(300);
    }
  }
  // a dialog (modal) is shot at window size, a page or open panel whole
  const full = act && act.full !== undefined ? act.full : !(act && (act.click || act.steps));
  await page.screenshot({ path: `${out}/${name}-${vp}.png`, fullPage: full });
  await ctx.close();
}

// Fedora home server: the agent's own web UI (agent/web), API answered
// from fixtures. One variant per situation the Fedora persona looks at.
const webDir = `${repo}/agent/web`;
const webTypes = { html: 'text/html', js: 'application/javascript', css: 'text/css', svg: 'image/svg+xml' };
const fedoraSettings = { role: 'server', mode: 'speed', public_addr: 'home.example.net', effective_public_addr: 'home.example.net:443', port: 443, subnet: '10.77.0.0/24', tls: 'tailscale', tls_name: 'homebox.tail1234.ts.net', version: '0.1.0-r42', glinet_ipk: true, password_set: false, ...landing };
const fedoraTs = { installed: true, running: true, self: 'homebox', dns_name: 'homebox.tail1234.ts.net', tailnet: 'alex@example.com', ips: ['100.101.1.1', 'fd7a:115c:a1e0::1'], route: '10.77.0.0/24', advertised_routes: ['10.77.0.0/24'], approved_routes: ['10.77.0.0/24'], route_advertised: true, routes_approved: true, admin_url: 'https://login.tailscale.com/admin/machines', peers: 4, peers_online: 3 };
async function fedora(vp, name, { status = fedoraStatus, after = null, settings = {}, ts = {}, health = fedoraHealth, tab = null, click = null, full = false, waitFor = null, settle = 1500 } = {}) {
  const ctx = await browser.newContext({ ...viewports[vp], locale: 'en-US', timezoneId: 'UTC' });
  const page = await ctx.newPage();
  let st = status;
  await page.route('http://homebox.test/**', (route) => {
    const req = route.request();
    const p = new URL(req.url()).pathname;
    const json = (v) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(v) });
    if (req.method() === 'POST' && p.startsWith('/api/pair-requests/')) { if (after) st = after; return json({ ok: true }); }
    if (p === '/api/status') return json(st);
    if (p === '/api/settings') return json({ ...fedoraSettings, ...settings });
    if (p === '/api/tailscale') return json({ ...fedoraTs, ...ts });
    if (p === '/api/health') return json(health);
    if (p === '/api/logs') return json({ entries: [] });
    if (p.startsWith('/api/')) return route.fulfill({ status: 404, contentType: 'application/json', body: '{"ok":false}' });
    const f = p === '/' ? 'index.html' : p.slice(1);
    try {
      return route.fulfill({ status: 200, body: readFileSync(`${webDir}/${f}`), contentType: webTypes[f.split('.').pop()] || 'application/octet-stream' });
    } catch { return route.fulfill({ status: 404, body: '' }); }
  });
  await page.goto('http://homebox.test/');
  await page.waitForSelector(waitFor || '#tabs');
  await page.waitForTimeout(600);
  if (tab) {
    if (vp === 'phone') { const b = await page.$('#show-dashboard'); if (b && await b.isVisible()) await b.click(); }
    await page.click(`#tabs button[data-tab="${tab}"]`);
    await page.waitForTimeout(700);
  }
  if (click) { await page.getByRole('button', { name: click }).first().click(); await page.waitForTimeout(settle); }
  await page.screenshot({ path: `${out}/${name}-${vp}.png`, fullPage: full });
  await ctx.close();
}

async function fedoraAll(vp) {
  await fedora(vp, '14-fedora-dashboard-request', { waitFor: '#pair-requests:not(.hidden)', full: true });
  if (vp !== 'desktop') return;
  await fedora(vp, '14b-fedora-after-approve', { waitFor: '#pair-requests:not(.hidden)', click: 'Approve', after: fedoraStatusApproved });
  await fedora(vp, '14c-fedora-phone-request', { status: fedoraStatusPhone, waitFor: '#pair-requests:not(.hidden)' });
  await fedora(vp, '14d-fedora-phone-approved', { status: fedoraStatusPhone, waitFor: '#pair-requests:not(.hidden)', click: 'Approve', after: fedoraStatusPhoneDone, full: true, settle: 4000 });
  await fedora(vp, '15-fedora-devices-tab', { tab: 'clients', full: true });
  await fedora(vp, '15b-fedora-tailscale-tab', { tab: 'tailscale', full: true });
  await fedora(vp, '15c-fedora-fresh-install', { status: fedoraStatusFresh, settings: { public_addr: '', effective_public_addr: '192.168.1.2:443', public_addr_guessed: true, tls: 'self-signed' },
    ts: { routes_approved: false, approved_routes: [] }, health: fedoraHealthRoute, tab: 'clients', full: true });
  await fedora(vp, '15d-fedora-health-all-good', { status: fedoraStatusApproved, health: fedoraHealthGood, tab: 'clients', full: true });
}

const withActApi = (fx, act) => ({ ...fx, api: { ...fx.api, ...(act.api || {}) }, ...(act.after ? { after: act.after } : {}) });
if (set === 'bypass-failover') {
  for (const vp of Object.keys(viewports)) {
    for (const [name, fx] of Object.entries(bfStates)) if (vp === 'desktop' || bfPhone.includes(name)) await shot(name, fx, vp);
    for (const [name, act] of Object.entries(bfActions)) if (vp === 'desktop' || bfPhone.includes(name)) await shot(name, withActApi(bfStates[act.base], act), vp, act);
  }
  await browser.close();
  console.log('rendered to', out);
  process.exit(0);
}

for (const vp of Object.keys(viewports)) {
  await fedoraAll(vp);
  await shot('01a-install-gl-advanced-settings', {}, vp, null, 'gl-advanced.html');
  await shot('01b-install-luci-upload', {}, vp, null, 'luci.html');
  for (const [name, fx] of Object.entries(states)) await shot(name, fx, vp);
  for (const [name, act] of Object.entries(actions)) if (act) await shot(name, states[act.base], vp, act);
}
await browser.close();
console.log('rendered to', out);
