import { describe, expect, it } from 'vitest';
import { OpenPositionFactory } from '@/domain/position/factories/OpenPositionFactory';
import type { OpenPositionInput } from '@/domain/position/factories/OpenPositionFactory';
import type { OpenPosition } from '@/domain/position/entities/OpenPosition';
import { PositionDomainError } from '@/domain/position/errors/PositionDomainError';
import { InvalidPositionIdError } from '@/domain/position/errors/InvalidPositionIdError';
import { InvalidTradingSymbolError } from '@/domain/position/errors/InvalidTradingSymbolError';
import { InvalidPositionSideError } from '@/domain/position/errors/InvalidPositionSideError';
import { InvalidDecimalAmountError } from '@/domain/position/errors/InvalidDecimalAmountError';
import { InvalidLeverageError } from '@/domain/position/errors/InvalidLeverageError';
import { InvalidTimestampError } from '@/domain/position/errors/InvalidTimestampError';
import { UpdateBeforeOpenError } from '@/domain/position/errors/UpdateBeforeOpenError';

function validInput(overrides: Partial<OpenPositionInput> = {}): OpenPositionInput {
  return {
    positionId: '9100000000000000001',
    symbol: 'SUI-USDT',
    side: 'LONG',
    isolated: true,
    positionAmount: '500',
    avgOpenPrice: '0.9600',
    markPrice: '1.0100',
    leverage: 10,
    initialMargin: '48.0000',
    margin: '60.5000',
    positionValue: '505',
    unrealizedProfit: '25.0000',
    realisedProfit: '-0.2400',
    pnlRatio: '0.5208',
    liquidationPrice: '0.865',
    openTime: 1791500000000,
    updateTime: 1791600000000,
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

describe('OpenPositionFactory.create', () => {
  it('erzeugt eine offene Position mit primitiven, unveränderten Feldern', () => {
    const expected: OpenPosition = {
      positionId: '9100000000000000001',
      symbol: 'SUI-USDT',
      side: 'LONG',
      isolated: true,
      positionAmount: '500',
      avgOpenPrice: '0.9600',
      markPrice: '1.0100',
      leverage: 10,
      initialMargin: '48.0000',
      margin: '60.5000',
      positionValue: '505',
      unrealizedProfit: '25.0000',
      realisedProfit: '-0.2400',
      pnlRatio: '0.5208',
      liquidationPrice: '0.865',
      openTime: 1791500000000,
      updateTime: 1791600000000,
    };

    expect(OpenPositionFactory.create(validInput())).toEqual(expected);
  });

  it('übernimmt Dezimal-Strings exakt, ohne Rundung oder Umwandlung in number', () => {
    const position = OpenPositionFactory.create(
      validInput({
        side: 'SHORT',
        realisedProfit: '-0.5187',
        unrealizedProfit: '-6.2500',
        pnlRatio: '-0.0904',
        initialMargin: '0.05121506352000001',
      }),
    );

    expect(position.side).toBe('SHORT');
    expect(position.realisedProfit).toBe('-0.5187');
    expect(position.unrealizedProfit).toBe('-6.2500');
    expect(position.pnlRatio).toBe('-0.0904');
    expect(position.initialMargin).toBe('0.05121506352000001');
  });

  it('akzeptiert liquidationPrice "0" (keine Liquidation)', () => {
    const position = OpenPositionFactory.create(validInput({ liquidationPrice: '0' }));

    expect(position.liquidationPrice).toBe('0');
  });

  it('akzeptiert einen Dezimal-Hebel', () => {
    const position = OpenPositionFactory.create(validInput({ leverage: 2.5 }));

    expect(position.leverage).toBe(2.5);
  });

  it('erlaubt updateTime gleich openTime', () => {
    const position = OpenPositionFactory.create(
      validInput({ openTime: 1791500000000, updateTime: 1791500000000 }),
    );

    expect(position.updateTime).toBe(1791500000000);
  });

  it('enthält keine Felder über das Entity hinaus', () => {
    const position = OpenPositionFactory.create(validInput());

    expect(Object.keys(position).sort()).toEqual(
      [
        'positionId',
        'symbol',
        'side',
        'isolated',
        'positionAmount',
        'avgOpenPrice',
        'markPrice',
        'leverage',
        'initialMargin',
        'margin',
        'positionValue',
        'unrealizedProfit',
        'realisedProfit',
        'pnlRatio',
        'liquidationPrice',
        'openTime',
        'updateTime',
      ].sort(),
    );
  });

  it.each([
    ['positionId', { positionId: 'abc' }, InvalidPositionIdError],
    ['symbol', { symbol: 'sui' }, InvalidTradingSymbolError],
    ['side', { side: 'BOTH' }, InvalidPositionSideError],
    ['positionAmount', { positionAmount: 'abc' }, InvalidDecimalAmountError],
    ['avgOpenPrice', { avgOpenPrice: '' }, InvalidDecimalAmountError],
    ['markPrice', { markPrice: '1e3' }, InvalidDecimalAmountError],
    ['initialMargin', { initialMargin: '1,5' }, InvalidDecimalAmountError],
    ['margin', { margin: '.5' }, InvalidDecimalAmountError],
    ['positionValue', { positionValue: '+1' }, InvalidDecimalAmountError],
    ['unrealizedProfit', { unrealizedProfit: 'NaN' }, InvalidDecimalAmountError],
    ['realisedProfit', { realisedProfit: 'x' }, InvalidDecimalAmountError],
    ['pnlRatio', { pnlRatio: '25%' }, InvalidDecimalAmountError],
    ['liquidationPrice', { liquidationPrice: '1e-7' }, InvalidDecimalAmountError],
    ['leverage', { leverage: 0 }, InvalidLeverageError],
    ['openTime', { openTime: 0 }, InvalidTimestampError],
    ['updateTime', { updateTime: -1 }, InvalidTimestampError],
  ] as const)('lehnt ungültiges Feld %s ab', (_fieldName, overrides, expectedError) => {
    const caught = captureError(() => OpenPositionFactory.create(validInput(overrides)));

    expect(caught).toBeInstanceOf(expectedError);
    expect(caught).toBeInstanceOf(PositionDomainError);
  });

  it('wirft UpdateBeforeOpenError, wenn updateTime vor openTime liegt', () => {
    const caught = captureError(() =>
      OpenPositionFactory.create(
        validInput({ openTime: 1791500000000, updateTime: 1791499999999 }),
      ),
    );

    expect(caught).toBeInstanceOf(UpdateBeforeOpenError);
    expect(caught).toBeInstanceOf(PositionDomainError);
  });
});
