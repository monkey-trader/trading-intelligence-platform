// Macht den README-Screenshot gegen eine laufende UI mit der Fake-API (scripts/mock-api.mjs).
// Aufruf: node scripts/screenshot.mjs [url] [ausgabe.png]
import { chromium } from '@playwright/test';

const url = process.argv[2] ?? 'http://127.0.0.1:5174';
const output = process.argv[3] ?? '../../docs/images/app.png';
const chartOutput = process.argv[4] ?? '../../docs/images/app-chart.png';

const browser = await chromium.launch();
const page = await browser.newPage({
  viewport: { width: 1440, height: 820 },
  deviceScaleFactor: 2,
  colorScheme: 'dark',
});
await page.goto(url);
await page.getByText('BTC').first().waitFor();
await page.waitForTimeout(500);
await page.screenshot({ path: output });

// Zweites Bild: Chart der BTC-Position mit Entry, SL, TP und R-Linien.
await page.getByRole('button', { name: /Chart/ }).first().click();
await page.locator('canvas').first().waitFor();
await page.waitForTimeout(1500);
await page.mouse.move(0, 0); // kein Crosshair im Bild
const panel = page.getByText('BTC-USDT Long').locator('xpath=ancestor::*[.//canvas][1]');
await panel.screenshot({ path: chartOutput });
await browser.close();
