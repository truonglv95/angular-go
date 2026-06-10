const { chromium } = require('playwright');
const { exec } = require('child_process');

(async () => {
  const server = exec('npx vite preview');
  await new Promise(r => setTimeout(r, 3000));
  
  const browser = await chromium.launch();
  const page = await browser.newPage();
  await page.goto('http://localhost:4173/');
  await page.waitForSelector('h1', { timeout: 3000 }).catch(() => console.log('h1 not found'));
  
  const h1Texts = await page.$$eval('h1', h1s => h1s.map(h1 => h1.textContent));
  console.log('H1 Texts:', h1Texts);
  
  await browser.close();
  server.kill();
})();
