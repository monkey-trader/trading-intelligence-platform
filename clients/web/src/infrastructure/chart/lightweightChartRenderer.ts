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
  switch (kind) {
    case 'stop':
      return readToken('--color-chart-stop-loss');
    case 'target':
      return readToken('--color-chart-take-profit');
    case 'entry':
      return readToken('--color-chart-entry');
    case 'r2':
      return readToken('--color-warning');
    case 'r3':
      return readToken('--color-chart-r-target');
  }
}

/** 3R gepunktet, 2R (Break-even-Trigger) gestrichelt, alles andere wie bisher gestrichelt. */
function lineStyle(kind: ChartPriceLine['kind']): LineStyle {
  return kind === 'r3' ? LineStyle.Dotted : LineStyle.Dashed;
}

/** Adapter des ChartRenderer-Ports auf lightweight-charts (braucht Canvas, nicht per Unit-Test geprüft). */
export const createLightweightChartRenderer: ChartRendererFactory = (
  container: HTMLElement,
): ChartRenderer => {
  const chartOptions = () => ({
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
  const seriesOptions = () => ({
    upColor: readToken('--color-chart-up'),
    downColor: readToken('--color-chart-down'),
    borderUpColor: readToken('--color-chart-up'),
    borderDownColor: readToken('--color-chart-down'),
    wickUpColor: readToken('--color-chart-up'),
    wickDownColor: readToken('--color-chart-down'),
  });
  const chart: IChartApi = createChart(container, { autoSize: true, ...chartOptions() });
  const series: ISeriesApi<'Candlestick'> = chart.addSeries(CandlestickSeries, seriesOptions());
  const markersPlugin: ISeriesMarkersPluginApi<Time> = createSeriesMarkers(series, []);
  let priceLines: IPriceLine[] = [];

  return {
    setCandles(candles) {
      series.setData(candles.map((candle) => ({ ...candle, time: candle.time as UTCTimestamp })));
      chart.timeScale().fitContent();
    },
    updateCandles(candles) {
      series.setData(candles.map((candle) => ({ ...candle, time: candle.time as UTCTimestamp })));
    },
    applyTheme() {
      chart.applyOptions(chartOptions());
      series.applyOptions(seriesOptions());
    },
    setPriceLines(lines) {
      priceLines.forEach((priceLine) => series.removePriceLine(priceLine));
      priceLines = lines.map((line) =>
        series.createPriceLine({
          price: line.price,
          title: line.title,
          color: lineColor(line.kind),
          lineStyle: lineStyle(line.kind),
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
