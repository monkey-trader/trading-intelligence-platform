import type { OpenPosition } from '@/domain/position/OpenPosition';
import { decimalSign } from '@/presentation/position/format/decimalString';
import { formatDecimal } from '@/presentation/position/format/formatDecimal';
import {
  formatRatioAsPercent,
  MISSING_VALUE,
} from '@/presentation/position/format/formatRatioAsPercent';
import { formatDateTime } from '@/presentation/position/format/formatTimestamp';
import { pnlTone } from '@/presentation/position/format/pnlTone';
import styles from '@/presentation/position/components/OpenPositionsTable.module.css';

const COLUMN_LABELS = {
  symbol: 'Symbol',
  side: 'Seite',
  amount: 'Menge',
  entry: 'Einstieg',
  mark: 'Mark',
  leverage: 'Hebel',
  unrealizedProfit: 'Unreal. PnL',
  pnlRatio: 'PnL %',
  liquidationPrice: 'Liq.-Preis',
  liquidationDistance: 'Liq.-Abstand',
  openTime: 'Eröffnet',
} as const;

const SIDE_LABELS: Readonly<Record<OpenPosition['side'], string>> = {
  LONG: 'Long',
  SHORT: 'Short',
};

type OpenPositionsTableProps = Readonly<{
  positions: readonly OpenPosition[];
}>;

/** Offene Positionen als Tabelle; auf schmalen Bildschirmen als Karten. */
export function OpenPositionsTable({ positions }: OpenPositionsTableProps) {
  return (
    <table className={styles.table} aria-label="Offene Positionen">
      <thead>
        <tr>
          {Object.values(COLUMN_LABELS).map((columnLabel) => (
            <th key={columnLabel} scope="col" className={styles.headerCell}>
              {columnLabel}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {positions.map((position) => (
          <OpenPositionRow key={`${position.positionId}-${position.side}`} position={position} />
        ))}
      </tbody>
    </table>
  );
}

function OpenPositionRow({ position }: Readonly<{ position: OpenPosition }>) {
  // Tönung passend zur Anzeige: PnL mit 2 Stellen, Verhältnis als Prozent mit 2 Stellen (= 4 Stellen).
  const profitToneClass = styles[pnlTone(position.unrealizedProfit, 2)];
  const ratioToneClass = styles[pnlTone(position.pnlRatio, 4)];
  const sideClass = position.side === 'LONG' ? styles.sideLong : styles.sideShort;
  const hasLiquidationPrice = decimalSign(position.liquidationPrice) !== 0;

  return (
    <tr className={styles.row}>
      <th
        scope="row"
        data-label={COLUMN_LABELS.symbol}
        className={`${styles.cell} ${styles.symbolCell}`}
      >
        {position.symbol}
      </th>
      <td data-label={COLUMN_LABELS.side} className={`${styles.cell} ${sideClass}`}>
        {SIDE_LABELS[position.side]}
      </td>
      <td data-label={COLUMN_LABELS.amount} className={styles.cell}>
        {formatDecimal(position.positionAmount)}
      </td>
      <td data-label={COLUMN_LABELS.entry} className={styles.cell}>
        {formatDecimal(position.avgOpenPrice)}
      </td>
      <td data-label={COLUMN_LABELS.mark} className={styles.cell}>
        {formatDecimal(position.markPrice)}
      </td>
      <td data-label={COLUMN_LABELS.leverage} className={styles.cell}>
        {`${position.leverage}×`}
      </td>
      <td
        data-label={COLUMN_LABELS.unrealizedProfit}
        className={`${styles.cell} ${profitToneClass}`}
      >
        {formatDecimal(position.unrealizedProfit, {
          minimumFractionDigits: 2,
          maximumFractionDigits: 2,
        })}
      </td>
      <td data-label={COLUMN_LABELS.pnlRatio} className={`${styles.cell} ${ratioToneClass}`}>
        {formatRatioAsPercent(position.pnlRatio)}
      </td>
      <td data-label={COLUMN_LABELS.liquidationPrice} className={styles.cell}>
        {hasLiquidationPrice ? formatDecimal(position.liquidationPrice) : MISSING_VALUE}
      </td>
      <td data-label={COLUMN_LABELS.liquidationDistance} className={styles.cell}>
        {formatRatioAsPercent(position.liquidationDistanceRatio)}
      </td>
      <td data-label={COLUMN_LABELS.openTime} className={styles.cell}>
        {formatDateTime(position.openTime)}
      </td>
    </tr>
  );
}
