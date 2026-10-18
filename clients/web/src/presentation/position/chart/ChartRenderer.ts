/** Kerze für den Renderer: Zeit in Sekunden, Preise als Zahlen. */
export type ChartCandle = Readonly<{
  time: number;
  open: number;
  high: number;
  low: number;
  close: number;
}>;

export type ChartPriceLine = Readonly<{
  kind: 'entry' | 'stop' | 'target';
  price: number;
  title: string;
}>;

export type ChartMarker = Readonly<{
  kind: 'entry';
  /** Sekunden */
  time: number;
  price: number;
}>;

/** Port zur Chart-Zeichenbibliothek, damit die Präsentation ohne Canvas testbar bleibt. */
export type ChartRenderer = Readonly<{
  setCandles: (candles: readonly ChartCandle[]) => void;
  setPriceLines: (lines: readonly ChartPriceLine[]) => void;
  setMarkers: (markers: readonly ChartMarker[]) => void;
  destroy: () => void;
}>;

export type ChartRendererFactory = (container: HTMLElement) => ChartRenderer;
