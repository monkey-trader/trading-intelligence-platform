// Hilfen zum defensiven Lesen roher BingX-Objekte. Fehlende oder falsch typisierte Felder
// führen zu UnexpectedBingXResponseError, nie zu stillen Standardwerten.
import { UnexpectedBingXResponseError } from '@/infrastructure/bingx/errors';

export type RawObject = Readonly<Record<string, unknown>>;

export function isRawObject(value: unknown): value is RawObject {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

/** Erster vorhandener Wert unter einem der Feldnamen (BingX benennt Felder je nach Endpoint anders). */
export function pick(raw: RawObject, ...fieldNames: readonly string[]): unknown {
  for (const fieldName of fieldNames) {
    if (raw[fieldName] !== undefined) return raw[fieldName];
  }
  return undefined;
}

export function requireString(raw: RawObject, ...fieldNames: readonly string[]): string {
  const value = pick(raw, ...fieldNames);
  if (typeof value !== 'string') throw unexpectedField(fieldNames, 'String');
  return value;
}

export function requireNumber(raw: RawObject, ...fieldNames: readonly string[]): number {
  const value = pick(raw, ...fieldNames);
  if (typeof value !== 'number') throw unexpectedField(fieldNames, 'Zahl');
  return value;
}

export function requireBoolean(raw: RawObject, ...fieldNames: readonly string[]): boolean {
  const value = pick(raw, ...fieldNames);
  if (typeof value !== 'boolean') throw unexpectedField(fieldNames, 'Boolean');
  return value;
}

/**
 * Führt `mapEntry` aus und stellt Feldfehlern den Endpunkt voran, damit im Log klar ist,
 * welche BingX-Antwort nicht passte (z.B. "user/positions: Feld markPrice fehlt ...").
 */
export function withEndpointContext<Result>(endpoint: string, mapEntry: () => Result): Result {
  try {
    return mapEntry();
  } catch (error: unknown) {
    if (error instanceof UnexpectedBingXResponseError && !error.message.startsWith(endpoint)) {
      throw new UnexpectedBingXResponseError(`${endpoint}: ${error.message}`);
    }
    throw error;
  }
}

function unexpectedField(fieldNames: readonly string[], expectedType: string): Error {
  return new UnexpectedBingXResponseError(
    `Feld ${fieldNames.join('/')} fehlt oder ist kein ${expectedType}.`,
  );
}
