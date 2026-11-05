// Accessibility run over every web surface tailbond ships; see a11y.sh.
// Usage (inside the Playwright image):
//   node a11y.mjs <repo> <outdir> <dashboard-url> <phone-page-url> <client-url>
//
// Each page (and each state worth checking: a dialog open, a tab switched,
// an error shown) gets axe-core plus two checks axe doesn't make:
//   tb-pointer-only   something that looks clickable (cursor: pointer) that
//                     a keyboard can't reach (not a link, button, form field,
//                     label or tabindex >= 0, nor inside one);
//   tb-motion         an endless animation still running with "reduce
//                     motion" on (every page is opened that way);
//   tb-no-key-handler a control that isn't a native one (role checkbox,
//                     radio, switch, button… on a div or span) that Tab
//                     reaches but no key handler answers (Chromium's
//                     DevTools protocol lists its listeners).
// A self-test page with one of each known problem runs first: if the run
// stops seeing them, it fails rather than passing everything.
// Fails on impact serious or critical (allow-listed ones aside); moderate
// and minor are reported only. A rule turned off for a page says why next
// to it (`off:`); the allow-list (a11y-allow.json) needs a reason and a link.
import { createRequire } from 'node:module';
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { states, actions, shareStates, bfStates, bfActions, bfPhone, landing as landingSettings,
  fedoraStatus, fedoraStatusFresh, fedoraHealth, fedoraHealthRoute } from './glinet/fixtures.mjs';
import { clock as ubClock, glStates, glActions, glPhone, webStates, webActions, webPhone } from './glinet/ub-fixtures.mjs';
const require = createRequire(import.meta.url);
const { chromium } = require('playwright');

const [repo, out, D, L, C] = process.argv.slice(2);
mkdirSync(`${out}/shots`, { recursive: true });
const AXE = readFileSync(require.resolve('axe-core/axe.min.js'), 'utf8');
const AXE_VERSION = require('axe-core/package.json').version;
const ALLOW = JSON.parse(readFileSync(`${repo}/scripts/ux/a11y-allow.json`, 'utf8')).allow || [];
for (const a of ALLOW) {
  if (!a.rule || !a.reason || !/^https?:\/\//.test(a.link || '')) {
    console.error(`a11y-allow.json: every entry needs rule, reason and link: ${JSON.stringify(a)}`);
    process.exit(2);
  }
}
// TB_A11Y_ONLY=<regex>: only pages whose "surface/name" matches (debugging)
const only = process.env.TB_A11Y_ONLY ? new RegExp(process.env.TB_A11Y_ONLY) : null;
const FAIL = new Set(['serious', 'critical']);

const UA = {
  android: 'Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0 Mobile Safari/537.36',
  iphone: 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1',
  mac: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15',
  linux: 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0 Safari/537.36',
};
const VP = {
  desktop: { viewport: { width: 1280, height: 800 }, deviceScaleFactor: 1 },
  phone: { viewport: { width: 412, height: 915 }, deviceScaleFactor: 2, isMobile: true, hasTouch: true, userAgent: UA.android },
  // the GL panel's phone width (glinet-screens.mjs)
  glphone: { viewport: { width: 390, height: 844 }, deviceScaleFactor: 2, isMobile: true, hasTouch: true },
};

// ---- files served from the repo (no agent needed) ----
const T = { html: 'text/html', js: 'application/javascript', css: 'text/css', svg: 'image/svg+xml', png: 'image/png' };
const type = (f) => T[f.split('.').pop()] || 'application/octet-stream';
const glFiles = {
  '/shell.html': `${repo}/scripts/ux/glinet/shell.html`,
  '/luci.html': `${repo}/scripts/ux/a11y-luci.html`,
  '/luci/view/tailbond.js': `${repo}/packaging/openwrt/luci/www/luci-static/resources/view/tailbond.js`,
  '/views/gl-sdk4-ui-tailbond.common.js': `${repo}/packaging/openwrt/gl-ui/www/views/gl-sdk4-ui-tailbond.common.js`,
  '/vendor/vue.runtime.min.js': require.resolve('vue/dist/vue.runtime.min.js'),
};
const serveDir = (dir, map = {}) => (route) => {
  const p = new URL(route.request().url()).pathname;
  const f = map[p] || `${dir}${p === '/' ? '/index.html' : p}`;
  try { return route.fulfill({ status: 200, body: readFileSync(f), contentType: type(f) }); } catch { return route.fulfill({ status: 404, body: '' }); }
};

// ---- the run ----
const browser = await chromium.launch();
const results = [];

