import { describe, expect, it } from 'vitest';
import { BingXTradedSymbolSource } from '@/infrastructure/position/BingXTradedSymbolSource';
import incomePageFixture from '@/infrastructure/position/bingx/fixtures/income.page.json';

const INCOME_PATH = '/openApi/swap/v2/user/income';
const DAY_MS = 86_400_000;
const WEEK_MS = 7 * DAY_MS;
const INCOME_LIMIT = 1000;
const FROM_MS = 1_780_000_000_000;

type QueryParams = Readonly<Record<string, string | number>>;
type GetCall = { path: string; params: QueryParams };
type IncomeRow = (typeof incomePageFixture)[number];

/** Fake für den BingX-Client: die Antwort wird pro Aufruf aus den Parametern berechnet. */
function createFakeClient(
  respond: (params: QueryParams, callNumber: number) => unknown = () => [],
) {
  const calls: GetCall[] = [];
  const client = {
    async get(path: string, params: QueryParams): Promise<unknown> {
      calls.push({ path, params });
      return respond(params, calls.length);
    },
  };
  return { client, calls };
}

function incomeRow(overrides: Partial<IncomeRow>): IncomeRow {
  return {
    symbol: 'BTC-USDT',
    incomeType: 'REALIZED_PNL',
    income: '1.0',
    asset: 'USDT',
    time: FROM_MS,
    info: 'Realized PnL',
    tranId: '1',
    tradeId: '1',
    ...overrides,
  };
}

describe('BingXTradedSymbolSource', () => {
  it('liefert nur Symbole mit REALIZED_PNL (nicht TRADING_FEE oder FUNDING_FEE)', async () => {
    const { client } = createFakeClient((_params, callNumber) =>
      callNumber === 1 ? incomePageFixture : [],
    );
    const source = new BingXTradedSymbolSource(client);

    const symbols = await source.listSymbolsWithRealizedPnl(FROM_MS, FROM_MS + DAY_MS);

    expect([...symbols].sort()).toEqual(['NCCOGOLD2USD-USDT', 'ONDO-USDT']);
    expect(symbols.has('XRP-USDT')).toBe(false);
  });

  it('fragt in 7-Tage-Fenstern ab, lückenlos und nie über toMs hinaus', async () => {
    const toMs = FROM_MS + 20 * DAY_MS;
    const { client, calls } = createFakeClient();
    const source = new BingXTradedSymbolSource(client);

    await source.listSymbolsWithRealizedPnl(FROM_MS, toMs);

    expect(calls.map((call) => [call.params.startTime, call.params.endTime])).toEqual([
      [FROM_MS, FROM_MS + WEEK_MS],
      [FROM_MS + WEEK_MS, FROM_MS + 2 * WEEK_MS],
      [FROM_MS + 2 * WEEK_MS, toMs],
    ]);
    for (const call of calls) {
      expect(call.path).toBe(INCOME_PATH);
      expect(call.params.limit).toBe(INCOME_LIMIT);
    }
  });

  it('blättert bei 1000 Zeilen mit startTime = max(time) + 1 im selben Fenster weiter', async () => {
    const toMs = FROM_MS + DAY_MS;
    const fullPage = Array.from({ length: INCOME_LIMIT }, (_unused, index) =>
      incomeRow({
        symbol: 'BTC-USDT',
        incomeType: 'TRADING_FEE',
        // absichtlich unsortiert: das Maximum liegt nicht am Ende
        time: index === 10 ? FROM_MS + 5_000 : FROM_MS + index,
        tranId: String(index),
      }),
    );
    const secondPage = [incomeRow({ symbol: 'ETH-USDT', time: FROM_MS + 6_000, tranId: 'x' })];
    const { client, calls } = createFakeClient((_params, callNumber) =>
      callNumber === 1 ? fullPage : callNumber === 2 ? secondPage : [],
    );
    const source = new BingXTradedSymbolSource(client);

    const symbols = await source.listSymbolsWithRealizedPnl(FROM_MS, toMs);

    expect(calls).toHaveLength(2);
    expect(calls[0]?.params).toMatchObject({
      startTime: FROM_MS,
      endTime: toMs,
      limit: INCOME_LIMIT,
    });
    expect(calls[1]?.params).toMatchObject({
      startTime: FROM_MS + 5_000 + 1,
      endTime: toMs,
      limit: INCOME_LIMIT,
    });
    expect([...symbols]).toEqual(['ETH-USDT']);
  });

  it('blättert bei weniger als 1000 Zeilen nicht weiter', async () => {
    const { client, calls } = createFakeClient(() => incomePageFixture);
    const source = new BingXTradedSymbolSource(client);

    await source.listSymbolsWithRealizedPnl(FROM_MS, FROM_MS + DAY_MS);

    expect(calls).toHaveLength(1);
  });

  it('liefert bei leerer Historie eine leere Menge', async () => {
    const { client } = createFakeClient(() => []);
    const source = new BingXTradedSymbolSource(client);

    const symbols = await source.listSymbolsWithRealizedPnl(FROM_MS, FROM_MS + 30 * DAY_MS);

    expect(symbols.size).toBe(0);
  });

  it('behandelt data: null wie eine leere Liste (gemessen: BingX liefert das für Fenster älter als ~90 Tage)', async () => {
    const { client, calls } = createFakeClient((_params, callNumber) =>
      callNumber === 1 ? null : incomePageFixture,
    );
    const source = new BingXTradedSymbolSource(client);

    const symbols = await source.listSymbolsWithRealizedPnl(FROM_MS, FROM_MS + 10 * DAY_MS);

    expect(calls).toHaveLength(2);
    expect([...symbols].sort()).toEqual(['NCCOGOLD2USD-USDT', 'ONDO-USDT']);
  });

  it('wirft bei einer Antwort, die weder Liste noch null ist', async () => {
    const { client } = createFakeClient(() => ({ list: [] }));
    const source = new BingXTradedSymbolSource(client);

    await expect(source.listSymbolsWithRealizedPnl(FROM_MS, FROM_MS + DAY_MS)).rejects.toThrow(
      'keine Liste',
    );
  });

  it('fragt nichts ab, wenn fromMs nicht vor toMs liegt', async () => {
    const { client, calls } = createFakeClient();
    const source = new BingXTradedSymbolSource(client);

    const symbols = await source.listSymbolsWithRealizedPnl(FROM_MS, FROM_MS);

    expect(calls).toHaveLength(0);
    expect(symbols.size).toBe(0);
  });

  it('reicht Fehler des Clients durch', async () => {
    const clientError = new Error('code 100001');
    const client = {
      async get(): Promise<unknown> {
        throw clientError;
      },
    };
    const source = new BingXTradedSymbolSource(client);

    await expect(source.listSymbolsWithRealizedPnl(FROM_MS, FROM_MS + DAY_MS)).rejects.toBe(
      clientError,
    );
  });
});
