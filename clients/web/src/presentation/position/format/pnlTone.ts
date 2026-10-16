import { decimalSign, isZeroWhenRounded } from '@/presentation/position/format/decimalString';

export type PnlTone = 'positive' | 'negative' | 'neutral';

/**
 * Tönung eines PnL-Werts nach seinem Vorzeichen; 0 und -0 sind neutral.
 * Mit `fractionDigits` zählt der angezeigte, gerundete Wert: -0.004 auf 2 Stellen ist neutral,
 * damit nie ein rotes "0.00" erscheint.
 */
export function pnlTone(value: string, fractionDigits?: number): PnlTone {
  if (fractionDigits !== undefined && isZeroWhenRounded(value, fractionDigits)) return 'neutral';
  const sign = decimalSign(value);
  if (sign > 0) return 'positive';
  if (sign < 0) return 'negative';
  return 'neutral';
}
