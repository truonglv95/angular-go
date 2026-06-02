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

  test('dialog opens and closes', async ({ page }) => {
    await page.locator('.p-menu-item-content').filter({ hasText: 'Dialog' }).click();
    
    await expect(page.locator('.p-dialog')).toBeHidden();
    await page.locator('button').filter({ hasText: 'Show Dialog' }).click();
    await expect(page.locator('.p-dialog')).toBeVisible();
    await expect(page.locator('text=Update your information.')).toBeVisible();
    
    await page.locator('button').filter({ hasText: 'Cancel' }).click();
    await expect(page.locator('.p-dialog')).toBeHidden();
  });

  test('datepicker popups and selects', async ({ page }) => {
    await page.locator('.p-menu-item-content').filter({ hasText: 'Extra' }).click();
    
    const datepicker = page.locator('p-datepicker input');
    await datepicker.click();
    await expect(page.locator('.p-datepicker')).toBeVisible();
    
    await page.locator('td:not(.p-datepicker-other-month) .p-datepicker-day:not(.p-disabled)').first().click();
    await expect(page.getByRole('dialog', { name: 'Choose Date' })).toBeHidden();
  });

  test('tree and treetable expand', async ({ page }) => {
    // Tree
    await page.locator('.p-menu-item-content').filter({ hasText: 'Tree' }).click();
    await expect(page.locator('.p-tree')).toBeVisible();
    const treeNode = page.locator('.p-tree-node-toggle-icon').first();
    await treeNode.click();
    
    // TreeTable in Advanced
    await page.locator('.p-menu-item-content').filter({ hasText: 'Advanced' }).click();
    await expect(page.getByText('Applications')).toHaveCount(1);
    await expect(page.getByText('200mb')).toHaveCount(1);
    await expect(page.getByText('Folder')).toHaveCount(1);
  });

  test('accordion and panel toggle', async ({ page }) => {
    await page.locator('.p-menu-item-content').filter({ hasText: 'Accordion' }).click();
    await expect(page.locator('p-accordion')).toBeVisible();
    await page.locator('p-accordion-header').filter({ hasText: 'Header II' }).first().click();
    await expect(page.locator('p-accordion-content').filter({ hasText: 'Sed ut perspiciatis' }).first()).toBeVisible();

    await page.locator('.p-menu-item-content').filter({ hasText: 'Advanced' }).click();
    const panelHeader = page.locator('p-panel .p-panel-header').first();
    await expect(panelHeader).toBeVisible();
    // Toggle
    await page.locator('p-panel .p-panel-toggle-button').click();
  });

  test('picklist and orderlist render', async ({ page }) => {
    await page.locator('.p-menu-item-content').filter({ hasText: 'Advanced' }).click();
    await expect(page.locator('p-picklist')).toBeVisible();
    await expect(page.locator('p-orderlist')).toBeVisible();
    
    // Verify items
    await expect(page.locator('p-picklist').locator('text=San Francisco').first()).toBeVisible();
  });

  test('splitter renders', async ({ page }) => {
    await page.locator('.p-menu-item-content').filter({ hasText: 'Advanced' }).click();
    await expect(page.locator('p-splitter')).toBeVisible();
    await expect(page.locator('text=Panel 1')).toBeVisible();
    await expect(page.locator('text=Panel 2')).toBeVisible();
  });

  test('router navigation works', async ({ page }) => {
    await expect(page.locator('.dummy-route-content')).toBeHidden();
    await page.locator('a.route-link').click();
    await expect(page.locator('.dummy-route-content')).toBeVisible();
    await expect(page.locator('text=Routed Component Successfully Loaded!')).toBeVisible();
  });
});
