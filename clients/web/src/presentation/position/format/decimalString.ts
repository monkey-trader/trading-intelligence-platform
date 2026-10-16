const DECIMAL_PATTERN = /^-?\d+(\.\d+)?$/;

/** Prüft, ob ein String eine Dezimalzahl ist, die Intl.NumberFormat exakt verarbeitet. */
export function isDecimalString(value: string): value is `${number}` {
  return DECIMAL_PATTERN.test(value);
}

/** Vorzeichen einer Dezimalzahl als String, ohne Umweg über Number. */
export function decimalSign(value: string): -1 | 0 | 1 {
  if (!isDecimalString(value)) return 0;
  const isZero = /^-?0+(\.0+)?$/.test(value);
  if (isZero) return 0;
  return value.startsWith('-') ? -1 : 1;
}

/**
 * Ob der Wert auf `fractionDigits` Stellen (kaufmännisch) gerundet 0 ergibt, rein über den String.
 * Das ist der Fall, wenn der Vorkommateil 0 ist, die ersten Nachkommastellen 0 sind und die
 * nächste Ziffer kleiner als 5 ist.
 */
export function isZeroWhenRounded(value: string, fractionDigits: number): boolean {
  if (!isDecimalString(value)) return false;
  const unsignedValue = value.startsWith('-') ? value.slice(1) : value;
  const [integerPart = '', fractionPart = ''] = unsignedValue.split('.');
  if (!/^0+$/.test(integerPart)) return false;
  const keptDigits = fractionPart.slice(0, fractionDigits);
  const firstDroppedDigit = fractionPart.charAt(fractionDigits);
  return /^0*$/.test(keptDigits) && (firstDroppedDigit === '' || firstDroppedDigit < '5');
}

/** Anzahl Nachkommastellen im String ("0.9600" → 4). */
export function fractionDigitCount(value: string): number {
  const separatorIndex = value.indexOf('.');
  return separatorIndex === -1 ? 0 : value.length - separatorIndex - 1;
}

/** Verschiebt den Dezimalpunkt nach rechts (×10^places), rein über den String. */
export function shiftDecimalPointRight(value: string, places: number): string {
  const isNegative = value.startsWith('-');
  const unsignedValue = isNegative ? value.slice(1) : value;
  const [integerPart = '0', fractionPart = ''] = unsignedValue.split('.');
  const paddedFraction = fractionPart.padEnd(places, '0');
  const shiftedInteger = (integerPart + paddedFraction.slice(0, places)).replace(/^0+(?=\d)/, '');
  const remainingFraction = paddedFraction.slice(places);
  const shifted = remainingFraction ? `${shiftedInteger}.${remainingFraction}` : shiftedInteger;
  return isNegative ? `-${shifted}` : shifted;
}
