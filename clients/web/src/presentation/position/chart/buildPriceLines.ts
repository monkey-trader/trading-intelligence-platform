import type { PositionChart } from '@/domain/position/PositionChart';
import type { ChartPriceLine } from '@/presentation/position/chart/ChartRenderer';
import { formatDecimal } from '@/presentation/position/format/formatDecimal';
import { formatRatioAsPercent } from '@/presentation/position/format/formatRatioAsPercent';

/** Titel einer SL-/TP-Order, z.B. "SL 55.8 (+0.09 %)". */
export function protectiveTitle(
  label: string,
  triggerPrice: string,
  distanceRatio: string,
): string {
  const distance = formatRatioAsPercent(distanceRatio, { signDisplay: 'exceptZero' });
  return `${label} ${formatDecimal(triggerPrice)} (${distance})`;
}

/** Alle Preislinien des Charts: Entry, SL, TP und (falls bekannt) die R-Level. */
export function buildPriceLines(chart: PositionChart): ChartPriceLine[] {
  const entryPrice = Number(chart.entry.price);
  const lines: ChartPriceLine[] = [
    { kind: 'entry', price: entryPrice, title: `Entry ${formatDecimal(chart.entry.price)}` },
  ];
  for (const order of chart.stopLoss) {
    lines.push({
      kind: 'stop',
      price: Number(order.triggerPrice),
      title: protectiveTitle('SL', order.triggerPrice, order.distanceRatio),
    });
  }
  const threeR = (chart.levels ?? []).find((level) => level.kind === 'THREE_R');
  // Das Backend meldet nur, dass ein TP auf 3R liegt; das ist immer der dem 3R-Preis nächste.
  let mergedOrder: PositionChart['takeProfit'][number] | undefined;
  if (threeR?.coincidesWithTakeProfit === true) {
    const threeRPrice = Number(threeR.price);
    mergedOrder = [...chart.takeProfit].sort(
      (left, right) =>
        Math.abs(Number(left.triggerPrice) - threeRPrice) -
        Math.abs(Number(right.triggerPrice) - threeRPrice),
    )[0];
    lines.push({ kind: 'target', price: threeRPrice, title: 'TP = 3R' });
  }
  for (const order of chart.takeProfit) {
    if (order === mergedOrder) continue;
    lines.push({
      kind: 'target',
      price: Number(order.triggerPrice),
      title: protectiveTitle('TP', order.triggerPrice, order.distanceRatio),
    });
  }
  for (const level of chart.levels ?? []) {
    if (level.kind === 'TWO_R') {
      lines.push({ kind: 'r2', price: Number(level.price), title: 'BE' });
    } else if (level.coincidesWithTakeProfit !== true) {
      lines.push({ kind: 'r3', price: Number(level.price), title: 'TP1' });
    }
  }
  return lines;
}
