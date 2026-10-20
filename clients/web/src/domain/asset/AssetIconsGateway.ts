import type { AssetIcon } from '@/domain/asset/AssetIcon';

/** Port zum Abfragen der Icons. Fehler ergeben eine leere Liste; nur ein Abbruch wird geworfen. */
export type AssetIconsGateway = Readonly<{
  fetchAssetIcons: (
    symbols: readonly string[],
    signal: AbortSignal,
  ) => Promise<readonly AssetIcon[]>;
}>;
