import type { OpenPosition } from '@/domain/position/entities/OpenPosition';
import { OpenPositionFactory } from '@/domain/position/factories/OpenPositionFactory';
import { UnexpectedBingXResponseError } from '@/infrastructure/bingx/errors';
import {
  isRawObject,
  pick,
  type RawObject,
  requireBoolean,
  requireNumber,
  requireString,
  withEndpointContext,
} from '@/infrastructure/bingx/rawFields';

/**
 * Endliche Zahl als einfache Dezimalzahl. Basis ist String(n), die kürzeste Darstellung, die
 * denselben Wert ergibt. Eine Exponentenform (1e-7, 1e+21) wird nur umgeschrieben, ohne zu runden.
 */
export function toPlainDecimal(value: number): string {
  const text = String(value);
  const exponentMatch = /^(-?)(\d+)(?:\.(\d+))?e([+-]\d+)$/.exec(text);
  if (!exponentMatch) return text;
  const [, sign = '', integerDigits = '', fractionDigits = '', exponentText = '0'] = exponentMatch;
  const digits = integerDigits + fractionDigits;
  const pointPosition = integerDigits.length + Number(exponentText);
  if (pointPosition <= 0) return `${sign}0.${'0'.repeat(-pointPosition)}${digits}`;
  if (pointPosition >= digits.length) {
    return `${sign}${digits}${'0'.repeat(pointPosition - digits.length)}`;
  }
  return `${sign}${digits.slice(0, pointPosition)}.${digits.slice(pointPosition)}`;
}

/**
 * `liquidationPrice` kommt von BingX als number, alle anderen Beträge als String.
 * Ein String wird übernommen, eine endliche number verlustfrei als Dezimal-String.
 */
function requireLiquidationPrice(raw: RawObject): string {
  const value = pick(raw, 'liquidationPrice');
  if (typeof value === 'string') return value;
  if (typeof value === 'number' && Number.isFinite(value)) return toPlainDecimal(value);
  throw new UnexpectedBingXResponseError(
    'Feld liquidationPrice fehlt oder ist weder String noch endliche Zahl.',
  );
}

function mapRawOpenPosition(rawPosition: unknown): OpenPosition {
  if (!isRawObject(rawPosition)) {
    throw new UnexpectedBingXResponseError(
      'user/positions enthält einen Eintrag, der kein Objekt ist.',
    );
  }
  return OpenPositionFactory.create({
    positionId: requireString(rawPosition, 'positionId'),
    symbol: requireString(rawPosition, 'symbol'),
    side: requireString(rawPosition, 'positionSide'),
    isolated: requireBoolean(rawPosition, 'isolated'),
    positionAmount: requireString(rawPosition, 'positionAmt'),
    avgOpenPrice: requireString(rawPosition, 'avgPrice'),
    markPrice: requireString(rawPosition, 'markPrice'),
    leverage: requireNumber(rawPosition, 'leverage'),
    initialMargin: requireString(rawPosition, 'initialMargin'),
    margin: requireString(rawPosition, 'margin'),
    positionValue: requireString(rawPosition, 'positionValue'),
    unrealizedProfit: requireString(rawPosition, 'unrealizedProfit'),
    realisedProfit: requireString(rawPosition, 'realisedProfit'),
    pnlRatio: requireString(rawPosition, 'pnlRatio'),
    liquidationPrice: requireLiquidationPrice(rawPosition),
    openTime: requireNumber(rawPosition, 'createTime'),
    updateTime: requireNumber(rawPosition, 'updateTime'),
  });
}

/**
 * Mappt das Feld `data` von /openApi/swap/v2/user/positions auf offene Positionen.
 * `null` gilt als "keine offene Position". Jede andere Form außer einem Array wird geworfen.
 */
export function mapOpenPositionsResponse(data: unknown): OpenPosition[] {
  if (data === null) return [];
  if (!Array.isArray(data)) {
    throw new UnexpectedBingXResponseError('Antwort von user/positions ist kein Array.');
  }
  return data.map((rawPosition: unknown) =>
    withEndpointContext('user/positions', () => mapRawOpenPosition(rawPosition)),
  );
}
