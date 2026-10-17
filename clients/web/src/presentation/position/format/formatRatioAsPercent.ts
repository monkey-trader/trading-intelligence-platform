import {
  isDecimalString,
  shiftDecimalPointRight,
} from '@/presentation/position/format/decimalString';
import { formatDecimal } from '@/presentation/position/format/formatDecimal';

/** Platzhalter, wenn kein Wert vorliegt. */
export const MISSING_VALUE = '–';

/** Geschütztes Leerzeichen zwischen Zahl und %, damit es nicht umbricht. */
const NO_BREAK_SPACE = '\u00a0';

export type RatioFormatOptions = Readonly<{
  /** 'exceptZero' zeigt positive Prozente mit "+" (z.B. PnL %), Standard ist 'negative'. */
  signDisplay?: 'negative' | 'exceptZero';
}>;

/** Zeigt ein Verhältnis (0.25) als Prozent mit zwei Nachkommastellen (25.00 %), ohne Number. */
export function formatRatioAsPercent(
  ratio: string | null,
  options: RatioFormatOptions = {},
): string {
  if (ratio === null || !isDecimalString(ratio)) return MISSING_VALUE;
  const percentValue = shiftDecimalPointRight(ratio, 2);
  const formattedValue = formatDecimal(percentValue, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
    signDisplay: options.signDisplay ?? 'negative',
  });
  return `${formattedValue}${NO_BREAK_SPACE}%`;
}
