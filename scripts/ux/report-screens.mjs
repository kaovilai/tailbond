// Report a problem in a browser against a real tailbondd; see report-screens.sh.
// Usage (inside the Playwright image): node report-screens.mjs <repo> <outdir> <agent-url>
import { createRequire } from 'node:module';
import { readFileSync } from 'node:fs';
const require = createRequire(import.meta.url);
const { chromium } = require('playwright');

const [repo, out, agent] = process.argv.slice(2);
const SECRETS = ['hunter2', '203.0.113.7', '198.51.100.9', 'sam@example.com', '2001:db8:1::42', 'tail1234'];
const ISSUE = 'https://github.com/kaovilai/tailbond/issues/new?template=bug_report.yml&';
let failed = 0;
const ok = (c, msg) => { console.log(`  ${c ? 'PASS' : 'FAIL'} ${msg}`); if (!c) failed++; };
const clean = (what, s) => {
  const bad = SECRETS.filter((x) => s.includes(x) || s.includes(encodeURIComponent(x)));
  ok(bad.length === 0, `${what}: nothing secret${bad.length ? ' (found ' + bad.join(', ') + ')' : ''}`);
};
// window.open is captured: nothing is opened
const captureOpen = () => {
  window.__opened = [];
  window.open = () => ({ opener: null, close() {}, location: { set href(u) { window.__opened.push(u); } } });
};
const checkURL = (where, u) => {
  ok(!!u && u.startsWith(ISSUE), `${where}: the issue URL is GitHub's bug report form`);
  ok(!!u && u.length <= 7000, `${where}: URL is ${u ? u.length : 0} characters (≤ 7000)`);
  ok(!!u && u.includes('what=The%20tunnel%20drops'), `${where}: the URL has what the user typed`);
  clean(`${where} URL`, u || '');
};

const browser = await chromium.launch();
const vps = {
  desktop: { viewport: { width: 1280, height: 800 }, deviceScaleFactor: 1 },
  phone: { viewport: { width: 412, height: 915 }, deviceScaleFactor: 2, isMobile: true, hasTouch: true },
};

// ---- the dashboard ----
for (const [vp, opts] of Object.entries(vps)) {
  console.log(`== dashboard (${vp})`);
  const ctx = await browser.newContext({ ...opts, acceptDownloads: true, locale: 'en-US', timezoneId: 'UTC' });
  await ctx.grantPermissions(['clipboard-read', 'clipboard-write'], { origin: agent });
  const page = await ctx.newPage();
  await page.addInitScript(captureOpen);
  await page.goto(agent + '/');
  await page.waitForSelector('#crash-banner:not(.hidden)', { timeout: 15000 }).catch(() => {});
  ok(await page.isVisible('#crash-banner'), 'crash banner shows after a crash');
  await page.screenshot({ path: `${out}/report-01-crash-banner-${vp}.png` });
  // desktop: the footer's link; phone (the server page opens on "Add this
  // phone" there): the crash banner's own link
  await page.click(vp === 'desktop' ? '#report-link' : '#crash-banner a[href="#report"]');
  await page.waitForSelector('#rep-what');
  await page.waitForFunction(() => document.querySelector('#rep-logs').value.includes('mqvpn') || document.querySelector('#rep-logs').value.includes('path wifi'));
  const notice = await page.textContent('#rep-notice');
  ok(notice.startsWith('Once you submit this issue on GitHub, it is public.') && notice.includes('Nothing is sent automatically'), 'the notice is shown, "public" first');
  const logs = await page.inputValue('#rep-logs');
  const crash = await page.inputValue('#rep-crash');
  ok(logs.includes('[public IPv4]:443') && logs.includes('token=[redacted]'), 'logs are shown redacted');
  ok(crash.includes('panic: send on closed channel') && crash.includes('key=[redacted]'), 'crash details are shown redacted');
  clean('preview', logs + crash);
  ok(await page.isDisabled('#rep-open'), 'Open GitHub issue is off at first');
  ok((await page.textContent('#rep-why')).includes('say what happened'), 'it says why the button is off');
  ok((await page.textContent('#rep-account')).includes('GitHub account'), 'it says a GitHub account is needed, and what to do without one');
  ok(await page.$eval('#rep-logs', (e) => getComputedStyle(e).whiteSpace) === 'pre-wrap', 'long log lines wrap (readable without scrolling sideways)');
  await page.screenshot({ path: `${out}/report-02-preview-${vp}.png` });
  await page.fill('#rep-what', 'The tunnel drops when I plug in the iPhone; token=hunter2');
  ok(await page.isDisabled('#rep-open'), 'still off until the checkbox is ticked');
  await page.check('#rep-ok');
  ok(await page.isEnabled('#rep-open'), 'on with the text and the checkbox');
  await page.screenshot({ path: `${out}/report-03-ready-${vp}.png` });
  await page.click('#rep-open');
  await page.waitForFunction(() => window.__opened.length > 0);
  checkURL('dashboard', (await page.evaluate(() => window.__opened))[0]);
  const [dl] = await Promise.all([page.waitForEvent('download'), page.click('#rep-save')]);
  const file = readFileSync(await dl.path(), 'utf8');
  ok(/^tailbond-report-\d{8}-\d{6}\.txt$/.test(dl.suggestedFilename()), `Save report file: ${dl.suggestedFilename()}`);
  ok(file.includes('### Logs') && file.includes('--- crash ---'), 'the file is the full report');
  clean('report file', file);
  await page.click('#rep-copy');
  await page.waitForTimeout(500);
  const clip = await page.evaluate(() => navigator.clipboard.readText());
  ok(clip === file, 'Copy full report copies the same text');
  // the Mac menu's "Report a problem…" opens /#report: the form, not just the dashboard
  const p2 = await ctx.newPage();
  await p2.goto(agent + '/#report');
  ok(await p2.waitForSelector('#rep-what', { timeout: 10000 }).then(() => true, () => false), '/#report opens the report form directly (the menu bar link)');
  await ctx.close();
}

