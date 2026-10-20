export const COLUMN_ORDER_STORAGE_KEY = 'tip_open_positions_column_order_v2';

export const DEFAULT_COLUMN_ORDER = [
  'symbol',
  'positionValue',
  'margin',
  'unrealizedProfit',
  'realisedProfit',
  'entry',
  'tpSl',
] as const;

export type ColumnKey = (typeof DEFAULT_COLUMN_ORDER)[number];

const FIXED_FIRST_KEY: ColumnKey = 'symbol';

function isColumnKey(value: unknown): value is ColumnKey {
  return DEFAULT_COLUMN_ORDER.some((key) => key === value);
}

/** Die Symbol-Spalte bleibt immer erste Spalte und ist nicht verschiebbar. */
export function moveColumn(
  order: readonly ColumnKey[],
  fromKey: ColumnKey,
  toKey: ColumnKey,
): ColumnKey[] {
  const fromIndex = order.indexOf(fromKey);
  const toIndex = order.indexOf(toKey);
  const isMovable =
    fromIndex >= 0 &&
    toIndex >= 0 &&
    fromKey !== toKey &&
    fromKey !== FIXED_FIRST_KEY &&
    toKey !== FIXED_FIRST_KEY;
  if (!isMovable) return [...order];
  const withoutSource = order.filter((key) => key !== fromKey);
  withoutSource.splice(toIndex, 0, fromKey);
  return withoutSource;
}

/** Macht aus beliebigen gespeicherten Daten eine vollständige, gültige Reihenfolge. */
export function sanitizeColumnOrder(stored: unknown): ColumnKey[] {
  const candidates: unknown[] = Array.isArray(stored) ? stored : [];
  const result: ColumnKey[] = [FIXED_FIRST_KEY];
  for (const candidate of candidates) {
    if (isColumnKey(candidate) && !result.includes(candidate)) result.push(candidate);
  }
  for (const key of DEFAULT_COLUMN_ORDER) {
    if (!result.includes(key)) result.push(key);
  }
  return result;
}

export function isDefaultColumnOrder(order: readonly ColumnKey[]): boolean {
  return (
    order.length === DEFAULT_COLUMN_ORDER.length &&
    order.every((key, index) => key === DEFAULT_COLUMN_ORDER[index])
  );
}
