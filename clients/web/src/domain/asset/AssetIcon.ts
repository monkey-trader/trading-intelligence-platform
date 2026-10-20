/** Anzeigedaten eines Handelssymbols: Basis-Asset, Anzeigename und (optional) Icon-URL. */
export type AssetIcon = Readonly<{
  symbol: string;
  baseAsset: string;
  displayName: string;
  iconUrl: string | null;
}>;
