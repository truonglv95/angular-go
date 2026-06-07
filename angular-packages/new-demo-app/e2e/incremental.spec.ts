import { test, expect } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const projectRoot = path.resolve(__dirname, '..');
const appHtml = path.join(projectRoot, 'src/app/app.html');
const appTs = path.join(projectRoot, 'src/app/app.ts');
const appScss = path.join(projectRoot, 'src/app/app.scss');

function read(filePath: string): string {
  return fs.readFileSync(filePath, 'utf8');
}

function write(filePath: string, text: string): void {
  fs.writeFileSync(filePath, text);
}

async function reloadAfterEdit(page: any): Promise<void> {
  await page.waitForTimeout(1000);
  await page.reload();
  await page.waitForLoadState('networkidle');
}

test.describe.configure({ mode: 'serial' });

test.describe('Incremental edit matrix', () => {
  let originalHtml = '';
  let originalTs = '';
  let originalScss = '';

  test.beforeAll(() => {
    originalHtml = read(appHtml);
    originalTs = read(appTs);
    originalScss = read(appScss);
  });

  test.afterEach(() => {
    write(appHtml, originalHtml);
    write(appTs, originalTs);
    write(appScss, originalScss);
  });

  test.afterAll(() => {
    write(appHtml, originalHtml);
    write(appTs, originalTs);
    write(appScss, originalScss);
  });

  test('external template edit updates the rendered app', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('.demo-heading')).toBeVisible();

    const marker = `Incremental template marker ${Date.now()}`;
    write(appHtml, originalHtml.replace('<router-outlet></router-outlet>', `<p data-testid="incremental-template-marker">${marker}</p>\n\n        <router-outlet></router-outlet>`));

    await reloadAfterEdit(page);
    await expect(page.getByTestId('incremental-template-marker')).toHaveText(marker, { timeout: 30000 });
  });

  test('component TypeScript edit updates bound state', async ({ page }) => {
    await page.goto('/');
    await page.locator('.p-menu-item-content').filter({ hasText: 'Input' }).click();
    await expect(page.locator('#customerName')).toHaveValue('Go compiler');

    const marker = `Go incremental ${Date.now()}`;
    write(appTs, originalTs.replace("customerName = 'Go compiler';", `customerName = '${marker}';`));

    await reloadAfterEdit(page);
    await page.locator('.p-menu-item-content').filter({ hasText: 'Input' }).click();
    await expect(page.locator('#customerName')).toHaveValue(marker, { timeout: 30000 });
    await expect(page.locator('p').filter({ hasText: `Value: ${marker}` })).toBeVisible();
  });

  test('external style edit updates component styles', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('.demo-heading')).toBeVisible();

    write(appScss, `${originalScss}\n.demo-heading { color: rgb(220, 38, 38); }\n`);

    await reloadAfterEdit(page);
    await expect(page.locator('.demo-heading')).toHaveCSS('color', 'rgb(220, 38, 38)', { timeout: 30000 });
  });
});
