import { describe, expect, it } from 'vitest';
import { TradingSymbol } from '@/domain/position/valueObjects/TradingSymbol';
import { InvalidTradingSymbolError } from '@/domain/position/errors/InvalidTradingSymbolError';
import { PositionDomainError } from '@/domain/position/errors/PositionDomainError';

describe('TradingSymbol', () => {
  it.each(['BTC-USDT', 'ONDO-USDT', 'NCCOGOLD2USD-USDT', '1000PEPE-USDT'])(
    'akzeptiert "%s"',
    (rawSymbol) => {
      expect(TradingSymbol.create(rawSymbol).value).toBe(rawSymbol);
    },
  );

  it.each(['', 'BTCUSDT', 'btc-usdt', 'BTC-', '-USDT', 'BTC-USDT-X', 'BTC_USDT', ' BTC-USDT'])(
    'lehnt ungültiges Symbol "%s" ab',
    (rawSymbol) => {
      expect(() => TradingSymbol.create(rawSymbol)).toThrow(InvalidTradingSymbolError);
    },
  );

  it('lehnt Nicht-Strings ab', () => {
    expect(() => TradingSymbol.create(42)).toThrow(InvalidTradingSymbolError);
    expect(() => TradingSymbol.create(undefined)).toThrow(InvalidTradingSymbolError);
  });

  it('wirft einen Fehler, der von PositionDomainError erbt', () => {
    let caught: unknown;
    try {
      TradingSymbol.create('kaputt');
    } catch (error) {
      caught = error;
    }

    expect(caught).toBeInstanceOf(InvalidTradingSymbolError);
    expect(caught).toBeInstanceOf(PositionDomainError);
  });
});