// check(surface, name, opts): open a page, bring it into a state, run axe.
//   vp: desktop | phone | glphone; scheme: light | dark; ua: user agent
//   scope: a selector; only that part of the page is ours (GL, LuCI)
//   off: { rule: 'why it does not apply here' }
//   setup(page): routes and init scripts; open(page): navigate and act
// Pages are queued and run TB_A11Y_JOBS (default 4) at a time.
const queue = [];
function check(surface, name, opts) { queue.push([surface, name, opts]); }
async function runChecks() {
  const jobs = Math.max(1, Number(process.env.TB_A11Y_JOBS) || 4);
  let next = 0;
  const slots = queue.map(() => null);
  await Promise.all(Array.from({ length: jobs }, async () => {
    while (next < queue.length) { const i = next++; slots[i] = await runCheck(...queue[i]); }
  }));
  results.push(...slots.filter(Boolean));
  queue.length = 0;
}
async function runCheck(surface, name, { vp = 'desktop', scheme = 'light', ua, scope, off = {}, setup, open }) {
  const id = `${surface}/${name}${vp === 'desktop' ? '' : '@' + vp}${scheme === 'dark' ? '-dark' : ''}`;
  if (only && surface !== 'self-test' && !only.test(id)) return null;
  const ctx = await browser.newContext({ ...VP[vp], ...(ua ? { userAgent: ua } : {}), colorScheme: scheme, reducedMotion: 'reduce',
    bypassCSP: true, locale: 'en-US', timezoneId: 'UTC', acceptDownloads: true });
  ctx.setDefaultTimeout(15000);
  const page = await ctx.newPage();
  const r = { surface, page: id, vp, scheme, scope: scope || null, off, violations: [], error: null };
  page.on('pageerror', (e) => { if (process.env.TB_UX_DEBUG) console.log(`  ${id} pageerror: ${e.message}`); });
  page.on('dialog', (d) => d.accept());
  try {
    if (setup) await setup(page);
    await open(page);
    await page.waitForTimeout(200);
    r.url = page.url();
    // TB_A11Y_SHOTS=1: what each check looked at → <outdir>/shots/
    if (process.env.TB_A11Y_SHOTS) await page.screenshot({ path: `${out}/shots/${id.replace(/[^\w.@-]+/g, '_')}.png` });
    await page.addScriptTag({ content: AXE });
    const axeV = await page.evaluate(async ({ scope, off }) => {
      const rules = {};
      for (const k of Object.keys(off)) rules[k] = { enabled: false };
      const root = scope ? document.querySelector(scope) : document;
      if (!root) throw new Error(`scope ${scope} not found`);
      const res = await window.axe.run(root, { rules, resultTypes: ['violations'] });
      return res.violations.map((v) => ({ id: v.id, impact: v.impact, help: v.help, helpUrl: v.helpUrl,
        nodes: v.nodes.map((n) => ({ target: n.target.join(' '), html: n.html.slice(0, 400), summary: n.failureSummary })) }));
    }, { scope, off });
    const own = await page.evaluate(ownChecks, { scope, off });
    if (!off['tb-no-key-handler']) own.push(...await keyHandlers(page, scope));
    own.push(...await modalFocus(page, scope));
    r.violations = [...axeV, ...own];
  } catch (e) {
    r.error = String(e && e.message || e).split('\n')[0];
    await page.screenshot({ path: `${out}/error-${id.replace(/[^\w.-]+/g, '_')}.png` }).catch(() => {});
  }
  for (const v of r.violations) for (const n of v.nodes) {
    const a = ALLOW.find((x) => x.rule === v.id && (!x.surface || x.surface === surface) && (!x.target || n.target.includes(x.target)));
    if (a) n.allowed = { reason: a.reason, link: a.link };
  }
  await ctx.close();
  const bad = r.violations.filter((v) => FAIL.has(v.impact) && v.nodes.some((n) => !n.allowed));
  if (surface !== 'self-test' && (process.env.TB_UX_DEBUG || bad.length || r.error)) {
    console.log(`  ${r.error ? 'ERROR' : bad.length ? 'FAIL ' : 'ok   '} ${id}${r.error ? ': ' + r.error : ''}`);
    for (const v of bad) console.log(`        ${v.impact} ${v.id}: ${v.nodes.filter((n) => !n.allowed).map((n) => n.target).slice(0, 4).join(' | ')}`);
  }
  return r;
}

// The checks axe doesn't make (runs in the page).
function ownChecks({ scope, off }) {
  const root = scope ? document.querySelector(scope) : document.body;
  const out = [];
  const FOCUS = 'a[href],button,input,select,textarea,summary,label,iframe,[contenteditable="true"],[tabindex]:not([tabindex^="-"])';
  // roving tabindex: one radio/tab/option is in the Tab order and the arrows
  // move between the rest, so they are keyboard reachable with tabindex="-1"
  const ROVING = '[role="radiogroup"] [role="radio"],[role="tablist"] [role="tab"],[role="listbox"] [role="option"]';
  const shown = (el) => { const b = el.getBoundingClientRect(); return b.width > 0 && b.height > 0 && el.checkVisibility({ visibilityProperty: true, opacityProperty: true }); };
  const name = (el) => {
    let s = el.tagName.toLowerCase();
    if (el.id) return s + '#' + el.id;
    if (el.classList.length) s += '.' + [...el.classList].slice(0, 3).join('.');
    const p = el.parentElement;
    return p && p !== document.body && p !== root ? name(p) + ' > ' + s : s;
  };
  const html = (el) => el.outerHTML.slice(0, 400);
  if (!off['tb-pointer-only']) {
    const nodes = [];
    for (const el of root.querySelectorAll('*')) {
      if (getComputedStyle(el).cursor !== 'pointer' || el.closest(FOCUS) || el.querySelector(FOCUS) || el.closest(ROVING) || el.closest('[inert]')) continue;
      // cursor is inherited: report the outermost element of a clickable area
      const p = el.parentElement;
      if (p && root.contains(p) && p !== root && getComputedStyle(p).cursor === 'pointer' && !p.querySelector(FOCUS)) continue;
      if (!shown(el)) continue;
      nodes.push({ target: name(el), html: html(el), summary: 'Looks clickable (cursor: pointer) but a keyboard cannot reach it: make it a <button> or <a href>, or give it tabindex="0", a role and key handling.' });
    }
    if (nodes.length) out.push({ id: 'tb-pointer-only', impact: 'serious', help: 'Clickable things must be reachable from the keyboard', helpUrl: 'https://www.w3.org/WAI/WCAG22/Understanding/keyboard.html', nodes });
  }
  if (!off['tb-motion']) {
    const nodes = [];
    for (const a of document.getAnimations()) {
      const t = a.effect && a.effect.target;
      if (!t || a.playState !== 'running' || a.effect.getTiming().iterations !== Infinity || !root.contains(t) || !shown(t)) continue;
      nodes.push({ target: name(t) + (a.animationName ? ` (${a.animationName})` : ''), html: html(t), summary: 'An endless animation keeps running with "reduce motion" on: stop it under @media (prefers-reduced-motion: reduce).' });
    }
    if (nodes.length) out.push({ id: 'tb-motion', impact: 'serious', help: 'Endless animations stop when the system asks for reduced motion', helpUrl: 'https://www.w3.org/WAI/WCAG22/Understanding/pause-stop-hide.html', nodes });
  }
  return out;
}

