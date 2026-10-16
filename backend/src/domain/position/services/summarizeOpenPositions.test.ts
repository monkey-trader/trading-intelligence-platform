import { describe, expect, it } from 'vitest';
import { summarizeOpenPositions } from '@/domain/position/services/summarizeOpenPositions';
import type { OpenPosition } from '@/domain/position/entities/OpenPosition';
import { InvalidDecimalError } from '@/domain/shared/decimal/InvalidDecimalError';

function openPosition(overrides: Partial<OpenPosition> = {}): OpenPosition {
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

describe('summarizeOpenPositions', () => {
  it('summiert den unrealisierten PnL gemischter Vorzeichen exakt', () => {
    const positions = [
      openPosition({ positionId: '1', unrealizedProfit: '25.0000' }),
      openPosition({ positionId: '2', unrealizedProfit: '-6.2500' }),
      openPosition({ positionId: '3', unrealizedProfit: '0.00000001' }),
    ];

    expect(summarizeOpenPositions(positions)).toEqual({ totalUnrealizedProfit: '18.75000001' });
  });

  it('summiert ohne Float-Fehler (0.1 + 0.2 = 0.3)', () => {
    const positions = [
      openPosition({ positionId: '1', unrealizedProfit: '0.1' }),
      openPosition({ positionId: '2', unrealizedProfit: '0.2' }),
    ];

    expect(summarizeOpenPositions(positions).totalUnrealizedProfit).toBe('0.3');
  });

  it('liefert "0" und nicht "-0", wenn sich Gewinne und Verluste aufheben', () => {
    const positions = [
      openPosition({ positionId: '1', unrealizedProfit: '-3.50' }),
      openPosition({ positionId: '2', unrealizedProfit: '3.5' }),
    ];

    expect(summarizeOpenPositions(positions).totalUnrealizedProfit).toBe('0');
  });

  it('liefert "0" bei leerer Liste', () => {
    expect(summarizeOpenPositions([])).toEqual({ totalUnrealizedProfit: '0' });
  });

  it('wirft InvalidDecimalError bei ungültigem PnL', () => {
    expect(() => summarizeOpenPositions([openPosition({ unrealizedProfit: '1e2' })])).toThrow(
      InvalidDecimalError,
    );
  });
});
