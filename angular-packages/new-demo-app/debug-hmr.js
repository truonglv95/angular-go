import { chromium } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const appHtml = path.resolve(__dirname, 'src/app/app.html');

async function run() {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();

  let requests = [];
  page.on('request', (req) => {
    const url = req.url();
    console.log(`[Network Request] ${url}`);
    if (url.includes('/@ng/component')) {
      requests.push(url);
    }
  });

  page.on('console', (msg) => {
    console.log(`Console [${msg.type()}]: ${msg.text()}`);
  });

  page.on('pageerror', (err) => {
    console.error(`Page error: ${err.message}`);
  });

  try {
    console.log('Navigating to http://localhost:4201/');
    await page.goto('http://localhost:4201/', { waitUntil: 'networkidle', timeout: 10000 });
    
    // Set a reload marker on window to detect full page reloads
    await page.evaluate(() => {
      window.reloadMarker = 'persisted';
    });

    console.log('Reload marker set. Initial requests for /@ng/component:', requests.length);
    requests = []; // Clear initial load requests

    // Edit app.html dynamically
    const originalHtml = fs.readFileSync(appHtml, 'utf8');
    const headingMatch = originalHtml.match(/<h1 class="demo-heading">([^<]+)<\/h1>/);
    if (!headingMatch) {
      throw new Error('Could not find h1.demo-heading in app.html');
    }
    const currentHeading = headingMatch[1].trim();
    console.log('Current heading on disk:', currentHeading);

    const targetHeading = `HMR WORKS REAL EDIT SUCCESS ${Date.now()}`;
    console.log(`Editing app.html to: ${targetHeading}`);
    const updatedHtml = originalHtml.replace(currentHeading, targetHeading);
    fs.writeFileSync(appHtml, updatedHtml, 'utf8');

    console.log('Waiting for HMR text update...');
    await page.waitForSelector(`text=${targetHeading}`, { timeout: 15000 });
    console.log('HMR text update detected!');

    const headingText = await page.textContent('.demo-heading');
    console.log('ACTUAL HEADING TEXT IN BROWSER:', headingText.trim());

    // Check if page reloaded
    const marker = await page.evaluate(() => window.reloadMarker);
    if (marker === 'persisted') {
      console.log('SUCCESS: Page did NOT reload (HMR applied in-place)');
    } else {
      console.error('FAIL: Page reloaded (reloadMarker was lost)');
    }

    // Check all resource entries from the browser performance API
    const resources = await page.evaluate(() => {
      return performance.getEntriesByType('resource').map(r => r.name);
    });
    
    const hmrRequests = resources.filter(r => r.includes('/@ng/component'));
    console.log('--- All HMR Requests in Browser ---');
    hmrRequests.forEach(r => console.log(r));
    console.log('-----------------------------------');
    console.log('Total HMR requests in browser:', hmrRequests.length);

    // Restore app.html
    fs.writeFileSync(appHtml, originalHtml, 'utf8');

  } catch (err) {
    console.error('HMR test failed:', err);
    try {
      const content = await page.content();
      console.log('--- Page Content on Failure ---');
      console.log(content);
      console.log('-------------------------------');
      await page.screenshot({ path: 'hmr-failure.png' });
      console.log('Screenshot saved to hmr-failure.png');
    } catch (e) {
      console.error('Failed to take screenshot/dump HTML:', e);
    }
  } finally {
    await browser.close();
  }
}

run();