// tb-modal-focus: with a modal open, Tab and Shift+Tab must stay inside it.
// An open .tb-dialog (or any aria-modal dialog) says the page behind is out
// of reach, so focus that escapes into it strands a keyboard user.
async function modalFocus(page, scope) {
  const SEL = scope ? `${scope} .tb-dialog, ${scope} [role="dialog"][aria-modal="true"]` : '.tb-dialog, [role="dialog"][aria-modal="true"]';
  if (!(await page.locator(SEL).count())) return [];
  const where = () => page.evaluate((sel) => {
    const d = document.querySelector(sel.split(', ')[0]) || document.querySelector('[role="dialog"][aria-modal="true"]');
    const a = document.activeElement;
    return { inside: !!d && d.contains(a), at: a ? (a.tagName.toLowerCase() + (a.className && typeof a.className === 'string' ? '.' + a.className.trim().split(/\s+/)[0] : '')) : 'none' };
  }, SEL);
  const nodes = [];
  for (const [key, label] of [['Tab', 'Tab'], ['Shift+Tab', 'Shift+Tab']]) {
    for (let i = 1; i <= 24; i++) {
      await page.keyboard.press(key);
      const r = await where();
      if (!r.inside) { nodes.push({ target: 'dialog', html: '', summary: `${label} #${i} left the dialog (focus on ${r.at})` }); break; }
    }
  }
  return nodes.length ? [{ id: 'tb-modal-focus', impact: 'serious', help: 'Keyboard focus stays inside an open dialog', helpUrl: 'https://www.w3.org/WAI/WCAG22/Understanding/focus-not-obscured-minimum.html', nodes }] : [];
}

// tb-no-key-handler: custom controls with no keydown/keyup/keypress listener
// of their own. Only the element's own listeners count: the GL view has an
// Escape handler on its root, which says nothing about Space on a checkbox.
async function keyHandlers(page, scope) {
  const n = await page.evaluate((scope) => {
    const root = scope ? document.querySelector(scope) : document.body;
    const ROLES = new Set(['button', 'checkbox', 'radio', 'switch', 'link', 'tab', 'menuitem', 'menuitemcheckbox', 'menuitemradio', 'option', 'treeitem']);
    window.__tbKeyCands = [...root.querySelectorAll('[tabindex]:not([tabindex^="-"])')]
      .filter((el) => ROLES.has(el.getAttribute('role')) && !el.matches('a[href],button,input,select,textarea,summary') && !el.closest('[inert]'));
    return window.__tbKeyCands.length;
  }, scope);
  if (!n) return [];
  const cdp = await page.context().newCDPSession(page);
  const nodes = [];
  for (let i = 0; i < n; i++) {
    const { result } = await cdp.send('Runtime.evaluate', { expression: `window.__tbKeyCands[${i}]` });
    const { listeners } = await cdp.send('DOMDebugger.getEventListeners', { objectId: result.objectId });
    if (listeners.some((l) => /^key(down|up|press)$/.test(l.type))) continue;
    const info = await page.evaluate((i) => { const e = window.__tbKeyCands[i]; return { target: e.tagName.toLowerCase() + (e.id ? '#' + e.id : '') + '[role=' + e.getAttribute('role') + ']' + (e.className ? '.' + String(e.className).trim().split(/\s+/).join('.') : ''), html: e.outerHTML.slice(0, 400) }; }, i);
    nodes.push({ ...info, summary: 'Tab reaches it, but Space and Enter do nothing: no key handler. Use a native control, or handle keydown (Space/Enter) like the click.' });
  }
  await cdp.detach();
  return nodes.length ? [{ id: 'tb-no-key-handler', impact: 'serious', help: 'Custom controls work from the keyboard', helpUrl: 'https://www.w3.org/WAI/WCAG22/Understanding/keyboard.html', nodes }] : [];
}

