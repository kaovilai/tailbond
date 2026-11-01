// Renders the tailbond GL admin panel in a mock GL 4.x shell, every state
// in fixtures.mjs, desktop (1280x800) and phone (390x844), full page.
// Usage (inside the Playwright image): node glinet-screens.mjs <repo> <outdir>
import { createRequire } from 'node:module';
import { readFileSync, mkdirSync } from 'node:fs';
import { states, actions, fedoraStatus } from './fixtures.mjs';
const require = createRequire(import.meta.url);
const { chromium } = require('playwright');

const [repo, out] = process.argv.slice(2);
mkdirSync(out, { recursive: true });
const files = {
  '/shell.html': [readFileSync(`${repo}/scripts/ux/glinet/shell.html`), 'text/html'],
  '/luci.html': [readFileSync(`${repo}/scripts/ux/glinet/luci.html`), 'text/html'],
  '/views/gl-sdk4-ui-tailbond.common.js': [readFileSync(`${repo}/packaging/openwrt/gl-ui/www/views/gl-sdk4-ui-tailbond.common.js`), 'application/javascript'],
  '/vendor/vue.runtime.min.js': [readFileSync(require.resolve('vue/dist/vue.runtime.min.js')), 'application/javascript'],
};
const viewports = {
  desktop: { viewport: { width: 1280, height: 800 }, deviceScaleFactor: 1 },
  phone: { viewport: { width: 390, height: 844 }, deviceScaleFactor: 2, isMobile: true, hasTouch: true },
};

const browser = await chromium.launch();
async function shot(name, fixture, vp, act, page0) {
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
    if (act && act.click) { await page.click(act.click); await page.waitForTimeout(300); }
    if (act && act.clickText) { await page.getByText(act.clickText).first().click(); await page.waitForTimeout(300); }
  }
  await page.screenshot({ path: `${out}/${name}-${vp}.png`, fullPage: !(act && act.click) });
  await ctx.close();
}

// Fedora home server: the agent's own web UI (agent/web), API answered
// from fixtures.
const webDir = `${repo}/agent/web`;
const webTypes = { html: 'text/html', js: 'application/javascript', css: 'text/css', svg: 'image/svg+xml' };
async function fedora(vp) {
  const ctx = await browser.newContext({ ...viewports[vp], locale: 'en-US', timezoneId: 'UTC' });
  const page = await ctx.newPage();
  await page.route('http://homebox.test/**', (route) => {
    const p = new URL(route.request().url()).pathname;
    if (p === '/api/status') return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(fedoraStatus) });
    if (p === '/api/settings') return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ role: 'server', effective_public_addr: 'home.example.net:443', version: '0.1.0-r42' }) });
    if (p === '/api/logs') return route.fulfill({ status: 200, contentType: 'application/json', body: '{"entries":[]}' });
    if (p.startsWith('/api/')) return route.fulfill({ status: 404, contentType: 'application/json', body: '{"ok":false}' });
    const f = p === '/' ? 'index.html' : p.slice(1);
    try {
      return route.fulfill({ status: 200, body: readFileSync(`${webDir}/${f}`), contentType: webTypes[f.split('.').pop()] || 'application/octet-stream' });
    } catch { return route.fulfill({ status: 404, body: '' }); }
  });
  await page.goto('http://homebox.test/');
  await page.waitForSelector('#pair-requests:not(.hidden)');
  await page.waitForTimeout(500);
  await page.screenshot({ path: `${out}/14-fedora-dashboard-request-${vp}.png` });
  await ctx.close();
}

for (const vp of Object.keys(viewports)) {
  await fedora(vp);
  await shot('01-not-installed-luci-upload', {}, vp, null, 'luci.html');
  for (const [name, fx] of Object.entries(states)) await shot(name, fx, vp);
  for (const [name, act] of Object.entries(actions)) if (act) await shot(name, states[act.base], vp, act);
}
await browser.close();
console.log('rendered to', out);
