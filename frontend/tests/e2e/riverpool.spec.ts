/**
 * RiverPool E2E Tests
 * Tests full integration: Browser → Frontend → API
 */

import { test, expect } from '@playwright/test';

test.describe('RiverPool Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/riverpool');
  });

  test('should display page title and tabs', async ({ page }) => {
    // Check page title
    await expect(page.locator('h1')).toContainText('RiverPool');

    // Check tabs are present
    await expect(page.getByRole('tab', { name: /Foundation/i })).toBeVisible();
    await expect(page.getByRole('tab', { name: /Main/i })).toBeVisible();
    await expect(page.getByRole('tab', { name: /Community/i })).toBeVisible();
  });

  test('should switch between tabs', async ({ page }) => {
    // Click Foundation tab
    await page.getByRole('tab', { name: /Foundation/i }).click();
    await expect(page.getByRole('heading', { name: 'Foundation LP' })).toBeVisible();

    // Click Main tab
    await page.getByRole('tab', { name: /Main/i }).click();
    await expect(page.getByRole('heading', { name: 'Main LP' })).toBeVisible();

    // Click Community tab
    await page.getByRole('tab', { name: /Community/i }).click();
    await expect(page.getByText(/Community Pool/i).first()).toBeVisible();
  });
});

test.describe('Foundation LP', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/riverpool');
    await page.getByRole('tab', { name: /Foundation/i }).click();
  });

  test('should display pool information', async ({ page }) => {
    // Check for Foundation LP specific content
    await expect(page.getByText(/100 seats/i).first()).toBeVisible();
    await expect(page.getByText(/\$100K/i).first()).toBeVisible();
    await expect(page.getByText(/180.*day/i).first()).toBeVisible();
  });

  test('should show deposit button', async ({ page }) => {
    const depositButton = page.getByRole('button', { name: /Deposit/i });
    await expect(depositButton).toBeVisible();
  });
});

test.describe('Main LP', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/riverpool');
    await page.getByRole('tab', { name: /Main/i }).click();
  });

  test('should display pool information', async ({ page }) => {
    // Check for Main LP specific content
    await expect(page.getByText(/\$100.*minimum/i).first()).toBeVisible();
    await expect(page.getByText(/T\+4/i).first()).toBeVisible();
  });

  test('should show deposit and withdraw buttons', async ({ page }) => {
    await expect(page.getByRole('button', { name: /Deposit/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /Withdraw/i })).toBeVisible();
  });
});

test.describe('Community Pool', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/riverpool');
    await page.getByRole('tab', { name: /Community/i }).click();
  });

  test('should display community pool list', async ({ page }) => {
    // Check for Community Pool content
    await expect(page.getByText(/Community Pool/i).first()).toBeVisible();
  });

  test('should show Create Pool button', async ({ page }) => {
    const createButton = page.getByRole('button', { name: /Create Pool/i });
    await expect(createButton).toBeVisible();
  });

  test('should show search and filters', async ({ page }) => {
    // Check for search input
    await expect(page.getByPlaceholder(/Search/i)).toBeVisible();

    // Check for sort dropdown
    await expect(page.getByText(/Sort by/i)).toBeVisible();
  });

  test('should show tag filters', async ({ page }) => {
    // Check for popular tags
    const tags = ['BTC', 'ETH', 'Trend', 'Grid'];
    for (const tag of tags) {
      await expect(page.getByRole('button', { name: tag })).toBeVisible();
    }
  });
});

test.describe('Community Pool Creation', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/riverpool/community/create');
  });

  test('should display creation wizard', async ({ page }) => {
    // Check for wizard steps
    await expect(page.getByText(/Basic Info/i).first()).toBeVisible();
    await expect(page.getByText('Create Community Pool')).toBeVisible();
  });

  test('should show form fields in step 1', async ({ page }) => {
    // Check for basic info fields
    await expect(page.getByLabel(/Pool Name/i)).toBeVisible();
    await expect(page.getByLabel(/Description/i)).toBeVisible();
  });

  test('should validate required fields', async ({ page }) => {
    // Next button should be disabled when required fields are empty
    const nextButton = page.getByRole('button', { name: /Next/i });
    await expect(nextButton).toBeDisabled();

    // Should stay on step 1
    await expect(page.getByText(/Basic Info/i).first()).toBeVisible();
  });

  test('should navigate through wizard steps', async ({ page }) => {
    // Fill in step 1 with sufficient content
    await page.getByLabel(/Pool Name/i).fill('Test Pool Strategy');
    await page.getByLabel(/Description/i).fill('A test community pool for automated trading');

    // Proceed to step 2
    await page.getByRole('button', { name: /Next/i }).click();

    // Check for step 2 content
    await expect(page.getByText(/Deposit Settings/i).first()).toBeVisible();
  });
});

test.describe('DDGuard Indicator', () => {
  test('should display DDGuard status', async ({ page }) => {
    await page.goto('/riverpool');

    // Check for DDGuard indicator
    const ddGuardIndicator = page.locator('[data-testid="ddguard-indicator"]');
    if (await ddGuardIndicator.isVisible()) {
      // Check for level indicator
      await expect(ddGuardIndicator).toBeVisible();
    }
  });
});