const wait = (ms) => (page) => page.waitForTimeout(ms);
const fakeFile = ([n, size]) => ({ name: n, mimeType: 'application/octet-stream', buffer: Buffer.alloc(Math.round(size), 7) });
// fixture steps (fixtures.mjs, ub-fixtures.mjs): {text}, {click}, {button}, {fill}, {files}, {wait}
async function steps(page, act) {
  if (!act) return;
  if (act.click) { await page.click(act.click); await page.waitForTimeout(300); }
  if (act.clickText) { await page.getByText(act.clickText).first().click(); await page.waitForTimeout(300); }
  for (const s of act.steps || []) {
    if (s.text) await page.getByText(s.text).first().click();
    if (s.click) await page.click(s.click);
    if (s.button) await page.getByRole('button', { name: s.button, exact: true }).first().click();
    if (s.fill) await page.fill(s.fill[0], s.fill[1]);
    if (s.files) await page.setInputFiles(s.files[0], s.files[1].map(fakeFile));
    if (s.wait) await page.waitForTimeout(s.wait);
    await page.waitForTimeout(300);
  }
}

// ============ self-test: a page with one of each known problem ============
const SELFTEST = `<!doctype html><html><head><title>self-test</title>
<style>@keyframes spin { to { transform: rotate(1turn); } } .m { display: inline-block; animation: spin 1s linear infinite; }</style></head>
<body><main><h1>self-test</h1><img src="data:,"><div style="cursor:pointer" onclick="1">click me</div>
<span role="checkbox" tabindex="0" aria-checked="false" onclick="1">tick</span><span class="m">*</span>
<div role="dialog" aria-modal="true"><button autofocus onclick="1">in</button></div><button onclick="1">outside</button></main></body></html>`;
const EXPECT = ['html-has-lang', 'image-alt', 'tb-pointer-only', 'tb-no-key-handler', 'tb-motion', 'tb-modal-focus'];
const st = await runCheck('self-test', 'known-problems', { setup: (p) => p.route('http://selftest.test/**', (r) => r.fulfill({ contentType: 'text/html', body: SELFTEST })),
  open: (p) => p.goto('http://selftest.test/') });
{
  const seen = new Set(st.violations.map((v) => v.id));
  const missed = EXPECT.filter((x) => !seen.has(x));
  if (st.error || missed.length) {
    console.error(`self-test: the run no longer sees ${missed.join(', ') || '(error) ' + st.error}; the checks are broken, not the pages`);
    process.exit(2);
  }
  console.log(`self-test: sees ${EXPECT.join(', ')}`);
}

// ======================= the site (site/index.html) =======================
for (const vp of ['desktop', 'phone']) for (const scheme of ['light', 'dark']) {
  await check('site', 'index', { vp, scheme,
    setup: (p) => p.route('http://site.test/**', serveDir(`${repo}/site`)),
    open: async (p) => { await p.goto('http://site.test/'); await p.waitForTimeout(300); } });
}

// ============ the phone page ("Get tailbond", landing.html) ===============
const landingOpen = (hash = '') => async (p) => { await p.goto(L + hash); await p.waitForTimeout(500); };
for (const scheme of ['light', 'dark']) {
  await check('phone-page', 'android', { vp: 'phone', scheme, open: landingOpen() });
  await check('phone-page', 'mac', { ua: UA.mac, scheme, open: landingOpen() });
}
await check('phone-page', 'no-app-yet', { vp: 'phone', open: landingOpen('#get-app') });
for (const plat of ['android', 'mac', 'iphone', 'linux']) {
  await check('phone-page', `tab-${plat}`, { vp: 'phone', ua: plat === 'iphone' ? UA.iphone : UA.android, open: landingOpen('#' + plat) });
  await check('phone-page', `tab-${plat}`, { ua: plat === 'linux' ? UA.linux : UA.mac, open: landingOpen('#' + plat) });
}

