import { useEffect, useMemo, useRef } from 'react';
import { CHART_INTERVALS, type PositionChart } from '@/domain/position/PositionChart';
import type { PositionChartGateway } from '@/domain/position/PositionChartGateway';
import type {
  ChartCandle,
  ChartMarker,
  ChartPriceLine,
  ChartRenderer,
  ChartRendererFactory,
} from '@/presentation/position/chart/ChartRenderer';
import { formatDecimal } from '@/presentation/position/format/formatDecimal';
import { formatRatioAsPercent } from '@/presentation/position/format/formatRatioAsPercent';
import { usePositionChartViewModel } from '@/presentation/position/hooks/usePositionChartViewModel';
import styles from '@/presentation/position/components/PositionChartPanel.module.css';

type PositionChartPanelProps = Readonly<{
  positionId: string;
  gateway: PositionChartGateway;
  createRenderer: ChartRendererFactory;
  onClose?: (() => void) | undefined;
}>;

const SIDE_LABELS: Readonly<Record<PositionChart['side'], string>> = {
  LONG: 'Long',
  SHORT: 'Short',
};

/** Menge 0 heißt bei BingX: die ganze Position. */
const WHOLE_POSITION_LABEL = 'ganze Position';

type LegendEntry = Readonly<{ key: string; kind: ChartPriceLine['kind']; text: string }>;

/** Chart-Panel einer offenen Position mit Intervallwahl und Legende. */
export function PositionChartPanel({
  positionId,
  gateway,
  createRenderer,
  onClose,
}: PositionChartPanelProps) {
  const { status, chart, errorMessage, interval, selectInterval } = usePositionChartViewModel(
    positionId,
    gateway,
  );
  const containerRef = useRef<HTMLDivElement>(null);
  const rendererRef = useRef<ChartRenderer | null>(null);

  // Renderer einmal pro Panel erzeugen und beim Entfernen aufräumen.
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return undefined;
    const renderer = createRenderer(container);
    rendererRef.current = renderer;
    return () => {
      renderer.destroy();
      rendererRef.current = null;
    };
    // createRenderer ist ein stabiler Composition-Root-Wert; ein Wechsel soll nicht neu erzeugen.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const legend = useMemo(() => (chart ? buildLegend(chart) : []), [chart]);

  const candlesKeyRef = useRef<string | null>(null);
  const priceLinesKeyRef = useRef<string | null>(null);

  useEffect(() => {
    const renderer = rendererRef.current;
    if (!renderer || !chart) return;
    const candlesKey = `${positionId}|${interval}`;
    if (candlesKeyRef.current !== candlesKey) {
      renderer.setCandles(toChartCandles(chart));
      candlesKeyRef.current = candlesKey;
    } else {
      renderer.updateCandles(toChartCandles(chart));
    }
    const priceLines = legend.map(toPriceLine);
    const priceLinesKey = JSON.stringify(priceLines);
    if (priceLinesKeyRef.current !== priceLinesKey) {
      renderer.setPriceLines(priceLines);
      priceLinesKeyRef.current = priceLinesKey;
    }
    renderer.setMarkers(toChartMarkers(chart));
  }, [chart, legend, positionId, interval]);

  // Design folgt dem System (prefers-color-scheme): Farben im Renderer neu anwenden.
  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const onThemeChange = () => rendererRef.current?.applyTheme();
    mediaQuery.addEventListener('change', onThemeChange);
    return () => mediaQuery.removeEventListener('change', onThemeChange);
  }, []);

  return (
    <section className={styles.panel} aria-label="Chart">
      <header className={styles.header}>
        <h2 className={styles.title}>
          {chart ? `${chart.symbol} ${SIDE_LABELS[chart.side]}` : 'Chart'}
        </h2>
        <div role="group" aria-label="Intervall" className={styles.intervals}>
          {CHART_INTERVALS.map((chartInterval) => (
            <button
              key={chartInterval}
              type="button"
              aria-pressed={chartInterval === interval}
              className={styles.intervalButton}
              onClick={() => selectInterval(chartInterval)}
            >
              {chartInterval}
            </button>
          ))}
        </div>
        <button type="button" className={styles.closeButton} onClick={onClose}>
          Schließen
        </button>
      </header>

      {status === 'loading' && (
        <p role="status" className={styles.message}>
          Lade Chart …
        </p>
      )}
      {status === 'error' && (
        <p role="alert" className={styles.errorText}>
          {errorMessage}
        </p>
      )}

      <div ref={containerRef} className={styles.chart} />

      {chart && (
        <ul aria-label="Legende" className={styles.legend}>
          {legend.map((entry) => (
            <li key={entry.key} className={`${styles.legendItem} ${styles[entry.kind]}`}>
              {entry.text}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function toChartCandles(chart: PositionChart): ChartCandle[] {
  return chart.candles
    .map((candle) => ({
      time: Math.floor(candle.time / 1000),
      open: Number(candle.open),
      high: Number(candle.high),
      low: Number(candle.low),
      close: Number(candle.close),
    }))
    .sort((first, second) => first.time - second.time);
}

function toChartMarkers(chart: PositionChart): ChartMarker[] {
  return chart.markers.map((marker) => ({
    kind: 'entry',
    time: Math.floor(marker.time / 1000),
    price: Number(marker.price),
  }));
}

function toPriceLine(entry: LegendEntry & { price: number; title: string }): ChartPriceLine {
  return { kind: entry.kind, price: entry.price, title: entry.title };
}

function buildLegend(chart: PositionChart): Array<LegendEntry & { price: number; title: string }> {
  const entryTitle = `Entry ${formatDecimal(chart.entry.price)}`;
  const entries: Array<LegendEntry & { price: number; title: string }> = [
    {
      key: 'entry',
      kind: 'entry',
      price: Number(chart.entry.price),
      title: entryTitle,
      text: entryTitle,
    },
  ];
  for (const order of chart.stopLoss) {
    const title = protectiveTitle('SL', order.triggerPrice, order.distanceRatio);
    const quantity = isZeroQuantity(order.quantity)
      ? WHOLE_POSITION_LABEL
      : `Menge ${formatDecimal(order.quantity)}`;
    entries.push({
      key: `sl-${order.orderId}`,
      kind: 'stop',
      price: Number(order.triggerPrice),
      title,
      text: `${title} · ${quantity}`,
    });
  }
  for (const order of chart.takeProfit) {
    const title = protectiveTitle('TP', order.triggerPrice, order.distanceRatio);
    const quantity = isZeroQuantity(order.quantity)
      ? WHOLE_POSITION_LABEL
      : `Menge ${formatDecimal(order.quantity)}`;
    entries.push({
      key: `tp-${order.orderId}`,
      kind: 'target',
      price: Number(order.triggerPrice),
      title,
      text: `${title} · ${quantity}`,
    });
  }
  return entries;
}

function protectiveTitle(label: string, triggerPrice: string, distanceRatio: string): string {
  const distance = formatRatioAsPercent(distanceRatio, { signDisplay: 'exceptZero' });
  return `${label} ${formatDecimal(triggerPrice)} (${distance})`;
}

function isZeroQuantity(quantity: string): boolean {
  return Number(quantity) === 0;
}
