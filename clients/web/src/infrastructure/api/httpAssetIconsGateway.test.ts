import { describe, expect, it, vi } from 'vitest';
import { createHttpAssetIconsGateway } from '@/infrastructure/api/httpAssetIconsGateway';

// Antwort von GET /assets/icons laut docs/openapi.yaml (erfundene Werte).
const body = {
  icons: [
    {
      symbol: 'ONDO-USDT',
      baseAsset: 'ONDO',
      displayName: 'ONDO',
      iconUrl: 'https://coin-images.coingecko.com/coins/images/26580/large/ONDO.png',
    },
    { symbol: 'NCCOGOLD2USD-USDT', baseAsset: 'XAU', displayName: 'Gold', iconUrl: null },
  ],
};

function jsonResponse(payload: unknown, status = 200): Response {
  return new Response(JSON.stringify(payload), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

describe('createHttpAssetIconsGateway', () => {
  it('ruft GET /assets/icons mit den kommaseparierten Symbolen auf und mappt die Einträge', async () => {
    const fetchFn = vi.fn<typeof fetch>().mockResolvedValue(jsonResponse(body));
    const gateway = createHttpAssetIconsGateway(fetchFn, '/api');

    const icons = await gateway.fetchAssetIcons(
      ['ONDO-USDT', 'NCCOGOLD2USD-USDT'],
      new AbortController().signal,
    );

    expect(fetchFn).toHaveBeenCalledTimes(1);
    const url = decodeURIComponent(String(fetchFn.mock.calls[0]?.[0]));
    expect(url).toBe('/api/assets/icons?symbols=ONDO-USDT,NCCOGOLD2USD-USDT');
    expect(fetchFn.mock.calls[0]?.[1]?.method ?? 'GET').toBe('GET');
    expect(icons).toEqual(body.icons);
  });

  it('liefert bei Netzwerkfehler ein leeres Ergebnis statt einer Exception', async () => {
    const fetchFn = vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch'));
    const gateway = createHttpAssetIconsGateway(fetchFn, '/api');

    await expect(
      gateway.fetchAssetIcons(['ONDO-USDT'], new AbortController().signal),
    ).resolves.toEqual([]);
  });

  it.each([400, 403, 404, 429, 500, 502])(
    'liefert bei HTTP %i ein leeres Ergebnis',
    async (status) => {
      const fetchFn = vi
        .fn<typeof fetch>()
        .mockResolvedValue(jsonResponse({ type: 'internal' }, status));
      const gateway = createHttpAssetIconsGateway(fetchFn, '/api');

      await expect(
        gateway.fetchAssetIcons(['ONDO-USDT'], new AbortController().signal),
      ).resolves.toEqual([]);
    },
  );

  it.each([
    ['kein JSON', new Response('<html>', { status: 200 })],
    ['falsche Form', jsonResponse({ foo: 1 })],
    ['icons kein Array', jsonResponse({ icons: 'x' })],
  ])('liefert bei 200 mit %s ein leeres Ergebnis', async (_name, response) => {
    const fetchFn = vi.fn<typeof fetch>().mockResolvedValue(response);
    const gateway = createHttpAssetIconsGateway(fetchFn, '/api');

    await expect(
      gateway.fetchAssetIcons(['ONDO-USDT'], new AbortController().signal),
    ).resolves.toEqual([]);
  });

  it('verwirft Einträge, die nicht zum Vertrag passen (fehlender displayName)', async () => {
    const fetchFn = vi.fn<typeof fetch>().mockResolvedValue(
      jsonResponse({
        icons: [
          { symbol: 'A-USDT', baseAsset: 'A', iconUrl: null },
          { symbol: 'B-USDT', baseAsset: 'B', displayName: 'B', iconUrl: null },
        ],
      }),
    );
    const gateway = createHttpAssetIconsGateway(fetchFn, '/api');

    const icons = await gateway.fetchAssetIcons(['A-USDT', 'B-USDT'], new AbortController().signal);

    expect(icons.map((icon) => icon.symbol)).not.toContain('A-USDT');
    expect(icons.map((icon) => icon.symbol)).toContain('B-USDT');
  });
});
