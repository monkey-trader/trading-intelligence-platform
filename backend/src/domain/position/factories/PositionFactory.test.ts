import { describe, expect, it } from 'vitest';
import { PositionFactory } from '@/domain/position/factories/PositionFactory';
import type { PositionInput } from '@/domain/position/factories/PositionFactory';
import type { Position } from '@/domain/position/entities/Position';
import { PositionDomainError } from '@/domain/position/errors/PositionDomainError';
import { InvalidPositionIdError } from '@/domain/position/errors/InvalidPositionIdError';
import { InvalidTradingSymbolError } from '@/domain/position/errors/InvalidTradingSymbolError';
import { InvalidPositionSideError } from '@/domain/position/errors/InvalidPositionSideError';
import { InvalidDecimalAmountError } from '@/domain/position/errors/InvalidDecimalAmountError';
import { InvalidTimestampError } from '@/domain/position/errors/InvalidTimestampError';
import { CloseBeforeOpenError } from '@/domain/position/errors/CloseBeforeOpenError';
import { InvalidLeverageError } from '@/domain/position/errors/InvalidLeverageError';

function validInput(overrides: Partial<PositionInput> = {}): PositionInput {
  return {
    positionId: '9000000000000000000',
    symbol: 'ONDO-USDT',
    side: 'LONG',
    isolated: true,
    openTime: 1780000000000,
    closeTime: 1780018017000,
    avgOpenPrice: '0.4120',
    avgClosePrice: '0.4188',
    realisedProfit: '1.7000',
    netProfit: '1.5123',
    positionAmount: '250.00',
    closedAmount: '250.00',
    commission: '-0.05121506352000001',
    totalFunding: '-0.1356',
    leverage: 7,
    fullyClosed: true,
    ...overrides,
  };
}

function captureError(action: () => unknown): unknown {
  try {
    action();
  } catch (error) {
    return error;
  }
  throw new Error('Es wurde ein Fehler erwartet, aber keiner geworfen.');
}

describe('PositionFactory.create', () => {
  it('erzeugt eine Position mit primitiven, unveränderten Feldern', () => {
    const expected: Position = {
      positionId: '9000000000000000000',
      symbol: 'ONDO-USDT',
      side: 'LONG',
      isolated: true,
      openTime: 1780000000000,
      closeTime: 1780018017000,
      avgOpenPrice: '0.4120',
      avgClosePrice: '0.4188',
      realisedProfit: '1.7000',
      netProfit: '1.5123',
      positionAmount: '250.00',
      closedAmount: '250.00',
      commission: '-0.05121506352000001',
      totalFunding: '-0.1356',
      leverage: 7,
      fullyClosed: true,
    };

    expect(PositionFactory.create(validInput())).toEqual(expected);
  });

  it('erlaubt closeTime gleich openTime', () => {
    const position = PositionFactory.create(
      validInput({ openTime: 1780000000000, closeTime: 1780000000000 }),
    );

    expect(position.closeTime).toBe(1780000000000);
  });

  it('akzeptiert eine SHORT-Position mit negativem Gewinn', () => {
    const position = PositionFactory.create(
      validInput({ side: 'SHORT', realisedProfit: '-2.1675', netProfit: '-2.4019' }),
    );

    expect(position.side).toBe('SHORT');
    expect(position.netProfit).toBe('-2.4019');
  });

  it.each([
    ['positionId', { positionId: 'abc' }, InvalidPositionIdError],
    ['symbol', { symbol: 'ondo' }, InvalidTradingSymbolError],
    ['side', { side: 'BOTH' as unknown as 'LONG' }, InvalidPositionSideError],
    ['openTime', { openTime: 0 }, InvalidTimestampError],
    ['closeTime', { closeTime: -1 }, InvalidTimestampError],
    ['avgOpenPrice', { avgOpenPrice: 'x' }, InvalidDecimalAmountError],
    ['avgClosePrice', { avgClosePrice: '' }, InvalidDecimalAmountError],
    ['realisedProfit', { realisedProfit: '1e3' }, InvalidDecimalAmountError],
    ['netProfit', { netProfit: '1,5' }, InvalidDecimalAmountError],
    ['positionAmount', { positionAmount: 'abc' }, InvalidDecimalAmountError],
    ['closedAmount', { closedAmount: '.5' }, InvalidDecimalAmountError],
    ['commission', { commission: '+1' }, InvalidDecimalAmountError],
    ['totalFunding', { totalFunding: 'NaN' }, InvalidDecimalAmountError],
    ['leverage', { leverage: 0 }, InvalidLeverageError],
  ] as const)('lehnt ungültiges Feld %s ab', (_fieldName, overrides, expectedError) => {
    const caught = captureError(() => PositionFactory.create(validInput(overrides)));

    expect(caught).toBeInstanceOf(expectedError);
    expect(caught).toBeInstanceOf(PositionDomainError);
  });

  it('wirft CloseBeforeOpenError, wenn closeTime vor openTime liegt', () => {
    const caught = captureError(() =>
      PositionFactory.create(validInput({ openTime: 1780000000000, closeTime: 1779999999999 })),
    );

    expect(caught).toBeInstanceOf(CloseBeforeOpenError);
    expect(caught).toBeInstanceOf(PositionDomainError);
  });
});
