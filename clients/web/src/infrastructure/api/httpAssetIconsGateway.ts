import type { AssetIcon } from '@/domain/asset/AssetIcon';
import type { AssetIconsGateway } from '@/domain/asset/AssetIconsGateway';

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function toAssetIcon(entry: unknown): AssetIcon | undefined {
  if (!isRecord(entry)) return undefined;
  const { symbol, baseAsset, displayName, iconUrl } = entry;
  if (
    typeof symbol !== 'string' ||
    typeof baseAsset !== 'string' ||
    typeof displayName !== 'string' ||
    (iconUrl !== null && typeof iconUrl !== 'string')
  ) {
    return undefined;
  }
  return { symbol, baseAsset, displayName, iconUrl };
}

/** Gateway auf GET /assets/icons; jeder Fehler ergibt eine leere Liste (Icons sind nur Zierde). */
export function createHttpAssetIconsGateway(
  fetchFn: typeof fetch = fetch,
  baseUrl = '/api',
): AssetIconsGateway {
  return {
    fetchAssetIcons: async (symbols, signal) => {
      try {
        const response = await fetchFn(
          `${baseUrl}/assets/icons?symbols=${encodeURIComponent(symbols.join(','))}`,
          { method: 'GET', signal },
        );
        if (!response.ok) return [];
        const body: unknown = await response.json();
        if (!isRecord(body) || !Array.isArray(body.icons)) return [];
        const entries: readonly unknown[] = body.icons;
        return entries.flatMap((entry) => {
          const icon = toAssetIcon(entry);
          return icon === undefined ? [] : [icon];
        });
      } catch (error) {
        if (error instanceof DOMException && error.name === 'AbortError') throw error;
        return [];
      }
    },
  };
}
