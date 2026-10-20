import { describe, expect, it } from 'vitest';
import { expectDocumentedResponse, expectProblem } from '@/contract/assertions';
import { createContractContext } from '@/contract/contractContext';
import { OPERATION_TEST_NAMES } from '@/contract/coverage';
import { hostHeaderFor } from '@/config';
import { isJsonObject } from '@/openapi';

const { config, validator, send } = createContractContext();

const ICONS_PATH = '/assets/icons';

describe(OPERATION_TEST_NAMES.getAssetIcons, () => {
  it('liefert für Rohstoffe 200 mit Anzeigename und ohne Icon (ohne CoinGecko-Zugriff)', async () => {
    const response = await send({
      method: 'GET',
      path: `${ICONS_PATH}?symbols=NCCOGOLD2USD-USDT,NCCOXAG2USD-USDT`,
    });

    expect(response.status).toBe(200);
    const body = expectDocumentedResponse(validator, 'getAssetIcons', response);
    expect(validator.validateSchema('AssetIcons', body).errors).toEqual([]);
    const icons = isJsonObject(body) && Array.isArray(body['icons']) ? body['icons'] : [];
    expect(icons).toEqual([
      { symbol: 'NCCOGOLD2USD-USDT', baseAsset: 'XAU', displayName: 'Gold', iconUrl: null },
      { symbol: 'NCCOXAG2USD-USDT', baseAsset: 'XAG', displayName: 'Silber', iconUrl: null },
    ]);
  });

  it('liefert 400 bad-request ohne den Parameter symbols', async () => {
    const response = await send({ method: 'GET', path: ICONS_PATH });

    expectDocumentedResponse(validator, 'getAssetIcons', response);
    expectProblem(validator, response, 400, 'bad-request');
  });

  it('liefert 400 bad-request bei einem ungültigen Symbol', async () => {
    const response = await send({ method: 'GET', path: `${ICONS_PATH}?symbols=btc-usdt` });

    expectDocumentedResponse(validator, 'getAssetIcons', response);
    expectProblem(validator, response, 400, 'bad-request');
  });

  it('liefert 400 bad-request bei mehr als 50 Symbolen', async () => {
    const symbols = Array.from({ length: 51 }, (_, index) => `COIN${index}-USDT`).join(',');
    const response = await send({ method: 'GET', path: `${ICONS_PATH}?symbols=${symbols}` });

    expectDocumentedResponse(validator, 'getAssetIcons', response);
    expectProblem(validator, response, 400, 'bad-request');
  });

  it('antwortet bei fremdem Host mit 403 forbidden-host laut Spec', async () => {
    const response = await send({
      method: 'GET',
      path: `${ICONS_PATH}?symbols=NCCOGOLD2USD-USDT`,
      headers: { host: hostHeaderFor(config.baseUrl, 'evil.example') },
    });

    expectDocumentedResponse(validator, 'getAssetIcons', response);
    expectProblem(validator, response, 403, 'forbidden-host');
  });
});
