import { describe, expect, it } from 'vitest';
import { GetOpenPositionsUseCase } from '@/application/position/GetOpenPositionsUseCase';
import type { OpenPosition } from '@/domain/position/entities/OpenPosition';
import type { OpenPositionsSnapshot } from '@/domain/position/entities/OpenPositionsSnapshot';
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

const FETCHED_AT = 1_791_800_000_000;

/** Fake-Source: liefert einen Snapshot mit den vorbereiteten Positionen und zählt die Aufrufe. */
function createFakeSource(positions: readonly OpenPosition[], fetchedAt: number = FETCHED_AT) {
  let callCount = 0;
  const source: OpenPositionSource = {
    async fetchOpenPositions(): Promise<OpenPositionsSnapshot> {
      callCount += 1;
      return { positions: [...positions], fetchedAt };
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

    expect(result.positions.map((position) => position.positionId)).toEqual(['2', '3', '1']);
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

    expect(result.positions.map((position) => position.symbol)).toEqual([
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
    const { source } = createFakeSource([shortPosition, longB, longA]);

    const sorted = await new GetOpenPositionsUseCase(source).execute();

    expect(sorted.positions.map((position) => position.positionId)).toEqual(['1', '2', '3']);
  });

  it('liefert die Positionen unverändert, ergänzt um liquidationDistanceRatio', async () => {
    const position = openPosition();
    const { source } = createFakeSource([position]);

    const result = await new GetOpenPositionsUseCase(source).execute();

    // LONG: (1.0100 − 0.865) / 1.0100 = 0.1435643564… → 0.14356436
    expect(result.positions).toEqual([{ ...position, liquidationDistanceRatio: '0.14356436' }]);
  });

  it('berechnet liquidationDistanceRatio je Seite und null ohne Liquidationspreis', async () => {
    const longPosition = openPosition({
      positionId: '1',
      side: 'LONG',
      markPrice: '81365.0',
      liquidationPrice: '76507',
      openTime: 3,
    });
    const shortPosition = openPosition({
      positionId: '2',
      side: 'SHORT',
      markPrice: '4175.00',
      liquidationPrice: '4410.5',
      openTime: 2,
    });
    const crossPosition = openPosition({
      positionId: '3',
      liquidationPrice: '0',
      openTime: 1,
    });
    const { source } = createFakeSource([crossPosition, shortPosition, longPosition]);

    const result = await new GetOpenPositionsUseCase(source).execute();

    expect(
      result.positions.map((position) => [position.positionId, position.liquidationDistanceRatio]),
    ).toEqual([
      ['1', '0.05970626'],
      ['2', '0.05640719'],
      ['3', null],
    ]);
  });

  it('liefert die exakte Summe des unrealisierten PnL', async () => {
    const { source } = createFakeSource([
      openPosition({ positionId: '1', unrealizedProfit: '25.0000' }),
      openPosition({ positionId: '2', unrealizedProfit: '-6.2500' }),
      openPosition({ positionId: '3', unrealizedProfit: '0.00000001' }),
    ]);

    const result = await new GetOpenPositionsUseCase(source).execute();

    expect(result.totalUnrealizedProfit).toBe('18.75000001');
  });

  it('übernimmt fetchedAt aus dem Snapshot der Source', async () => {
    const { source } = createFakeSource([openPosition()], 1_791_123_456_789);

    const result = await new GetOpenPositionsUseCase(source).execute();

    expect(result.fetchedAt).toBe(1_791_123_456_789);
  });

  it('liefert genau positions, totalUnrealizedProfit und fetchedAt', async () => {
    const { source } = createFakeSource([openPosition()]);

    const result = await new GetOpenPositionsUseCase(source).execute();

    expect(Object.keys(result).sort()).toEqual(['fetchedAt', 'positions', 'totalUnrealizedProfit']);
  });

  it('liefert eine leere Liste und Summe "0", wenn keine Position offen ist', async () => {
    const { source } = createFakeSource([]);

    await expect(new GetOpenPositionsUseCase(source).execute()).resolves.toEqual({
      positions: [],
      totalUnrealizedProfit: '0',
      fetchedAt: FETCHED_AT,
    });
  });

  it('reicht OpenPositionsUnavailableError unverändert durch', async () => {
    const unavailableError = new OpenPositionsUnavailableError('rate-limited');
    const failingSource: OpenPositionSource = {
      async fetchOpenPositions(): Promise<OpenPositionsSnapshot> {
        throw unavailableError;
      },
    };

    await expect(new GetOpenPositionsUseCase(failingSource).execute()).rejects.toBe(
      unavailableError,
    );
  });
});
