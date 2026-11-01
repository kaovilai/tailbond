// Drives the phone page and the "Add this phone" flow at phone width; see web-screens.sh.
import { createRequire } from 'node:module';
const require = createRequire(import.meta.url);
const { chromium } = require('playwright');

const [url, out, landing] = process.argv.slice(2);
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
await browser.close();