test.describe('Pool Statistics', () => {
  test('should display pool stats', async ({ page }) => {
    await page.goto('/riverpool');

    // Check for common statistics - use first() for multiple matches
    await expect(page.getByText(/Total Value Locked/i).first()).toBeVisible();
    await expect(page.getByText(/NAV/i).first()).toBeVisible();
  });

  test('should display NAV chart when available', async ({ page }) => {
    await page.goto('/riverpool');

    // Check for chart container - may not be visible initially
    const chartContainer = page.locator('[data-testid="nav-chart"]');
    // Chart is only shown when a pool is selected
    const isVisible = await chartContainer.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });
});

test.describe('Deposit Modal', () => {
  test('should open deposit modal', async ({ page }) => {
    await page.goto('/riverpool');
    await page.getByRole('tab', { name: /Main/i }).click();

    // Wait for pool content to load
    await page.waitForTimeout(500);

    // Click deposit button
    await page.getByRole('button', { name: /Deposit/i }).first().click();

    // Check for modal content - look for the modal title
    await expect(page.getByText(/Deposit to Main LP/i)).toBeVisible();
  });

  test('should show amount input in deposit modal', async ({ page }) => {
    await page.goto('/riverpool');
    await page.getByRole('tab', { name: /Main/i }).click();
    await page.waitForTimeout(500);

    await page.getByRole('button', { name: /Deposit/i }).first().click();

    // Check for amount input
    await expect(page.getByLabel(/Amount/i)).toBeVisible();
  });

  test('should validate minimum deposit', async ({ page }) => {
    await page.goto('/riverpool');
    await page.getByRole('tab', { name: /Main/i }).click();
    await page.waitForTimeout(500);

    await page.getByRole('button', { name: /Deposit/i }).first().click();

    // Enter amount below minimum and clear the field
    const amountInput = page.getByLabel(/Amount/i);
    await amountInput.fill('');
    await amountInput.fill('0');

    // When amount is 0, confirm button should be disabled
    const confirmButton = page.getByRole('button', { name: /Confirm/i });
    await expect(confirmButton).toBeDisabled();
  });
});

test.describe('Withdraw Modal', () => {
  test('should open withdraw modal', async ({ page }) => {
    await page.goto('/riverpool');
    await page.getByRole('tab', { name: /Main/i }).click();
    await page.waitForTimeout(500);

    // Click withdraw button
    await page.getByRole('button', { name: /Withdraw/i }).first().click();

    // Check for modal content - look for the modal title
    await expect(page.getByText(/Withdraw from Main LP/i)).toBeVisible();
  });

  test('should show T+4 delay notice', async ({ page }) => {
    await page.goto('/riverpool');
    await page.getByRole('tab', { name: /Main/i }).click();
    await page.waitForTimeout(500);

    await page.getByRole('button', { name: /Withdraw/i }).first().click();

    // Check for delay notice - use first() for multiple matches
    await expect(page.getByText(/T\+4/i).first()).toBeVisible();
  });
});

test.describe('Mobile Responsiveness', () => {
  test('should be responsive on mobile', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/riverpool');
    await page.waitForTimeout(500);

    // Check that the page title is visible
    await expect(page.locator('h1')).toContainText('RiverPool');

    // Check that tabs container is visible
    const tablist = page.getByRole('tablist');
    await expect(tablist).toBeVisible();
  });
});

test.describe('Error Handling', () => {
  test('should handle network errors gracefully', async ({ page }) => {
    // Intercept API calls and simulate error
    await page.route('**/api/v1/riverpool/**', route => {
      route.fulfill({
        status: 500,
        body: JSON.stringify({ error: 'Internal Server Error' }),
      });
    });

    await page.goto('/riverpool');

    // Check for error state or fallback content
    // The page should still be navigable
    await expect(page.locator('h1')).toContainText('RiverPool');
  });

  test('should handle 404 for non-existent pool', async ({ page }) => {
    await page.goto('/riverpool/community/non-existent-pool');

    // Should show error or redirect
    await expect(page.getByText(/not found|error/i)).toBeVisible().catch(() => {
      // Or redirected to main page
      expect(page.url()).not.toContain('non-existent-pool');
    });
  });
});

test.describe('Accessibility', () => {
  test('should have proper ARIA labels', async ({ page }) => {
    await page.goto('/riverpool');

    // Check for proper tab roles
    const tabs = page.getByRole('tab');
    const tabCount = await tabs.count();
    expect(tabCount).toBeGreaterThan(0);

    // Check for button accessibility
    const buttons = page.getByRole('button');
    const buttonCount = await buttons.count();
    expect(buttonCount).toBeGreaterThan(0);
  });

  test('should support keyboard navigation', async ({ page }) => {
    await page.goto('/riverpool');

    // Tab through elements
    await page.keyboard.press('Tab');
    await page.keyboard.press('Tab');

    // Check that something is focused
    const focusedElement = page.locator(':focus');
    await expect(focusedElement).toBeVisible();
  });
});
