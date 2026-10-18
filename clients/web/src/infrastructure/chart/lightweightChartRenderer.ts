import {
  CandlestickSeries,
  ColorType,
  LineStyle,
  createChart,
  createSeriesMarkers,
  type IChartApi,
  type ISeriesApi,
  type ISeriesMarkersPluginApi,
  type IPriceLine,
  type Time,
  type UTCTimestamp,
} from 'lightweight-charts';
import type {
  ChartPriceLine,
  ChartRenderer,
  ChartRendererFactory,
} from '@/presentation/position/chart/ChartRenderer';

/** Liest eine Design-Token-Farbe (CSS-Variable) vom Dokument, damit nichts hartcodiert ist. */
function readToken(tokenName: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(tokenName).trim();
}

function lineColor(kind: ChartPriceLine['kind']): string {
  if (kind === 'stop') return readToken('--color-chart-stop-loss');
  if (kind === 'target') return readToken('--color-chart-take-profit');
  return readToken('--color-chart-entry');
}

/** Adapter des ChartRenderer-Ports auf lightweight-charts (braucht Canvas, nicht per Unit-Test geprüft). */
export const createLightweightChartRenderer: ChartRendererFactory = (
  container: HTMLElement,
): ChartRenderer => {
  const chart: IChartApi = createChart(container, {
    autoSize: true,
    layout: {
      background: { type: ColorType.Solid, color: readToken('--color-surface') },
      textColor: readToken('--color-text'),
      fontFamily: readToken('--font-family-base'),
    },
    grid: {
      vertLines: { color: readToken('--color-chart-grid') },
      horzLines: { color: readToken('--color-chart-grid') },
    },
    rightPriceScale: { borderColor: readToken('--color-border') },
    timeScale: { borderColor: readToken('--color-border'), timeVisible: true },
  });
  const series: ISeriesApi<'Candlestick'> = chart.addSeries(CandlestickSeries, {
    upColor: readToken('--color-chart-up'),
    downColor: readToken('--color-chart-down'),
    borderUpColor: readToken('--color-chart-up'),
    borderDownColor: readToken('--color-chart-down'),
    wickUpColor: readToken('--color-chart-up'),
    wickDownColor: readToken('--color-chart-down'),
  });
  const markersPlugin: ISeriesMarkersPluginApi<Time> = createSeriesMarkers(series, []);
  let priceLines: IPriceLine[] = [];

  return {
    setCandles(candles) {
      series.setData(candles.map((candle) => ({ ...candle, time: candle.time as UTCTimestamp })));
      chart.timeScale().fitContent();
    },
    setPriceLines(lines) {
      priceLines.forEach((priceLine) => series.removePriceLine(priceLine));
      priceLines = lines.map((line) =>
        series.createPriceLine({
          price: line.price,
          title: line.title,
          color: lineColor(line.kind),
          lineStyle: LineStyle.Dashed,
          lineWidth: 1,
          axisLabelVisible: true,
        }),
      );
    },
    setMarkers(markers) {
      markersPlugin.setMarkers(
        markers.map((marker) => ({
          time: marker.time as UTCTimestamp,
          position: 'belowBar',
          shape: 'arrowUp',
          color: readToken('--color-chart-entry'),
          text: 'Entry',
        })),
      );
    },
    destroy() {
      markersPlugin.detach();
      chart.remove();
    },
  };
};