// ---- the GL.iNet panel (the real view in the mock shell, a real agent behind it) ----
const files = {
  '/shell.html': [readFileSync(`${repo}/scripts/ux/glinet/shell.html`), 'text/html'],
  '/views/gl-sdk4-ui-tailbond.common.js': [readFileSync(`${repo}/packaging/openwrt/gl-ui/www/views/gl-sdk4-ui-tailbond.common.js`), 'application/javascript'],
  '/vendor/vue.runtime.min.js': [readFileSync(require.resolve('vue/dist/vue.runtime.min.js')), 'application/javascript'],
};
for (const [vp, opts] of Object.entries(vps)) {
  console.log(`== GL.iNet panel (${vp})`);
  const ctx = await browser.newContext({ ...opts, acceptDownloads: true, locale: 'en-US', timezoneId: 'UTC' });
  const page = await ctx.newPage();
  await page.route('http://gl.test/**', async (route) => {
    const u = new URL(route.request().url());
    if (u.pathname.startsWith('/live/')) {
      const r = route.request();
      const res = await route.fetch({ url: agent + u.pathname.slice(5) + u.search, headers: { 'content-type': 'application/json' }, method: r.method(), postData: r.postData() || undefined });
      return route.fulfill({ response: res });
    }
    const f = files[u.pathname];
    if (!f) return route.fulfill({ status: 404, body: '' });
    return route.fulfill({ status: 200, body: f[0], contentType: f[1] });
  });
  await page.addInitScript((fx) => { window.__TB = fx; }, { live: true, role: 'server', model: 'GL-BE9300' });
  await page.addInitScript(captureOpen);
  await page.goto('http://gl.test/shell.html');
  await page.waitForFunction(() => window.__mounted === true);
  await page.getByText('tailbond stopped unexpectedly').first().waitFor({ timeout: 15000 }).catch(() => {});
  ok(await page.getByText('tailbond stopped unexpectedly').first().isVisible(), 'GL: crash alert shows');
  await page.screenshot({ path: `${out}/report-11-gl-crash-${vp}.png`, fullPage: true });
  await page.click('#tb-report-link');
  await page.getByRole('dialog', { name: 'Report a problem' }).waitFor();
  await page.waitForTimeout(500);
  const dlg = page.getByRole('dialog', { name: 'Report a problem' });
  const text = await dlg.evaluate((d) => Array.from(d.querySelectorAll('textarea,input')).map((e) => e.value).join('\n'));
  ok(text.includes('[public IPv4]') && text.includes('[redacted]'), 'GL: preview is redacted');
  clean('GL preview', text);
  const open = dlg.getByRole('button', { name: 'Open GitHub issue' });
  ok(await open.isDisabled(), 'GL: Open GitHub issue is off at first');
  ok(await page.isVisible('#tb-rep-why'), 'GL: it says why the button is off');
  await page.screenshot({ path: `${out}/report-12-gl-preview-${vp}.png` });
  await dlg.locator('textarea').first().fill('The tunnel drops when I plug in the iPhone');
  ok(await open.isDisabled(), 'GL: still off until the checkbox is ticked');
  await page.click('#tb-rep-ok');
  ok(await open.isEnabled(), 'GL: on with the text and the checkbox');
  await page.screenshot({ path: `${out}/report-13-gl-ready-${vp}.png` });
  await open.click();
  await page.waitForFunction(() => window.__opened.length > 0);
  checkURL('GL panel', (await page.evaluate(() => window.__opened))[0]);
  const [dl] = await Promise.all([page.waitForEvent('download'), dlg.getByRole('button', { name: 'Save report file' }).click()]);
  clean('GL report file', readFileSync(await dl.path(), 'utf8'));
  await ctx.close();
}

await browser.close();
console.log(failed ? `RESULT: FAIL (${failed})` : 'RESULT: PASS');
process.exit(failed ? 1 : 0);
