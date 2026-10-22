import type { PositionChart, RLevel } from '@/domain/position/PositionChart';
import type { ChartPriceLine } from '@/presentation/position/chart/ChartRenderer';
import { formatDecimal } from '@/presentation/position/format/formatDecimal';
import { formatRatioAsPercent } from '@/presentation/position/format/formatRatioAsPercent';

/** Liegt 3R höchstens so weit (relativ zum Entry) vom TP entfernt, wird eine Linie "TP = 3R" gezeichnet. */
const MERGE_TOLERANCE_RATIO = 0.001;
/** Rundungsfehler bei Gleitkomma-Vergleichen an der Grenze. */
const FLOAT_SLACK = 1e-9;

const R_LINE_KINDS: Readonly<Record<RLevel['kind'], 'r1' | 'r2' | 'r3'>> = {
  ONE_R: 'r1',
  TWO_R: 'r2',
  THREE_R: 'r3',
};

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
  for (const order of chart.takeProfit) {
    lines.push({
      kind: 'target',
      price: Number(order.triggerPrice),
      title: protectiveTitle('TP', order.triggerPrice, order.distanceRatio),
    });
  }
  for (const level of chart.levels ?? []) {
    addRLevelLine(lines, level, entryPrice, chart.takeProfit.length > 0);
  }
  return lines;
}

function addRLevelLine(
  lines: ChartPriceLine[],
  level: RLevel,
  entryPrice: number,
  hasTakeProfit: boolean,
): void {
  const price = Number(level.price);
  if (level.kind === 'THREE_R') {
    const mergeTargetIndex = lines.findIndex(
      (line) =>
        line.kind === 'target' &&
        Math.abs(line.price - price) <= entryPrice * MERGE_TOLERANCE_RATIO + FLOAT_SLACK,
    );
    const mergeTarget = lines[mergeTargetIndex];
    if (mergeTarget) {
      lines[mergeTargetIndex] = { kind: 'target', price: mergeTarget.price, title: 'TP = 3R' };
      return;
    }
    lines.push({ kind: 'r3', price, title: hasTakeProfit ? '3R' : '3R TP3' });
    return;
  }
  const kind = R_LINE_KINDS[level.kind];
  lines.push({ kind, price, title: level.kind === 'TWO_R' ? 'BE' : '1R' });
}
