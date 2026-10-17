import { describe, expect, it } from 'vitest';
import { UnexpectedBingXResponseError } from '@/infrastructure/bingx/errors';
import { BingXContractSymbolSource } from '@/infrastructure/position/BingXContractSymbolSource';
import contractsFixture from '@fixtures/bingx/contracts.sample.json';

const CONTRACTS_PATH = '/openApi/swap/v2/quote/contracts';

type QueryParams = Readonly<Record<string, string | number>>;

function createFakeClient(response: unknown) {
  const calls: { path: string; params: QueryParams }[] = [];
  const client = {
    async get(path: string, params: QueryParams): Promise<unknown> {
      calls.push({ path, params });
      return response;
    },
  };
  return { client, calls };
}

describe('BingXContractSymbolSource', () => {
  it('liefert alle USDT-Kontrakte sortiert, auch pausierte (Status 25)', async () => {
    const { client, calls } = createFakeClient(contractsFixture);
    const source = new BingXContractSymbolSource(client);

    const symbols = await source.listContractSymbols();

    expect(symbols).toEqual(['BTC-USDT', 'NCCONICKEL2USD-USDT']);
    expect(calls).toEqual([{ path: CONTRACTS_PATH, params: {} }]);
  });

  it('lässt USDC-Kontrakte weg', async () => {
    const { client } = createFakeClient(contractsFixture);
    const source = new BingXContractSymbolSource(client);

    expect(await source.listContractSymbols()).not.toContain('BTC-USDC');
  });

  it.each([null, {}, 'kaputt'])(
    'wirft bei einer Antwort, die keine Liste ist: %j',
    async (data) => {
      const { client } = createFakeClient(data);
      const source = new BingXContractSymbolSource(client);

      await expect(source.listContractSymbols()).rejects.toBeInstanceOf(
        UnexpectedBingXResponseError,
      );
    },
  );

  it('wirft bei einer leeren Liste statt still nichts zu prüfen', async () => {
    const { client } = createFakeClient([]);
    const source = new BingXContractSymbolSource(client);

    await expect(source.listContractSymbols()).rejects.toBeInstanceOf(UnexpectedBingXResponseError);
  });

  it('wirft bei einem Eintrag ohne symbol oder currency', async () => {
    const { client } = createFakeClient([{ symbol: 'BTC-USDT' }]);
    const source = new BingXContractSymbolSource(client);

    await expect(source.listContractSymbols()).rejects.toBeInstanceOf(UnexpectedBingXResponseError);
  });
});
