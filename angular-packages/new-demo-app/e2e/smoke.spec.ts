import { test, expect } from '@playwright/test';

test.describe('Runtime Smoke Suite', () => {
  test.beforeEach(async ({ page }) => {
    // Assert no console errors
    page.on('pageerror', (err) => {
      expect(err.message).toBeNull();
    });
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        throw new Error(`Console error: ${msg.text()}`);
      }
    });

    await page.goto('/');
  });

  test('renders menu and updates views', async ({ page }) => {
    await expect(page.locator('.p-menu')).toBeVisible();

    // Default view should be Button
    await expect(page.locator('h1').filter({ hasText: 'PrimeNG Components' })).toBeVisible();

    // Click on Input view
    await page.locator('.p-menu-item-content').filter({ hasText: 'Input' }).click();
    await expect(page.locator('label').filter({ hasText: 'Customer name' })).toBeVisible();
    
    // Type in input and verify two-way binding
    const input = page.locator('#customerName');
    await input.fill('Playwright Test');
    await expect(page.locator('p').filter({ hasText: 'Value: Playwright Test' })).toBeVisible();
  });

  test('checkbox and input events update model', async ({ page }) => {
    await page.locator('.p-menu-item-content').filter({ hasText: 'Form Elements' }).click();
    const checkbox = page.locator('.p-checkbox-box').first();
    await checkbox.click({ force: true });
    
    const radios = page.locator('.p-radiobutton-box');
    await radios.nth(1).click({ force: true });
  });

  test('dropdown panels render', async ({ page }) => {
    await page.locator('.p-menu-item-content').filter({ hasText: 'Form Elements' }).click();
    
    // The dropdown is the p-select
    const select = page.locator('p-select');
    await select.click();
    
    // Check if dropdown panel is visible by looking for options
    await expect(page.getByRole('option', { name: 'Rome', exact: true }).first()).toBeVisible();
    
    // Select "Rome"
    await page.getByRole('option', { name: 'Rome', exact: true }).first().click();
  });

  test('table renders known content', async ({ page }) => {
    await page.locator('.p-menu-item-content').filter({ hasText: 'Table' }).click();
    
    const table = page.locator('p-table');
    await expect(table).toBeVisible();
    
    // Should have rows
    await expect(page.locator('tr').filter({ hasText: 'Keyboard' })).toBeVisible();
    await expect(page.locator('tr').filter({ hasText: 'Monitor' })).toBeVisible();
  });

  test('tabs render and can be switched', async ({ page }) => {
    await page.locator('.p-menu-item-content').filter({ hasText: 'Tabs' }).click();
    
    await expect(page.locator('p-tablist')).toBeVisible();
    await page.getByRole('tab', { name: 'Header II', exact: true }).first().click();
    await expect(page.locator('p-tabpanel').filter({ hasText: 'eaque ipsa quae' })).toBeVisible();
  });

  test('deferred component loads after trigger', async ({ page }) => {
    await page.locator('.p-menu-item-content').filter({ hasText: 'Defer' }).click();
    
    // Should see placeholder initially
    await expect(page.locator('text=Placeholder...')).toBeVisible();
    
    // Click button to trigger defer
    await page.locator('button').filter({ hasText: 'Load Deferred' }).click();
    
    // Should see deferred content
    await expect(page.locator('.deferred-content')).toBeVisible();
    await expect(page.locator('text=Deferred Component Loaded!')).toBeVisible();
  });
});
