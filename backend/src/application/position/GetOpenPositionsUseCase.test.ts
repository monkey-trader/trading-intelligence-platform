import { describe, expect, it } from 'vitest';
import { GetOpenPositionsUseCase } from '@/application/position/GetOpenPositionsUseCase';
import type { OpenPosition } from '@/domain/position/entities/OpenPosition';
import type { OpenPositionSource } from '@/domain/position/interfaces/OpenPositionSource';
import { OpenPositionsUnavailableError } from '@/domain/position/errors/OpenPositionsUnavailableError';

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

/** Fake-Source: liefert die vorbereiteten Positionen und zählt die Aufrufe. */
function createFakeSource(positions: readonly OpenPosition[]) {
  let callCount = 0;
  const source: OpenPositionSource = {
    async fetchOpenPositions(): Promise<OpenPosition[]> {
      callCount += 1;
      return [...positions];
    },
  };
  return { source, getCallCount: () => callCount };
}

describe('GetOpenPositionsUseCase', () => {
  it('sortiert nach openTime absteigend (neueste zuerst)', async () => {
    const oldest = openPosition({ positionId: '1', symbol: 'AAA-USDT', openTime: 1791000000000 });
    const newest = openPosition({ positionId: '2', symbol: 'BBB-USDT', openTime: 1791900000000 });
    const middle = openPosition({ positionId: '3', symbol: 'CCC-USDT', openTime: 1791500000000 });
    const { source, getCallCount } = createFakeSource([oldest, newest, middle]);

    const result = await new GetOpenPositionsUseCase(source).execute();

    expect(result.map((position) => position.positionId)).toEqual(['2', '3', '1']);
    expect(getCallCount()).toBe(1);
  });

  it('sortiert bei gleicher openTime nach symbol aufsteigend', async () => {
    const sameOpenTime = 1791500000000;
    const suiPosition = openPosition({
      positionId: '1',
      symbol: 'SUI-USDT',
      openTime: sameOpenTime,
    });
    const btcPosition = openPosition({
      positionId: '2',
      symbol: 'BTC-USDT',
      openTime: sameOpenTime,
    });
    const ethPosition = openPosition({
      positionId: '3',
      symbol: 'ETH-USDT',
      openTime: sameOpenTime,
    });
    const newerPosition = openPosition({
      positionId: '4',
      symbol: 'ZEC-USDT',
      openTime: sameOpenTime + 1,
    });
    const { source } = createFakeSource([suiPosition, btcPosition, newerPosition, ethPosition]);

    const result = await new GetOpenPositionsUseCase(source).execute();

    expect(result.map((position) => position.symbol)).toEqual([
      'ZEC-USDT',
      'BTC-USDT',
      'ETH-USDT',
      'SUI-USDT',
    ]);
  });

  it('sortiert bei gleicher openTime und gleichem Symbol LONG vor SHORT, dann nach positionId', async () => {
    const sameTime = 1_791_000_000_000;
    const shortPosition = openPosition({
      positionId: '3',
      symbol: 'BTC-USDT',
      side: 'SHORT',
      openTime: sameTime,
    });
    const longB = openPosition({
      positionId: '2',
      symbol: 'BTC-USDT',
      side: 'LONG',
      openTime: sameTime,
    });
    const longA = openPosition({
      positionId: '1',
      symbol: 'BTC-USDT',
      side: 'LONG',
      openTime: sameTime,
    });
    const useCase = new GetOpenPositionsUseCase({
      fetchOpenPositions: async () => [shortPosition, longB, longA],
    });

    const sorted = await useCase.execute();

    expect(sorted.map((position) => position.positionId)).toEqual(['1', '2', '3']);
  });

  it('liefert die Positionen selbst unverändert', async () => {
    const position = openPosition();
    const { source } = createFakeSource([position]);

    const result = await new GetOpenPositionsUseCase(source).execute();

    expect(result).toEqual([position]);
  });

  it('liefert ein leeres Array, wenn keine Position offen ist', async () => {
    const { source } = createFakeSource([]);

    await expect(new GetOpenPositionsUseCase(source).execute()).resolves.toEqual([]);
  });

  it('reicht OpenPositionsUnavailableError unverändert durch', async () => {
    const unavailableError = new OpenPositionsUnavailableError('rate-limited');
    const failingSource: OpenPositionSource = {
      async fetchOpenPositions(): Promise<OpenPosition[]> {
        throw unavailableError;
      },
    };

    await expect(new GetOpenPositionsUseCase(failingSource).execute()).rejects.toBe(
      unavailableError,
    );
  });
});
