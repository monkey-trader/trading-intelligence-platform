import { PositionFactory } from '@/domain/position/factories/PositionFactory';
import type { PositionRecord } from '@/domain/position/interfaces/PositionRecord';
import { UnexpectedBingXResponseError } from '@/infrastructure/bingx/errors';

type RawObject = Readonly<Record<string, unknown>>;

function isRawObject(value: unknown): value is RawObject {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

/** Erster vorhandener Wert unter einem der Feldnamen (BingX benennt Felder je nach Endpoint anders). */
function pick(raw: RawObject, ...fieldNames: readonly string[]): unknown {
  for (const fieldName of fieldNames) {
    if (raw[fieldName] !== undefined) return raw[fieldName];
  }
  return undefined;
}

function requireString(raw: RawObject, ...fieldNames: readonly string[]): string {
  const value = pick(raw, ...fieldNames);
  if (typeof value !== 'string') throw unexpectedField(fieldNames, 'String');
  return value;
}

function requireNumber(raw: RawObject, ...fieldNames: readonly string[]): number {
  const value = pick(raw, ...fieldNames);
  if (typeof value !== 'number') throw unexpectedField(fieldNames, 'Zahl');
  return value;
}

function requireBoolean(raw: RawObject, ...fieldNames: readonly string[]): boolean {
  const value = pick(raw, ...fieldNames);
  if (typeof value !== 'boolean') throw unexpectedField(fieldNames, 'Boolean');
  return value;
}

function unexpectedField(fieldNames: readonly string[], expectedType: string): Error {
  return new UnexpectedBingXResponseError(
    `positionHistory: Feld ${fieldNames.join('/')} fehlt oder ist kein ${expectedType}.`,
  );
}

function mapRawPosition(rawPosition: unknown): PositionRecord {
  if (!isRawObject(rawPosition)) {
    throw new UnexpectedBingXResponseError(
      'positionHistory enthält einen Eintrag, der kein Objekt ist.',
    );
  }
  const position = PositionFactory.create({
    positionId: requireString(rawPosition, 'positionId'),
    symbol: requireString(rawPosition, 'symbol'),
    side: requireString(rawPosition, 'positionSide', 'side'),
    isolated: requireBoolean(rawPosition, 'isolated'),
    openTime: requireNumber(rawPosition, 'openTime'),
    closeTime: requireNumber(rawPosition, 'updateTime', 'closeTime'),
    avgOpenPrice: requireString(rawPosition, 'avgPrice', 'avgOpenPrice'),
    avgClosePrice: requireString(rawPosition, 'avgClosePrice'),
    realisedProfit: requireString(rawPosition, 'realisedProfit', 'realizedProfit'),
    netProfit: requireString(rawPosition, 'netProfit'),
    positionAmount: requireString(rawPosition, 'positionAmt', 'positionAmount'),
    closedAmount: requireString(rawPosition, 'closePositionAmt', 'closedAmount'),
    commission: requireString(rawPosition, 'positionCommission', 'commission'),
    totalFunding: requireString(rawPosition, 'totalFunding'),
    leverage: requireNumber(rawPosition, 'leverage'),
    fullyClosed: requireBoolean(rawPosition, 'closeAllPositions', 'fullyClosed'),
  });
  return { position, rawPayload: JSON.stringify(rawPosition) };
}

/**
 * Mappt das Feld `data` von /openApi/swap/v1/trade/positionHistory auf Positionen.
 * Fehlt `positionHistory` oder ist es kein Array, wird geworfen und nie still [] geliefert.
 */
export function mapPositionHistoryResponse(data: unknown): PositionRecord[] {
  if (!isRawObject(data) || !Array.isArray(data.positionHistory)) {
    throw new UnexpectedBingXResponseError(
      'Antwort von positionHistory enthält kein Array positionHistory.',
    );
  }
  return data.positionHistory.map(mapRawPosition);
}
