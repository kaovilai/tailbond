// Drives the phone page and the "Add this phone" flow at phone width; see web-screens.sh.
import { createRequire } from 'node:module';
const require = createRequire(import.meta.url);
const { chromium } = require('playwright');

const [url, out, landing, client] = process.argv.slice(2);
const phone = {
  viewport: { width: 412, height: 915 },
  deviceScaleFactor: 2.625,
  isMobile: true,
  hasTouch: true,
  userAgent: 'Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0 Mobile Safari/537.36',
};

const desktop = { viewport: { width: 1280, height: 800 }, deviceScaleFactor: 1 };

const browser = await chromium.launch();
// The phone page (http://homebox/): on the phone, light and dark, whole page;
// after the app link fell back to it (#get-app); on a computer (QR code).
for (const scheme of ['light', 'dark']) {
  const ctx = await browser.newContext({ ...phone, colorScheme: scheme });
  const page = await ctx.newPage();
  await page.goto(landing);
  await page.waitForTimeout(400);
  const sfx = scheme === 'dark' ? '-dark' : '';
  await page.screenshot({ path: `${out}/web-00-phone-page${sfx}.png`, fullPage: true });
  if (scheme === 'light') {
    await page.screenshot({ path: `${out}/web-00a-phone-page-first-screen.png` });
    const href = await page.getAttribute('a.btn.primary[href^="intent:"]', 'href');
    if (!href || !href.includes('scheme=tailbond')) throw new Error('no app link: ' + href);
    if (!(await page.textContent('body')).includes('Download tailbond (')) throw new Error('no download button');
    await page.goto(landing + '#get-app');
    await page.waitForTimeout(400);
    await page.screenshot({ path: `${out}/web-00b-phone-page-no-app-yet.png` });
  }
  await ctx.close();
}
{
  const ctx = await browser.newContext({ ...desktop });
  const page = await ctx.newPage();
  await page.goto(landing);
  await page.waitForTimeout(400);
  await page.screenshot({ path: `${out}/web-00c-phone-page-on-computer.png`, fullPage: true });
  // the dashboard's Devices tab: "Add an Android phone" with the QR code
  await page.goto(url);
  await page.waitForSelector('#tabs');
  await page.click('#tabs button[data-tab="clients"]');
  await page.waitForTimeout(800);
  await page.screenshot({ path: `${out}/web-04-dashboard-devices-phone-card.png`, fullPage: true });
  await ctx.close();
}

for (const scheme of ['light', 'dark']) {
  const ctx = await browser.newContext({ ...phone, colorScheme: scheme });
  const page = await ctx.newPage();
  const sfx = scheme === 'dark' ? '-dark' : '';
  await page.goto(url + '#add-phone');
  await page.waitForSelector('#add-phone:not(.hidden)');
  await page.waitForTimeout(800);
  await page.screenshot({ path: `${out}/web-01-add-this-phone${sfx}.png` });
  if (scheme === 'light') {
    // the deep link has no handler in headless Chromium, so the page stays put
    page.on('dialog', (d) => d.accept());
    await page.click('#add-phone-btn');
    await page.waitForSelector('#add-phone-done:not(.hidden)');
    await page.waitForTimeout(400);
    await page.screenshot({ path: `${out}/web-02-after-tap.png` });
    const href = await page.getAttribute('#phone-open', 'href');
    if (!href.startsWith('tailbond://pair?d=')) throw new Error('no pairing deep link: ' + href);
    // the other-screen path: reveal the dashboard, Devices tab, Add a device → code to scan
    await page.click('#show-dashboard');
    await page.click('#tabs button[data-tab="clients"]');
    await page.fill('#client-name', 'tablet');
    await page.click('#add-client button[type=submit]');
    await page.waitForSelector('#qr-dialog[open]');
    await page.waitForTimeout(300);
    await page.screenshot({ path: `${out}/web-03-devices-scan-code.png` });
  }
  await ctx.close();
}
// "Get tailbond" per platform (#android, #mac, #iphone, #linux) at phone
// and desktop widths, each with that platform's own browser (the page picks
// the section from it too), and the switcher actually switching.
const uas = {
  android: phone.userAgent,
  mac: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15',
  iphone: 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1',
  linux: 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0 Safari/537.36',
};
for (const [plat, ua] of Object.entries(uas)) {
  for (const [form, base] of [['phone', { ...phone }], ['desktop', { ...desktop }]]) {
    // a phone width is a phone (Android unless it's the iPhone tab), a
    // desktop width a computer (a Mac, or Linux for the Linux tab)
    const who = form === 'phone' ? (plat === 'iphone' ? uas.iphone : phone.userAgent) : (plat === 'linux' ? uas.linux : uas.mac);
    const ctx = await browser.newContext({ ...base, userAgent: who });
    const page = await ctx.newPage();
    await page.goto(landing + '#' + plat);
    await page.waitForTimeout(600);
    const shown = await page.evaluate(() => [...document.querySelectorAll('.plat')].filter((s) => getComputedStyle(s).display !== 'none').map((s) => s.id));
    if (shown.length !== 1 || shown[0] !== plat) throw new Error(`#${plat} (${form}) shows ${shown}`);
    const on = await page.getAttribute('nav.plats a.on', 'href');
    if (on !== '#' + plat) throw new Error(`#${plat} (${form}): tab ${on} highlighted`);
    await page.screenshot({ path: `${out}/web-05-get-${plat}-${form}.png`, fullPage: true });
    if (form === 'desktop' && plat === 'mac') {
      await page.screenshot({ path: `${out}/web-05-get-mac-desktop-first-screen.png` });
      // the switcher: Android from the Mac (the QR to scan with the phone)
      await page.click('nav.plats a[href="#android"]');
      await page.waitForTimeout(300);
      if (!(await page.isVisible('#android .qr svg'))) throw new Error('Mac → Android: no QR');
      await page.screenshot({ path: `${out}/web-06-mac-picks-android.png` });
      await page.click('nav.plats a[href="#mac"]');
      await page.click('#mac button.copy');
      await page.waitForTimeout(200);
      const t = await page.textContent('#mac button.copy');
      if (!/Copied|Select/.test(t)) throw new Error('copy button: ' + t);
    }
    await ctx.close();
  }
}
// dark, the Mac section on a Mac
{
  const ctx = await browser.newContext({ ...desktop, userAgent: uas.mac, colorScheme: 'dark' });
  const page = await ctx.newPage();
  await page.goto(landing);
  await page.waitForTimeout(400);
  await page.screenshot({ path: `${out}/web-05-get-mac-desktop-dark.png`, fullPage: true });
  await ctx.close();
}
// the device being added: its own dashboard, opened from the page's
// "Open tailbond and ask homebox" (/?ask=<full name>)
if (client) {
  const ctx = await browser.newContext({ ...desktop, userAgent: uas.mac });
  const page = await ctx.newPage();
  await page.goto(client + '?ask=homebox.tail1234.ts.net');
  await page.waitForSelector('#tab-setup:not(.hidden)');
  await page.waitForTimeout(500);
  const v = await page.inputValue('#ask-server');
  const b = await page.textContent('#ask-btn');
  if (v !== 'homebox.tail1234.ts.net' || b !== 'Ask homebox') throw new Error(`ask prefill: ${v} / ${b}`);
  await page.screenshot({ path: `${out}/web-07-mac-dashboard-ask-prefilled.png` });
  await ctx.close();
}
await browser.close();