// ========================= the dashboard (agent/web) ======================
const dash = (base, hash = '', sel = '#tabs', state = 'visible') => async (p) => { await p.goto(base + hash); await p.waitForSelector(sel, { state }); await p.waitForTimeout(800); };
// a tab of the dashboard; a phone opening a server's page lands on "Add this
// phone" first, and the owner reveals the dashboard
const tab = (base, t) => async (p) => {
  await dash(base, '', '#tabs', 'attached')(p);
  const show = await p.$('#show-dashboard');
  if (show && await show.isVisible()) await show.click();
  await p.click(`#tabs button[data-tab="${t}"]`);
  await p.waitForTimeout(800);
};
// a real server-role tailbondd (a11y.sh), with a crash from last time
for (const scheme of ['light', 'dark']) await check('dashboard', 'server-connections', { scheme, open: dash(D) });
for (const t of ['clients', 'tailscale', 'maint', 'logs']) await check('dashboard', `server-${t}`, { open: tab(D, t) });
await check('dashboard', 'server-clients', { scheme: 'dark', open: tab(D, 'clients') });
await check('dashboard', 'server-add-device-code', { open: async (p) => {
  await tab(D, 'clients')(p);
  await p.fill('#client-name', 'tablet');
  await p.click('#add-client button[type=submit]');
  await p.waitForSelector('#qr-dialog[open]');
  await p.waitForTimeout(300);
} });
const report = async (p) => {
  await p.goto(D + '#report');
  await p.waitForSelector('#rep-what');
  await p.waitForFunction(() => document.querySelector('#rep-logs').value.length > 0).catch(() => {});
  await p.waitForTimeout(300);
};
await check('dashboard', 'report-a-problem', { open: report });
await check('dashboard', 'report-a-problem', { vp: 'phone', open: report });
await check('dashboard', 'report-a-problem-ready', { open: async (p) => {
  await report(p);
  await p.fill('#rep-what', 'The tunnel drops when I plug in the phone');
  await p.check('#rep-ok');
} });
// a phone opening the server's page: "Add this phone"
for (const scheme of ['light', 'dark']) await check('dashboard', 'add-this-phone', { vp: 'phone', scheme, open: dash(D, '#add-phone', '#add-phone:not(.hidden)') });
await check('dashboard', 'add-this-phone-opening', { vp: 'phone', open: async (p) => {
  await dash(D, '#add-phone', '#add-phone:not(.hidden)')(p);
  await p.click('#add-phone-btn');
  await p.waitForSelector('#add-phone-done:not(.hidden)');
} });
await check('dashboard', 'server-connections', { vp: 'phone', open: tab(D, 'connections') });
// a client-role tailbondd, not paired: Setup
await check('dashboard', 'client-setup-ask', { open: dash(C, '?ask=homebox.tail1234.ts.net', '#tab-setup:not(.hidden)') });
await check('dashboard', 'client-setup', { vp: 'phone', open: tab(C, 'setup') });
// the same client with states patched into its status (web-screens.mjs)
const base = await (await fetch(C + 'api/status')).json();
const patched = (st) => async (p) => {
  await p.route('**/api/status', (r) => r.fulfill({ contentType: 'application/json', body: JSON.stringify(st) }));
  await p.route('**/api/events', (r) => r.fulfill({ contentType: 'text/event-stream', body: `data: ${JSON.stringify(st)}\n\n` }));
};
const wifi = { id: 1001, iface: 'en0', kind: 'wifi', label: 'Wi-Fi', priority: 'primary', enabled: true, state: 'active', tx_bps: 1.2e6, rx_bps: 31e6, tx_bytes: 0, rx_bytes: 0, srtt_ms: 31 };
const usb = { id: 1002, iface: 'en7', kind: 'usb', label: 'iPhone (USB)', priority: 'secondary', enabled: true, state: 'active', tx_bps: 0, rx_bps: 0, tx_bytes: 18400000, rx_bytes: 210000000, srtt_ms: 48 };
const pixel = { id: 'and-7f3c2a-hub', label: 'pixel-8', addr: '192.168.43.1:1888', via: 'mdns', bonded: true, iface: 'en0' };
const mac = { ...base, paired: true, platform: 'darwin', state: 'connected', server: 'homebox.tail1234.ts.net', tunnel_ip: '10.77.0.4', uptime_sec: 3600,
  paths: [wifi, usb], use_local: true, use_shares: true, shares: [], totals: { tx_bps: 1.2e6, rx_bps: 31e6, tx_bytes: 0, rx_bytes: 0 } };
const ifaces = [{ name: 'en0', kind: 'wifi', label: 'Wi-Fi', up: true, share_label: 'Wi-Fi from tiger-mbp' },
  { name: 'en5', kind: 'ethernet', label: 'USB 10/100/1000 LAN', up: true, share_label: 'Ethernet from tiger-mbp' }];
const shareOff = { enabled: false, iface: '', label: '', port: 47077, state: 'off', used_bytes: 0, hubs: [], ifaces };
const shareOn = { ...shareOff, enabled: true, iface: 'en5', label: 'Ethernet from tiger-mbp', state: 'sharing', used_bytes: 131000000, cap_mb: 5000,
  allowed_hubs: ['slate7'], hubs: [{ hub: 'slate7', addr: '192.168.8.1:52011', up_bytes: 21000000, down_bytes: 110000000 }] };
