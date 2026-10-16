import { describe, expect, it } from 'vitest';
import {
  addDecimals,
  divideDecimals,
  subtractDecimals,
} from '@/domain/shared/decimal/decimalArithmetic';
import { DecimalError } from '@/domain/shared/decimal/DecimalError';
import { DivisionByZeroError } from '@/domain/shared/decimal/DivisionByZeroError';
import { InvalidDecimalError } from '@/domain/shared/decimal/InvalidDecimalError';

function captureError(action: () => unknown): unknown {
  try {
    action();
  } catch (error) {
    return error;
  }
  throw new Error('Es wurde ein Fehler erwartet, aber keiner geworfen.');
}

describe('addDecimals', () => {
  it.each([
    ['1.5', '2.25', '3.75'],
    ['0.1', '0.2', '0.3'],
    ['-1.5', '0.25', '-1.25'],
    ['-1.5', '-0.25', '-1.75'],
    ['10', '0.001', '10.001'],
    ['999.999', '0.001', '1000'],
  ])('%s + %s = %s (unterschiedliche Nachkommazahl, Vorzeichen)', (left, right, expected) => {
    expect(addDecimals(left, right)).toBe(expected);
  });

  it.each([
    ['1.10', '-1.1'],
    ['-0.5', '0.5'],
    ['-0', '0'],
    ['-0.000', '0.0'],
  ])('liefert "0" und nie "-0" für %s + %s', (left, right) => {
    expect(addDecimals(left, right)).toBe('0');
  });

  it('rechnet Großzahlen ohne Präzisionsverlust', () => {
    expect(addDecimals('123456789012345678.123456789', '0.000000001')).toBe(
      '123456789012345678.12345679',
    );
  });

  it('rechnet sehr viele Nachkommastellen exakt', () => {
    expect(addDecimals('0.00000000000000000001', '0.00000000000000000002')).toBe(
      '0.00000000000000000003',
    );
  });

  it.each([
    ['1.50', '0', '1.5'],
    ['2.000', '0', '2'],
    ['0.0', '0.00', '0'],
    ['007.10', '0', '7.1'],
  ])('normalisiert %s + %s zu %s', (left, right, expected) => {
    expect(addDecimals(left, right)).toBe(expected);
  });
});

describe('subtractDecimals', () => {
  it.each([
    ['0.3', '0.1', '0.2'],
    ['1', '2.5', '-1.5'],
    ['-1', '0.25', '-1.25'],
    ['-1', '-2.5', '1.5'],
    ['81365.0', '76507', '4858'],
  ])('%s - %s = %s', (left, right, expected) => {
    expect(subtractDecimals(left, right)).toBe(expected);
  });

  it.each([
    ['-1', '-1'],
    ['1.50', '1.5'],
    ['0', '0.000'],
  ])('liefert "0" und nie "-0" für %s - %s', (left, right) => {
    expect(subtractDecimals(left, right)).toBe('0');
  });

  it('rechnet Großzahlen ohne Präzisionsverlust', () => {
    expect(subtractDecimals('100000000000000000000', '0.00000000000000000001')).toBe(
      '99999999999999999999.99999999999999999999',
    );
  });
});

