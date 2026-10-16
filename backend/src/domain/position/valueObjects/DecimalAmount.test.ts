import { describe, expect, it } from 'vitest';
import { DecimalAmount } from '@/domain/position/valueObjects/DecimalAmount';
import { InvalidDecimalAmountError } from '@/domain/position/errors/InvalidDecimalAmountError';
import { PositionDomainError } from '@/domain/position/errors/PositionDomainError';

describe('DecimalAmount', () => {
  it('übernimmt "-0.05121506352000001" unverändert (keine Float-Rundung)', () => {
    expect(DecimalAmount.create('-0.05121506352000001').value).toBe('-0.05121506352000001');
  });

  it.each(['0', '12', '-3.5', '250.00', '0.4120', '3412.55'])(
    'übernimmt "%s" unverändert',
    (rawAmount) => {
      expect(DecimalAmount.create(rawAmount).value).toBe(rawAmount);
    },
  );

  it.each(['', 'abc', '1e5', '1.', '.5', '+1', ' 1', '1,5', '--1'])(
    'lehnt ungültigen Betrag "%s" ab',
    (rawAmount) => {
      expect(() => DecimalAmount.create(rawAmount)).toThrow(InvalidDecimalAmountError);
    },
  );

  it('lehnt eine number ab, damit keine Präzision verloren geht', () => {
    expect(() => DecimalAmount.create(1.5)).toThrow(InvalidDecimalAmountError);
  });

  it('lehnt undefined und null ab', () => {
    expect(() => DecimalAmount.create(undefined)).toThrow(InvalidDecimalAmountError);
    expect(() => DecimalAmount.create(null)).toThrow(InvalidDecimalAmountError);
  });

  it('wirft einen Fehler, der von PositionDomainError erbt', () => {
    let caught: unknown;
    try {
      DecimalAmount.create('abc');
    } catch (error) {
      caught = error;
    }

    expect(caught).toBeInstanceOf(InvalidDecimalAmountError);
    expect(caught).toBeInstanceOf(PositionDomainError);
  });
});
