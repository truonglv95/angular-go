import { chromium } from '@playwright/test';

async function run() {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();

  page.on('requestfailed', (req) => {
    console.error(`Request failed: ${req.url()} - ${req.failure()?.errorText}`);
  });

  page.on('response', (res) => {
    if (res.status() >= 400) {
      console.error(`HTTP Error ${res.status()}: ${res.url()}`);
    }
  });

  page.on('console', (msg) => {
    console.log(`Console [${msg.type()}]: ${msg.text()}`);
  });

  page.on('pageerror', (err) => {
    console.error(`Page error: ${err.message}\n${err.stack}`);
  });

  try {
    console.log('Navigating to http://localhost:4201/');
    await page.goto('http://localhost:4201/', { waitUntil: 'networkidle', timeout: 10000 });
    console.log('Navigation complete');
    const heading = await page.locator('h1.demo-heading').textContent();
    console.log('Page Heading:', heading);
  } catch (err) {
    console.error('Navigation failed:', err);
  } finally {
    await browser.close();
  }
}

run();
