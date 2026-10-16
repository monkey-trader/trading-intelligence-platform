import { DivisionByZeroError } from '@/domain/shared/decimal/DivisionByZeroError';
import { InvalidDecimalError } from '@/domain/shared/decimal/InvalidDecimalError';

const DECIMAL_PATTERN = /^-?\d+(\.\d+)?$/;
const DEFAULT_DIVISION_SCALE = 8;

/** Dezimalzahl als ganzzahliger Wert `units / 10^scale`, damit ohne Float gerechnet wird. */
type ScaledDecimal = Readonly<{ units: bigint; scale: number }>;

/** Exakte Summe zweier Dezimal-Strings, normalisiert. */
export function addDecimals(left: string, right: string): string {
  const [leftUnits, rightUnits, scale] = alignScales(parseDecimal(left), parseDecimal(right));
  return formatDecimal({ units: leftUnits + rightUnits, scale });
}

/** Exakte Differenz zweier Dezimal-Strings, normalisiert. */
export function subtractDecimals(left: string, right: string): string {
  const [leftUnits, rightUnits, scale] = alignScales(parseDecimal(left), parseDecimal(right));
  return formatDecimal({ units: leftUnits - rightUnits, scale });
}

/**
 * Quotient zweier Dezimal-Strings mit `scale` Nachkommastellen.
 * Gerundet wird kaufmännisch (die genaue Hälfte weg von 0), das Ergebnis ist normalisiert.
 */
export function divideDecimals(
  dividend: string,
  divisor: string,
  scale: number = DEFAULT_DIVISION_SCALE,
): string {
  if (!Number.isInteger(scale) || scale < 0) {
    throw new InvalidDecimalError(`Ungültige Skala: ${String(scale)}`);
  }
  const parsedDividend = parseDecimal(dividend);
  const parsedDivisor = parseDecimal(divisor);
  if (parsedDivisor.units === 0n) {
    throw new DivisionByZeroError(`Division durch 0: "${dividend}" / "${divisor}"`);
  }
  // dividend / divisor = (a / 10^sa) / (b / 10^sb) = a · 10^sb / (b · 10^sa)
  const numerator = parsedDividend.units * 10n ** BigInt(parsedDivisor.scale + scale);
  const denominator = parsedDivisor.units * 10n ** BigInt(parsedDividend.scale);
  const isNegative = numerator < 0n !== denominator < 0n;
  const absoluteNumerator = absolute(numerator);
  const absoluteDenominator = absolute(denominator);
  const quotient = absoluteNumerator / absoluteDenominator;
  const remainder = absoluteNumerator % absoluteDenominator;
  const roundedQuotient = remainder * 2n >= absoluteDenominator ? quotient + 1n : quotient;
  return formatDecimal({ units: isNegative ? -roundedQuotient : roundedQuotient, scale });
}

function parseDecimal(rawAmount: string): ScaledDecimal {
  if (!DECIMAL_PATTERN.test(rawAmount)) {
    throw new InvalidDecimalError(`Ungültiger Betrag: "${rawAmount}"`);
  }
  const [integerPart = '', fractionPart = ''] = rawAmount.split('.');
  return { units: BigInt(integerPart + fractionPart), scale: fractionPart.length };
}

/** Bringt beide Werte auf die größere Skala. */
function alignScales(left: ScaledDecimal, right: ScaledDecimal): [bigint, bigint, number] {
  const scale = Math.max(left.scale, right.scale);
  return [rescale(left, scale), rescale(right, scale), scale];
}

function rescale(decimal: ScaledDecimal, targetScale: number): bigint {
  return decimal.units * 10n ** BigInt(targetScale - decimal.scale);
}

function absolute(value: bigint): bigint {
  return value < 0n ? -value : value;
}

/** Formatiert ohne unnötige Nullen am Ende und ohne "-0". */
function formatDecimal({ units, scale }: ScaledDecimal): string {
  if (units === 0n) return '0';
  const sign = units < 0n ? '-' : '';
  const digits = absolute(units)
    .toString()
    .padStart(scale + 1, '0');
  const integerPart = digits.slice(0, digits.length - scale);
  const fractionPart = digits.slice(digits.length - scale).replace(/0+$/, '');
  return fractionPart === '' ? `${sign}${integerPart}` : `${sign}${integerPart}.${fractionPart}`;
}
