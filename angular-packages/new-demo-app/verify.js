const puppeteer = require('puppeteer');

(async () => {
  const browser = await puppeteer.launch();
  const page = await browser.newPage();
  await page.goto('http://localhost:4173/');
  await page.waitForSelector('h1', { timeout: 3000 }).catch(() => console.log('h1 not found'));
  
  const h1Texts = await page.$$eval('h1', h1s => h1s.map(h1 => h1.textContent));
  console.log('H1 Texts:', h1Texts);
  
  const errors = [];
  page.on('console', msg => {
    if (msg.type() === 'error') errors.push(msg.text());
  });
  await new Promise(r => setTimeout(r, 1000));
  console.log('Errors:', errors);
  
  await browser.close();
})();
