// Drives the "Add this phone" flow on a Pixel-sized viewport; see web-screens.sh.
import { createRequire } from 'node:module';
const require = createRequire(import.meta.url);
const { chromium } = require('playwright');

const [url, out] = process.argv.slice(2);
const phone = {
  viewport: { width: 412, height: 915 },
  deviceScaleFactor: 2.625,
  isMobile: true,
  hasTouch: true,
  userAgent: 'Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0 Mobile Safari/537.36',
};

const browser = await chromium.launch();
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
    // the desktop-style path: Devices tab, Add a device → code to scan
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
