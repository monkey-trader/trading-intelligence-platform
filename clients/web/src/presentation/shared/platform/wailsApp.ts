/** Attribut am <html>, das die Wails-Desktop-Hülle markiert. Ohne Attribut läuft die Seite im Browser. */
export const WAILS_APP_ATTRIBUTE = 'data-desktop';
export const WAILS_APP_VALUE = 'wails';

export type PlatformScope = Readonly<{
  /** true, wenn Wails das Objekt window.runtime injiziert hat. */
  hasWailsRuntime: boolean;
  userAgent: string;
}>;

/** Erkennt die Wails-Hülle: window.runtime oder der User-Agent-Zusatz "wails.io" (macOS, Wails v2). */
export function isWailsApp(scope: PlatformScope): boolean {
  if (scope.hasWailsRuntime) return true;
  return /\bwails\b/i.test(scope.userAgent);
}

/** Setzt data-desktop="wails" am Wurzelelement, wenn die Seite in der Wails-App läuft. */
export function markWailsApp(root: HTMLElement, scope: PlatformScope): void {
  if (isWailsApp(scope)) {
    root.setAttribute(WAILS_APP_ATTRIBUTE, WAILS_APP_VALUE);
  }
}
