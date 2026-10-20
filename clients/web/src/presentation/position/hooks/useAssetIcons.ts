import { useEffect, useState } from 'react';
import type { AssetIcon } from '@/domain/asset/AssetIcon';
import type { AssetIconsGateway } from '@/domain/asset/AssetIconsGateway';

const NO_ICONS: ReadonlyMap<string, AssetIcon> = new Map();

/** Lädt die Icons für die Symbole; ruft nur bei geänderter Symbolmenge erneut ab. */
export function useAssetIcons(
  gateway: AssetIconsGateway | undefined,
  symbols: readonly string[],
): ReadonlyMap<string, AssetIcon> {
  const uniqueSymbols = [...new Set(symbols)];
  const symbolKey = [...uniqueSymbols].sort().join(',');
  // Nur bei geänderter Symbolmenge (nicht bei neuer Array-Instanz oder Reihenfolge) neu abrufen.
  const [request, setRequest] = useState({ key: symbolKey, symbols: uniqueSymbols });
  if (request.key !== symbolKey) setRequest({ key: symbolKey, symbols: uniqueSymbols });
  const [icons, setIcons] = useState<ReadonlyMap<string, AssetIcon>>(NO_ICONS);

  useEffect(() => {
    if (gateway === undefined || request.symbols.length === 0) return;
    const controller = new AbortController();
    gateway
      .fetchAssetIcons(request.symbols, controller.signal)
      .then((loaded) => {
        if (!controller.signal.aborted) {
          setIcons(new Map(loaded.map((icon) => [icon.symbol, icon])));
        }
      })
      .catch(() => undefined);
    return () => controller.abort();
  }, [gateway, request]);

  return icons;
}
