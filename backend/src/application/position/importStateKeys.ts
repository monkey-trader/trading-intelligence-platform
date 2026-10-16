// Schlüssel der State-Tabelle, die mehrere Use Cases teilen.

/** Präfix des Import-Stands pro Symbol ("importiert bis"). */
export const POSITION_STATE_KEY_PREFIX = 'position_history_until:';

export function positionStateKey(symbol: string): string {
  return `${POSITION_STATE_KEY_PREFIX}${symbol}`;
}

/** Merkt sich einen Verdacht auf still leere Antwort über Läufe hinweg (Wert: erstmals gesehen). */
export function suspiciousStateKey(symbol: string): string {
  return `position_suspicious:${symbol}`;
}

/** Fehlermeldung für das Ergebnis. Unsere Fehlertexte enthalten nie Secrets. */
export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}
