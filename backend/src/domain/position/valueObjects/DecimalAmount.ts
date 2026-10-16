import { InvalidDecimalAmountError } from '@/domain/position/errors/InvalidDecimalAmountError';

const DECIMAL_PATTERN = /^-?\d+(\.\d+)?$/;

/**
 * Betrag als Dezimal-String. Der Wert wird unverändert übernommen,
 * damit keine Float-Rundung passiert. number wird bewusst abgelehnt.
 */
export class DecimalAmount {
  private constructor(readonly value: string) {}

  static create(rawAmount: unknown): DecimalAmount {
    if (typeof rawAmount !== 'string' || !DECIMAL_PATTERN.test(rawAmount)) {
      throw new InvalidDecimalAmountError(
        `Ungültiger Betrag: ${typeof rawAmount === 'string' ? `"${rawAmount}"` : `Typ ${typeof rawAmount}`}`,
      );
    }
    return new DecimalAmount(rawAmount);
  }
}
