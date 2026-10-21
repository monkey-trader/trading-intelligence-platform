import { describe, expect, it } from 'vitest';
import {
  isWailsApp,
  markWailsApp,
  WAILS_APP_ATTRIBUTE,
} from '@/presentation/shared/platform/wailsApp';

const BROWSER_USER_AGENT =
  'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15';
const WAILS_USER_AGENT =
  'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) wails.io';

describe('isWailsApp', () => {
  it('erkennt einen normalen Browser nicht als Wails-App', () => {
    expect(isWailsApp({ hasWailsRuntime: false, userAgent: BROWSER_USER_AGENT })).toBe(false);
  });

  it('erkennt die Wails-App am User-Agent "wails.io" (macOS)', () => {
    expect(isWailsApp({ hasWailsRuntime: false, userAgent: WAILS_USER_AGENT })).toBe(true);
  });

  it('erkennt die Wails-App am injizierten window.runtime', () => {
    expect(isWailsApp({ hasWailsRuntime: true, userAgent: BROWSER_USER_AGENT })).toBe(true);
  });
});

describe('markWailsApp', () => {
  it('setzt das Attribut am Wurzelelement nur in der Wails-App', () => {
    const root = document.createElement('html');

    markWailsApp(root, { hasWailsRuntime: true, userAgent: BROWSER_USER_AGENT });

    expect(root.getAttribute(WAILS_APP_ATTRIBUTE)).toBe('wails');
  });

  it('lässt das Wurzelelement im Browser unverändert', () => {
    const root = document.createElement('html');

    markWailsApp(root, { hasWailsRuntime: false, userAgent: BROWSER_USER_AGENT });

    expect(root.hasAttribute(WAILS_APP_ATTRIBUTE)).toBe(false);
  });
});