describe('divideDecimals', () => {
  it.each([
    ['1', '3', '0.33333333'],
    ['2', '3', '0.66666667'],
    ['10', '4', '2.5'],
    ['4858', '81365.0', '0.05970626'],
    ['-2', '3', '-0.66666667'],
    ['2', '-3', '-0.66666667'],
    ['-2', '-3', '0.66666667'],
  ])('%s / %s = %s mit Standard-Skala 8', (dividend, divisor, expected) => {
    expect(divideDecimals(dividend, divisor)).toBe(expected);
  });

  it('rundet die genaue Hälfte weg von 0 (positiv)', () => {
    expect(divideDecimals('0.000000005', '1')).toBe('0.00000001');
  });

  it('rundet die genaue Hälfte weg von 0 (negativ)', () => {
    expect(divideDecimals('-0.000000005', '1')).toBe('-0.00000001');
  });

  it('rundet unterhalb der Hälfte ab und liefert dabei "0" statt "-0"', () => {
    expect(divideDecimals('0.000000004', '1')).toBe('0');
    expect(divideDecimals('-0.000000004', '1')).toBe('0');
  });

  it('beachtet eine abweichende Skala', () => {
    expect(divideDecimals('1', '3', 2)).toBe('0.33');
    expect(divideDecimals('5', '2', 0)).toBe('3');
    expect(divideDecimals('-5', '2', 0)).toBe('-3');
  });

  it('teilt Großzahlen ohne Präzisionsverlust', () => {
    expect(divideDecimals('123456789012345678.5', '1')).toBe('123456789012345678.5');
  });

  it('normalisiert das Ergebnis', () => {
    expect(divideDecimals('3.0', '1.50')).toBe('2');
    expect(divideDecimals('0', '7')).toBe('0');
    expect(divideDecimals('-0', '7')).toBe('0');
  });

  it.each(['0', '0.000', '-0'])('wirft DivisionByZeroError bei Divisor "%s"', (divisor) => {
    const caught = captureError(() => divideDecimals('1', divisor));

    expect(caught).toBeInstanceOf(DivisionByZeroError);
  });
});

describe('ungültige Eingaben', () => {
  const invalidAmounts = ['1e-7', 'abc', '', '1.', '.5', '+1', ' 1', '1,5', 'NaN'];

  it.each(invalidAmounts)('addDecimals wirft InvalidDecimalError für "%s"', (invalid) => {
    expect(captureError(() => addDecimals(invalid, '1'))).toBeInstanceOf(InvalidDecimalError);
    expect(captureError(() => addDecimals('1', invalid))).toBeInstanceOf(InvalidDecimalError);
  });

  it.each(invalidAmounts)('subtractDecimals wirft InvalidDecimalError für "%s"', (invalid) => {
    expect(captureError(() => subtractDecimals(invalid, '1'))).toBeInstanceOf(InvalidDecimalError);
    expect(captureError(() => subtractDecimals('1', invalid))).toBeInstanceOf(InvalidDecimalError);
  });

  it.each(invalidAmounts)('divideDecimals wirft InvalidDecimalError für "%s"', (invalid) => {
    expect(captureError(() => divideDecimals(invalid, '1'))).toBeInstanceOf(InvalidDecimalError);
    expect(captureError(() => divideDecimals('1', invalid))).toBeInstanceOf(InvalidDecimalError);
  });

  it('meldet einen ungültigen Divisor nicht als Division durch 0', () => {
    const caught = captureError(() => divideDecimals('1', ''));

    expect(caught).toBeInstanceOf(InvalidDecimalError);
    expect(caught).not.toBeInstanceOf(DivisionByZeroError);
  });

  describe('Ergänzungen aus dem Review', () => {
    it('beide Fehler erben von DecimalError', () => {
      expect(captureError(() => divideDecimals('1', '0'))).toBeInstanceOf(DecimalError);
      expect(captureError(() => addDecimals('x', '1'))).toBeInstanceOf(DecimalError);
    });

    it.each([-1, 1.5, Number.NaN])(
      'lehnt eine ungültige Skala %s mit InvalidDecimalError ab',
      (scale) => {
        expect(captureError(() => divideDecimals('1', '3', scale))).toBeInstanceOf(
          InvalidDecimalError,
        );
      },
    );

    it.each([
      ['1', '-8', 2, '-0.13'],
      ['1', '-8', 3, '-0.125'],
      ['-1', '8', 2, '-0.13'],
      ['-1', '-8', 2, '0.13'],
      ['1', '3', 2, '0.33'],
      ['2', '3', 2, '0.67'],
      ['1.123456789', '1', 4, '1.1235'],
      ['1', '0.003', 2, '333.33'],
    ])('%s / %s mit Skala %i ergibt %s', (dividend, divisor, scale, expected) => {
      expect(divideDecimals(dividend, divisor, scale)).toBe(expected);
    });
  });
});