const clientStates = {
  'client-connected': {},
  'client-error': { state: 'error', error: 'mqvpn exited: connection to home.example.net:443 timed out', tunnel_ip: '', uptime_sec: 0 },
  'client-disconnected': { state: 'disconnected', tunnel_ip: '', uptime_sec: 0 },
  'phone-hub-found': { state: 'disconnected', tunnel_ip: '', paths: [wifi], phone_hub: { found: [pixel] } },
  'phone-hub-using': { state: 'disconnected', tunnel_ip: '', paths: [], phone_hub: { found: [pixel], using: { ...pixel, mode: 'socks', service: 'Wi-Fi',
    local: '127.0.0.1:1889', since: new Date().toISOString(), up_bytes: 48200000, down_bytes: 912000000, conns: 6 } } },
  'phone-hub-lost': { state: 'disconnected', tunnel_ip: '', paths: [wifi], phone_hub: { found: [], note: 'Lost the phone hub, so the proxy is off again and this Mac uses its own connection.' } },
  'share-setup': { sharing: shareOff },
  'share-on': { sharing: shareOn },
  'share-unplugged': { sharing: { ...shareOn, state: 'error', error: "There's no connection called en5 on this computer right now (unplugged?).", hubs: [], used_bytes: 0 } },
  'hub-computer-share': { sharing: shareOff, paths: [wifi, { ...usb, id: 1003, iface: 'sh4f3c07', kind: 'shared-cellular', label: 'Ethernet from tiger-mbp' }],
    shares: [{ id: 'mac-4f3c07aa11', label: 'Ethernet from tiger-mbp', path: 'sh4f3c07', addr: '192.168.8.42:47077', via: 'mdns', state: 'ready', up_bytes: 21000000, down_bytes: 110000000, battery: -1, charging: false }] },
};
for (const [n, patch] of Object.entries(clientStates)) {
  await check('dashboard', n, { ua: UA.mac, setup: patched({ ...mac, ...patch }), open: dash(C, n.startsWith('share') ? '#share' : '') });
}
await check('dashboard', 'client-connected', { vp: 'phone', setup: patched(mac), open: dash(C) });
await check('dashboard', 'client-connected', { scheme: 'dark', ua: UA.mac, setup: patched(mac), open: dash(C) });
for (const n of ['client-setup', 'client-connected']) await check('dashboard', `${n}-tailscale-tab`, { ua: UA.mac, setup: patched(n === 'client-setup' ? base : mac), open: tab(C, 'tailscale') });
// password set: the sign-in dialog; someone else's device: "the home server's own page"
await check('dashboard', 'sign-in', { setup: (p) => p.route('**/api/**', (r) => r.fulfill({ status: 401, contentType: 'application/json', body: '{"ok":false,"error":"login required"}' })),
  open: async (p) => { await p.goto(D); await p.waitForSelector('#login-dialog[open]'); } });
await check('dashboard', 'owner-only', {
  setup: (p) => p.route('**/api/**', (r) => (/\/api\/(status|events)$/.test(new URL(r.request().url()).pathname) ? r.continue()
    : r.fulfill({ status: 403, contentType: 'application/json', body: '{"ok":false,"owner":"alex@example.com"}' }))),
  open: async (p) => {
    await dash(D, '', '#tabs', 'attached')(p);
    // the dialog may be up already (it's modal): click the tab from the page's side
    await p.evaluate(() => document.querySelector('#tabs button[data-tab="tailscale"]').click());
    await p.waitForSelector('#owner-dialog[open]');
  } });

// the server dashboard from fixtures (glinet-screens.mjs's Fedora homebox,
// updates-backup-screens.mjs's update and backup states): http://homebox.test/
const fedoraSettings = { role: 'server', mode: 'speed', public_addr: 'home.example.net', effective_public_addr: 'home.example.net:443', port: 443, subnet: '10.77.0.0/24', tls: 'tailscale', tls_name: 'homebox.tail1234.ts.net', version: '0.1.0-r42', glinet_ipk: true, password_set: false, ...landingSettings };
const fedoraTs = { installed: true, running: true, self: 'homebox', dns_name: 'homebox.tail1234.ts.net', tailnet: 'alex@example.com', ips: ['100.101.1.1', 'fd7a:115c:a1e0::1'], route: '10.77.0.0/24', advertised_routes: ['10.77.0.0/24'], approved_routes: ['10.77.0.0/24'], route_advertised: true, routes_approved: true, admin_url: 'https://login.tailscale.com/admin/machines', peers: 4, peers_online: 3 };
const fakeBackup = Buffer.from('TBBACKUP1\n{"v":1}\nAAAA\n');
// an /api/... call answered from a fixture map {"METHOD /api/path": value} (updates-backup-screens.mjs)
function answer(route, api, method, path) {
  const json = (v, code = 200) => route.fulfill({ status: code, contentType: 'application/json', body: JSON.stringify(v) });
  if (method === 'POST' && path === '/api/backup') return route.fulfill({ status: 200, body: fakeBackup, headers: { 'Content-Type': 'application/octet-stream', 'Content-Disposition': 'attachment; filename="backup.tbbackup"' } });
  const v = api[`${method} ${path}`];
  if (v !== undefined) return json(v);
  return method === 'GET' ? json({ ok: false, error: 'not found' }, 404) : json({ ok: true });
}
const homebox = ({ status, settings = fedoraSettings, ts = fedoraTs, health = fedoraHealth, api = {}, clock }) => async (p) => {
  if (clock) await p.clock.setFixedTime(new Date(clock));
  await p.route('http://homebox.test/**', (route) => {
    const req = route.request();
    const path = new URL(req.url()).pathname;
    const m = req.method();
    const json = (v, code = 200) => route.fulfill({ status: code, contentType: 'application/json', body: JSON.stringify(v) });
    if (path === '/api/status') return json(status);
    if (path === '/api/events') return route.fulfill({ status: 200, contentType: 'text/event-stream', body: `data: ${JSON.stringify(status)}\n\n` });
    if (path === '/api/settings') return json(settings);
    if (path === '/api/tailscale') return json(ts);
    if (path === '/api/health') return json(health);
    if (path === '/api/logs') return json({ entries: [] });
    if (path.startsWith('/api/')) return answer(route, api, m, path);
    return serveDir(`${repo}/agent/web`)(route);
  });
};
const H = 'http://homebox.test/';
for (const vp of ['desktop', 'phone']) {
  await check('dashboard', 'server-pair-request', { vp, setup: homebox({ status: fedoraStatus }), open: dash(H, '', '#pair-requests:not(.hidden)') });
}
await check('dashboard', 'server-fresh-install-health', { setup: homebox({ status: fedoraStatusFresh, health: fedoraHealthRoute,
  settings: { ...fedoraSettings, public_addr: '', effective_public_addr: '192.168.1.2:443', public_addr_guessed: true, tls: 'self-signed' },
  ts: { ...fedoraTs, routes_approved: false, approved_routes: [] } }), open: tab(H, 'clients') });
