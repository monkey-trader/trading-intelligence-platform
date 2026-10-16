import { PositionFactory } from '@/domain/position/factories/PositionFactory';
import type { PositionRecord } from '@/domain/position/interfaces/PositionRecord';
import { UnexpectedBingXResponseError } from '@/infrastructure/bingx/errors';
import {
  isRawObject,
  requireBoolean,
  requireNumber,
  requireString,
  withEndpointContext,
} from '@/infrastructure/bingx/rawFields';

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
  return data.positionHistory.map((rawPosition: unknown) =>
    withEndpointContext('positionHistory', () => mapRawPosition(rawPosition)),
  );
}
