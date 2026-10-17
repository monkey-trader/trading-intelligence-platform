import { describe, expect, it } from 'vitest';
import { BingXPositionHistorySource } from '@/infrastructure/position/BingXPositionHistorySource';
import { UnexpectedBingXResponseError } from '@/infrastructure/bingx/errors';
import page0Fixture from '@fixtures/bingx/positionHistory.page0.json';
import shortFixture from '@fixtures/bingx/positionHistory.short.json';
import emptyFixture from '@fixtures/bingx/positionHistory.empty.json';

const POSITION_HISTORY_PATH = '/openApi/swap/v1/trade/positionHistory';
const PAGE_SIZE = 100;
const FROM_MS = 1_780_000_000_000;
const TO_MS = 1_787_000_000_000;

type QueryParams = Readonly<Record<string, string | number>>;
type GetCall = { path: string; params: QueryParams };

/** Fake für den BingX-Client: liefert pro Aufruf die nächste vorbereitete `data`-Antwort. */
function createFakeClient(pages: readonly unknown[]) {
  const calls: GetCall[] = [];
  const client = {
    async get(path: string, params: QueryParams): Promise<unknown> {
      calls.push({ path, params });
      const page = pages[calls.length - 1];
      if (page === undefined) throw new Error(`Unerwartete Anfrage Nr. ${calls.length}`);
      return page;
    },
  };
  return { client, calls };
}

/** Erzeugt eine volle Seite (100 Einträge) mit eindeutigen positionIds. */
function fullPage(pageNumber: number): unknown {
  const template = page0Fixture.data.positionHistory[0];
  const positionHistory = Array.from({ length: PAGE_SIZE }, (_unused, index) => ({
    ...template,
    positionId: (9_100_000_000_000_000_000n + BigInt(pageNumber * PAGE_SIZE + index)).toString(),
  }));
  return { positionHistory };
}

describe('BingXPositionHistorySource', () => {
  it('Regression Prototyp-Bug: die erste Anfrage nutzt pageIndex=0 (nicht 1)', async () => {
    const { client, calls } = createFakeClient([page0Fixture.data]);
    const source = new BingXPositionHistorySource(client);

    await source.fetchClosedPositions('ONDO-USDT', FROM_MS, TO_MS);

    expect(calls[0]?.params.pageIndex).toBe(0);
  });

  it('übergibt Pfad, symbol, startTs, endTs und pageSize=100', async () => {
    const { client, calls } = createFakeClient([page0Fixture.data]);
    const source = new BingXPositionHistorySource(client);

    await source.fetchClosedPositions('ONDO-USDT', FROM_MS, TO_MS);

    expect(calls).toHaveLength(1);
    expect(calls[0]?.path).toBe(POSITION_HISTORY_PATH);
    expect(calls[0]?.params).toMatchObject({
      symbol: 'ONDO-USDT',
      startTs: FROM_MS,
      endTs: TO_MS,
      pageIndex: 0,
      pageSize: PAGE_SIZE,
    });
  });

  it('blättert bei vollen Seiten mit pageIndex 1, 2 weiter und stoppt bei einer kurzen Seite', async () => {
    const { client, calls } = createFakeClient([fullPage(0), fullPage(1), shortFixture.data]);
    const source = new BingXPositionHistorySource(client);

    const records = await source.fetchClosedPositions('ONDO-USDT', FROM_MS, TO_MS);

    expect(calls.map((call) => call.params.pageIndex)).toEqual([0, 1, 2]);
    expect(records).toHaveLength(2 * PAGE_SIZE + 1);
    for (const call of calls) {
      expect(call.params).toMatchObject({ symbol: 'ONDO-USDT', startTs: FROM_MS, endTs: TO_MS });
    }
  });

  it('stoppt nach einer vollen Seite, wenn die nächste leer ist', async () => {
    const { client, calls } = createFakeClient([fullPage(0), emptyFixture.data]);
    const source = new BingXPositionHistorySource(client);

    const records = await source.fetchClosedPositions('ONDO-USDT', FROM_MS, TO_MS);

    expect(calls).toHaveLength(2);
    expect(records).toHaveLength(PAGE_SIZE);
  });

  it('liefert die gemappten Positionen inklusive rawPayload', async () => {
    const { client } = createFakeClient([page0Fixture.data]);
    const source = new BingXPositionHistorySource(client);

    const records = await source.fetchClosedPositions('ONDO-USDT', FROM_MS, TO_MS);

    expect(records.map((record) => record.position.positionId)).toEqual([
      '9000000000000000000',
      '9000000000000000001',
      '9000000000000000002',
    ]);
    expect(records[0]?.position.commission).toBe('-0.05121506352000001');
    expect(JSON.parse(records[0]?.rawPayload ?? 'null')).toEqual(
      page0Fixture.data.positionHistory[0],
    );
  });

  it('liefert bei leerer Historie eine leere Liste nach genau einer Anfrage', async () => {
    const { client, calls } = createFakeClient([emptyFixture.data]);
    const source = new BingXPositionHistorySource(client);

    await expect(source.fetchClosedPositions('ONDO-USDT', FROM_MS, TO_MS)).resolves.toEqual([]);
    expect(calls).toHaveLength(1);
  });

  it('bricht mit Fehler ab, statt mehr als maxPages Seiten abzufragen', async () => {
    const endlessPages = Array.from({ length: 10 }, (_unused, pageNumber) => fullPage(pageNumber));
    const { client, calls } = createFakeClient(endlessPages);
    const source = new BingXPositionHistorySource(client, { maxPages: 3 });

    await expect(source.fetchClosedPositions('ONDO-USDT', FROM_MS, TO_MS)).rejects.toThrow(Error);
    expect(calls).toHaveLength(3);
  });

  it('reicht UnexpectedBingXResponseError bei unerwarteter Antwortform durch', async () => {
    const { client } = createFakeClient([{ list: [] }]);
    const source = new BingXPositionHistorySource(client);

    await expect(source.fetchClosedPositions('ONDO-USDT', FROM_MS, TO_MS)).rejects.toBeInstanceOf(
      UnexpectedBingXResponseError,
    );
  });

  it('reicht Fehler des Clients (z.B. HTTP 429 nach Retries) unverändert durch', async () => {
    const clientError = new Error('Rate-Limit erschöpft');
    const client = {
      async get(): Promise<unknown> {
        throw clientError;
      },
    };
    const source = new BingXPositionHistorySource(client);

    await expect(source.fetchClosedPositions('ONDO-USDT', FROM_MS, TO_MS)).rejects.toBe(
      clientError,
    );
  });
});