const maint = (act) => async (p) => { await tab(H, 'maint')(p); await p.waitForSelector('#upd-card .ub-head'); await p.waitForTimeout(300); await steps(p, act); };
for (const vp of ['desktop', 'phone']) {
  for (const [n, fx] of Object.entries(webStates)) if (vp === 'desktop' || webPhone.includes(n)) {
    await check('dashboard', n, { vp, setup: homebox({ ...fx, clock: ubClock }), open: maint() });
  }
  for (const [n, act] of Object.entries(webActions)) if (vp === 'desktop' || webPhone.includes(n)) {
    const fx = webStates[act.base];
    await check('dashboard', n, { vp, setup: homebox({ ...fx, api: { ...fx.api, ...(act.api || {}) }, clock: ubClock }), open: maint(act) });
  }
}

// ============== the GL.iNet admin panel view (VPN -> tailbond) =============
// The real view in the mock GL shell (scripts/ux/glinet/shell.html). Only
// #view is tailbond's; the menu and header around it are a mock of GL's.
const glSetup = (fx) => async (p) => {
  await p.route('http://gl.test/**', async (route) => {
    const u = new URL(route.request().url());
    if (u.pathname.startsWith('/live/')) {
      const req = route.request();
      const res = await route.fetch({ url: D.replace(/\/$/, '') + u.pathname.slice(5) + u.search, headers: { 'content-type': 'application/json' }, method: req.method(), postData: req.postData() || undefined });
      return route.fulfill({ response: res });
    }
    // uploads and downloads go to the same-origin proxy, not $rpcRequest
    if (u.pathname.startsWith('/tailbond-api/')) return answer(route, fx.api || {}, route.request().method(), '/api/' + u.pathname.slice('/tailbond-api/'.length));
    return serveDir('', glFiles)(route);
  });
  await p.addInitScript((f) => { window.__TB = f; }, fx);
  if (fx.clock) await p.clock.setFixedTime(new Date(fx.clock));
};
const glOpen = (fx, act) => async (p) => {
  await p.goto('http://gl.test/shell.html');
  await p.waitForFunction(() => window.__mounted === true);
  await p.waitForTimeout(700 + (fx.settle || 0));
  await steps(p, act);
  // Vue replaced #view with the view's own root element: that is what's checked
  await p.evaluate(() => window.__vm.$el.setAttribute('data-a11y-scope', 'gl'));
};
const gl = (name, fx, vp, act) => check('glinet', name, { vp, scope: '[data-a11y-scope]', setup: glSetup(fx), open: glOpen(fx, act),
  // off: region: the page's landmarks are GL's admin shell's, not the view's
  // (the view is one region inside GL's main content), and the mock shell
  // here is only a stand-in for GL's markup.
  off: { region: "GL's admin shell owns the page landmarks; the view sits inside its content area" } });
const withApi = (fx, act) => ({ ...fx, api: { ...fx.api, ...(act.api || {}) }, ...(act.after ? { after: act.after } : {}) });
const glPhoneToo = new Set([...glPhone, ...bfPhone, '03-found-server', '05-connected-two-wans', '11-server-pending-request', 'share-01-two-phones']);
for (const vp of ['desktop', 'glphone']) {
  const want = (n) => vp === 'desktop' || glPhoneToo.has(n);
  for (const [n, fx] of Object.entries(states)) if (want(n)) await gl(n, fx, vp);
  for (const [n, act] of Object.entries(actions)) if (act && want(n)) await gl(n, states[act.base], vp, act);
  for (const [n, fx] of Object.entries(shareStates)) if (want(n)) await gl(n, fx, vp);
  for (const [n, fx] of Object.entries(bfStates)) if (want(n)) await gl(n, fx, vp);
  for (const [n, act] of Object.entries(bfActions)) if (want(n)) await gl(n, withApi(bfStates[act.base], act), vp, act);
  for (const [n, fx] of Object.entries(glStates)) if (want(n)) await gl(n, fx, vp);
  for (const [n, act] of Object.entries(glActions)) if (want(n)) await gl(n, withApi(glStates[act.base], act), vp, act);
}
// Report a problem, against the real agent (report-screens.mjs)
await check('glinet', 'report-a-problem', { scope: '[data-a11y-scope]', off: { region: "GL's admin shell owns the page landmarks" },
  setup: glSetup({ live: true, role: 'server', model: 'GL-BE9300' }),
  open: async (p) => {
    await glOpen({})(p);
    await p.click('#tb-report-link');
    await p.getByRole('dialog', { name: 'Report a problem' }).waitFor();
    await p.waitForTimeout(500);
  } });

