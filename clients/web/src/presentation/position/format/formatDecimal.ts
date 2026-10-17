import { fractionDigitCount, isDecimalString } from '@/presentation/position/format/decimalString';

export const DISPLAY_LOCALE = 'de-CH';

/** Intl erlaubt höchstens 100 Nachkommastellen. */
const MAX_FRACTION_DIGITS = 100;

export type DecimalFormatOptions = Readonly<{
  minimumFractionDigits?: number;
  maximumFractionDigits?: number;
  /**
   * 'negative' (Standard): nur negative Werte mit Vorzeichen.
   * 'exceptZero': zusätzlich "+" für positive Werte (PnL), gerundet 0 bleibt ohne Vorzeichen.
   */
  signDisplay?: 'negative' | 'exceptZero';
}>;

/**
 * Formatiert einen Dezimal-String im Format de-CH (1'234.5).
 * Intl.NumberFormat bekommt den String direkt und rechnet damit exakt, nie über Number.
 * Ohne Optionen bleiben alle Nachkommastellen erhalten, Nullen am Ende fallen weg.
 * Gerundet wird kaufmännisch (half-expand).
 */
export function formatDecimal(value: string, options: DecimalFormatOptions = {}): string {
  if (!isDecimalString(value)) return value;
  const maximumFractionDigits =
    options.maximumFractionDigits ?? Math.min(fractionDigitCount(value), MAX_FRACTION_DIGITS);
  const minimumFractionDigits = Math.min(options.minimumFractionDigits ?? 0, maximumFractionDigits);
  return new Intl.NumberFormat(DISPLAY_LOCALE, {
    minimumFractionDigits,
    maximumFractionDigits,
    roundingMode: 'halfExpand',
    // Vorzeichen nur, wenn der gerundete Wert nicht 0 ist: kein "-0.00" und kein "+0.00".
    signDisplay: options.signDisplay ?? 'negative',
  }).format(value);
}
