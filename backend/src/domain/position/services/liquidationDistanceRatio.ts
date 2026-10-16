import type { OpenPosition } from '@/domain/position/entities/OpenPosition';
import { divideDecimals, subtractDecimals } from '@/domain/shared/decimal/decimalArithmetic';

const RATIO_SCALE = 8;

/**
 * Relativer Abstand des Mark-Preises zum Liquidationspreis mit 8 Nachkommastellen.
 * LONG: (mark − liq) / mark, SHORT: (liq − mark) / mark. Negativ, wenn Mark schon jenseits liegt.
 * Ohne Liquidationspreis ("0") oder bei Mark-Preis 0 gibt es keinen Abstand (null).
 * Negative Preise sind von BingX nicht plausibel und ergeben ebenfalls null statt einer
 * irreführenden Zahl (z.B. umgedrehtes Vorzeichen).
 */
export function liquidationDistanceRatio(
  side: OpenPosition['side'],
  markPrice: string,
  liquidationPrice: string,
): string | null {
  // Beide Preise immer prüfen, damit ein ungültiger Wert nie still als null durchgeht.
  const hasNoLiquidationPrice = !isPositive(liquidationPrice);
  const hasNoMarkPrice = !isPositive(markPrice);
  if (hasNoLiquidationPrice || hasNoMarkPrice) return null;
  const distance =
    side === 'LONG'
      ? subtractDecimals(markPrice, liquidationPrice)
      : subtractDecimals(liquidationPrice, markPrice);
  return divideDecimals(distance, markPrice, RATIO_SCALE);
}

/** Echt größer als 0 (auch "0.000" zählt als 0). Ungültige Eingaben werfen InvalidDecimalError. */
function isPositive(amount: string): boolean {
  const normalized = subtractDecimals(amount, '0');
  return normalized !== '0' && !normalized.startsWith('-');
}
