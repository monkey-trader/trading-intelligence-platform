import { InvalidTradingSymbolError } from '@/domain/position/errors/InvalidTradingSymbolError';

const SYMBOL_PATTERN = /^[A-Z0-9]+-[A-Z0-9]+$/;

/** Handelssymbol im BingX-Format BASIS-QUOTE, z.B. BTC-USDT. */
export class TradingSymbol {
  private constructor(readonly value: string) {}

  static create(rawSymbol: unknown): TradingSymbol {
    if (typeof rawSymbol !== 'string' || !SYMBOL_PATTERN.test(rawSymbol)) {
      throw new InvalidTradingSymbolError(
        `Ungültiges Symbol: ${typeof rawSymbol === 'string' ? `"${rawSymbol}"` : `Typ ${typeof rawSymbol}`}`,
      );
    }
    return new TradingSymbol(rawSymbol);
  }
}