// ===================== the LuCI view (Services -> tailbond) ================
for (const vp of ['desktop', 'phone']) {
  await check('luci', 'services-tailbond', { vp, scope: '#view', off: { region: "LuCI's theme owns the page landmarks" },
    setup: (p) => p.route('http://gl.test/**', serveDir('', glFiles)),
    open: async (p) => { await p.goto('http://gl.test/luci.html'); await p.waitForFunction(() => window.__mounted === true); } });
}

await runChecks();
await browser.close();

// ============================== the report ==============================
const surfaces = {};
for (const r of results) {
  const s = surfaces[r.surface] ||= { pages: 0, errors: 0, fail: new Map(), info: new Map(), allowed: new Map() };
  s.pages++;
  if (r.error) s.errors++;
  for (const v of r.violations) for (const n of v.nodes) {
    // one finding per rule and element, however many pages show it
    const key = `${v.id}|${n.target}`;
    const bucket = n.allowed ? s.allowed : FAIL.has(v.impact) ? s.fail : s.info;
    if (!bucket.has(key)) bucket.set(key, { rule: v.id, impact: v.impact, target: n.target, pages: [] });
    bucket.get(key).pages.push(r.page);
  }
}
const summary = Object.fromEntries(Object.entries(surfaces).map(([k, s]) => [k, {
  pages: s.pages, errors: s.errors, serious_critical: s.fail.size, moderate_minor: s.info.size, allowed: s.allowed.size,
  findings: { fail: [...s.fail.values()], info: [...s.info.values()], allowed: [...s.allowed.values()] } }]));
const failed = Object.values(summary).some((s) => s.serious_critical || s.errors);
writeFileSync(`${out}/report.json`, JSON.stringify({ axe: AXE_VERSION, generated: new Date().toISOString(), pass: !failed,
  policy: 'fail on impact serious/critical (unless allow-listed); moderate/minor reported only', summary, pages: results }, null, 1));

const esc = (s) => String(s ?? '').replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
const rows = Object.entries(summary).map(([k, s]) => `<tr><th scope="row">${esc(k)}</th><td>${s.pages}</td><td class="${s.serious_critical ? 'bad' : ''}">${s.serious_critical}</td><td>${s.moderate_minor}</td><td>${s.allowed}</td><td class="${s.errors ? 'bad' : ''}">${s.errors}</td></tr>`).join('');
const pageHtml = results.map((r) => {
  const vs = r.violations.map((v) => `<li><b class="i-${esc(v.impact)}">${esc(v.impact)}</b> <a href="${esc(v.helpUrl)}">${esc(v.id)}</a>: ${esc(v.help)}<ul>${v.nodes.map((n) => `<li><code>${esc(n.target)}</code>${n.allowed ? ` <i>allowed: ${esc(n.allowed.reason)} (<a href="${esc(n.allowed.link)}">why</a>)</i>` : ''}<pre>${esc(n.html)}</pre><p>${esc(n.summary)}</p></li>`).join('')}</ul></li>`).join('');
  const offs = Object.entries(r.off).map(([k, why]) => `<li><code>${esc(k)}</code> off: ${esc(why)}</li>`).join('');
  return `<section><h3>${esc(r.page)}</h3><p class="meta">${esc(r.url || '')}${r.scope ? ` · checked: <code>${esc(r.scope)}</code> only` : ''}</p>${r.error ? `<p class="bad">harness error: ${esc(r.error)}</p>` : ''}${offs ? `<ul class="meta">${offs}</ul>` : ''}${vs ? `<ul>${vs}</ul>` : '<p class="meta">no violations</p>'}</section>`;
}).join('\n');
writeFileSync(`${out}/report.html`, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>tailbond accessibility report</title>
<style>body{font:15px/1.5 system-ui,sans-serif;max-width:1100px;margin:0 auto;padding:16px;color:#111}table{border-collapse:collapse}th,td{border:1px solid #bbb;padding:4px 10px;text-align:left}.bad{color:#a00;font-weight:700}.meta{color:#555;font-size:13px}pre{white-space:pre-wrap;background:#f4f4f4;padding:6px;font-size:12px}.i-critical,.i-serious{color:#a00}.i-moderate{color:#8a5a00}</style></head>
<body><main><h1>tailbond accessibility report</h1><p>axe-core ${esc(AXE_VERSION)} · ${esc(new Date().toISOString())} · <b class="${failed ? 'bad' : ''}">${failed ? 'FAIL' : 'PASS'}</b> (fails on serious and critical; moderate and minor are for information)</p>
<h2>Surfaces</h2><table><thead><tr><th scope="col">surface</th><th scope="col">pages</th><th scope="col">serious + critical</th><th scope="col">moderate + minor</th><th scope="col">allowed</th><th scope="col">harness errors</th></tr></thead><tbody>${rows}</tbody></table>
<p class="meta">Counts are distinct (rule, element) pairs across a surface's pages.</p><h2>Pages</h2>${pageHtml}</main></body></html>`);

for (const [k, s] of Object.entries(summary)) {
  console.log(`${(s.serious_critical || s.errors ? 'FAIL' : 'PASS').padEnd(5)} ${k.padEnd(11)} ${String(s.pages).padStart(3)} pages  serious+critical ${s.serious_critical}  moderate+minor ${s.moderate_minor}  allowed ${s.allowed}${s.errors ? `  harness errors ${s.errors}` : ''}`);
}
process.exit(failed ? 1 : 0);
