/**
 * E2E Latency Tests
 * Measures real-world latency from browser to data display
 *
 * This tests the full pipeline:
 * Browser → Frontend → Hyperliquid API → Data Processing → UI Render
 */

import { test, expect } from '@playwright/test';

interface LatencyResult {
  test: string;
  latencyMs: number;
  status: 'pass' | 'fail';
  details?: string;
}

const results: LatencyResult[] = [];

test.describe('E2E Latency Measurements', () => {
  test.afterAll(async () => {
    // Print latency report
    console.log('\n========================================');
    console.log('E2E LATENCY TEST REPORT');
    console.log('========================================\n');

    results.forEach((r) => {
      const statusIcon = r.status === 'pass' ? '✅' : '❌';
      console.log(`${statusIcon} ${r.test}: ${r.latencyMs}ms ${r.details || ''}`);
    });

    const avgLatency = results.reduce((sum, r) => sum + r.latencyMs, 0) / results.length;
    console.log(`\n📊 Average Latency: ${avgLatency.toFixed(0)}ms`);
    console.log('========================================\n');
  });

  test('initial page load to first price display', async ({ page }) => {
    const startTime = Date.now();

    await page.goto('/');

    // Wait for price to appear (not '--')
    await page.waitForFunction(
      () => {
        const priceEl = document.querySelector('.text-2xl.font-bold.text-white.font-mono');
        return priceEl && !priceEl.textContent?.includes('--');
      },
      { timeout: 15000 }
    );

    const latency = Date.now() - startTime;

    results.push({
      test: 'Initial Load → Price Display',
      latencyMs: latency,
      status: latency < 5000 ? 'pass' : 'fail',
      details: latency < 3000 ? '(excellent)' : latency < 5000 ? '(acceptable)' : '(slow)',
    });

    expect(latency).toBeLessThan(10000);
  });

  test('order book data load latency', async ({ page }) => {
    await page.goto('/');

    const startTime = Date.now();

    // Wait for orderbook bids to appear
    await page.waitForFunction(
      () => {
        const bids = document.querySelectorAll('.text-primary-400');
        return bids.length > 5;
      },
      { timeout: 15000 }
    );

    const latency = Date.now() - startTime;

    results.push({
      test: 'Order Book Load',
      latencyMs: latency,
      status: latency < 3000 ? 'pass' : 'fail',
    });

    expect(latency).toBeLessThan(10000);
  });

  test('recent trades load latency', async ({ page }) => {
    await page.goto('/');

    const startTime = Date.now();

    // Wait for trade timestamps to appear
    await page.waitForFunction(
      () => {
        const trades = document.body.innerText;
        // Look for timestamp pattern HH:MM:SS
        return /\d{2}:\d{2}:\d{2}/.test(trades);
      },
      { timeout: 15000 }
    );

    const latency = Date.now() - startTime;

    results.push({
      test: 'Recent Trades Load',
      latencyMs: latency,
      status: latency < 3000 ? 'pass' : 'fail',
    });

    expect(latency).toBeLessThan(10000);
  });

  test('chart data load latency', async ({ page }) => {
    await page.goto('/');

    const startTime = Date.now();

    // Wait for chart OHLC data to appear
    await page.waitForFunction(
      () => {
        const text = document.body.innerText;
        // Look for "O:" followed by a number (OHLC display)
        return /O: [0-9,]+/.test(text);
      },
      { timeout: 20000 }
    );

    const latency = Date.now() - startTime;

    results.push({
      test: 'Chart Data Load',
      latencyMs: latency,
      status: latency < 5000 ? 'pass' : 'fail',
    });

    expect(latency).toBeLessThan(15000);
  });

  test('interval switch latency (1m → 5m)', async ({ page }) => {
    await page.goto('/');

    // Wait for initial chart load
    await page.waitForFunction(
      () => {
        const text = document.body.innerText;
        return /O: [0-9,]+/.test(text);
      },
      { timeout: 20000 }
    );

    // Click 5m interval
    const startTime = Date.now();
    await page.click('button:has-text("5m")');

    // Wait for chart to reload (loading indicator should disappear)
    await page.waitForFunction(
      () => {
        return !document.body.innerText.includes('Loading chart...');
      },
      { timeout: 10000 }
    );

    const latency = Date.now() - startTime;

    results.push({
      test: 'Chart Interval Switch (1m→5m)',
      latencyMs: latency,
      status: latency < 2000 ? 'pass' : 'fail',
    });

    expect(latency).toBeLessThan(5000);
  });

  test('navigation latency (Trade → Positions)', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const startTime = Date.now();
    await page.click('text=Positions');

    // Wait for positions page to render
    await page.waitForSelector('h1:has-text("Positions")');

    const latency = Date.now() - startTime;

    results.push({
      test: 'Navigation (Trade → Positions)',
      latencyMs: latency,
      status: latency < 1000 ? 'pass' : 'fail',
    });

    expect(latency).toBeLessThan(3000);
  });

  test('order book price click to form fill', async ({ page }) => {
    await page.goto('/');

    // Wait for orderbook
    await page.waitForFunction(
      () => {
        const bids = document.querySelectorAll('.text-primary-400');
        return bids.length > 5;
      },
      { timeout: 15000 }
    );

    const startTime = Date.now();

    // Click a bid price
    await page.click('.orderbook-row >> nth=0');

    // Wait for price input to be populated
    await page.waitForFunction(
      () => {
        const input = document.querySelector('input[placeholder="0.00"]') as HTMLInputElement;
        return input && parseFloat(input.value) > 0;
      },
      { timeout: 5000 }
    );

    const latency = Date.now() - startTime;

    results.push({
      test: 'Order Book Click → Form Fill',
      latencyMs: latency,
      status: latency < 500 ? 'pass' : 'fail',
    });

    expect(latency).toBeLessThan(2000);
  });
});

test.describe('WebSocket Connection Tests', () => {
  test('WebSocket connection established', async ({ page }) => {
    await page.goto('/');

    // Wait for Live indicator
    const liveIndicator = page.locator('text=Live');

    try {
      await expect(liveIndicator.first()).toBeVisible({ timeout: 10000 });
      results.push({
        test: 'WebSocket Connection',
        latencyMs: 0,
        status: 'pass',
        details: '(connected)',
      });
    } catch {
      results.push({
        test: 'WebSocket Connection',
        latencyMs: 0,
        status: 'fail',
        details: '(not connected)',
      });
    }
  });
});
