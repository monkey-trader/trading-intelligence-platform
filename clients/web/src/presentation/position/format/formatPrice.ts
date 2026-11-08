import { formatDecimal } from '@/presentation/position/format/formatDecimal';
import { decimalSign } from '@/presentation/position/format/decimalString';
import { MISSING_VALUE } from '@/presentation/position/format/formatRatioAsPercent';
import { pnlTone, type PnlTone } from '@/presentation/position/format/pnlTone';

const PERCENT_DIGITS = 2;
const PERCENT_FORMAT = {
  maximumFractionDigits: PERCENT_DIGITS,
  signDisplay: 'exceptZero',
} as const;

/** Kurs wie der Einstieg: exakt, alle Nachkommastellen, de-CH. */
export function formatPrice(price: string | null): string {
  return price === null ? MISSING_VALUE : formatDecimal(price);
}

/** 24h-Änderung in Prozent mit Vorzeichen; die Tönung folgt dem gerundeten Wert. */
export function formatChange24h(percent: string | null): { text: string; tone: PnlTone } {
  if (percent === null) return { text: MISSING_VALUE, tone: 'neutral' };
  return {
    text: `${formatDecimal(percent, PERCENT_FORMAT)}\u00a0%`,
    tone: pnlTone(percent, PERCENT_DIGITS),
  };
}

/** label: 'erreicht' (Abstand <= 0), 'offen' (> 0), null ohne Wert. */
export function formatDistance(percent: string | null): {
  text: string;
  label: 'erreicht' | 'offen' | null;
} {
  if (percent === null) return { text: MISSING_VALUE, label: null };
  return {
    text: `${formatDecimal(percent, PERCENT_FORMAT)}\u00a0%`,
    label: decimalSign(percent) > 0 ? 'offen' : 'erreicht',
  };
}
