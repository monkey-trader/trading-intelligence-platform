import type { OpenPosition } from '@/domain/position/OpenPosition';

export type SortKey =
  | 'symbol'
  | 'value'
  | 'margin'
  | 'unrealizedProfit'
  | 'realisedProfit'
  | 'entry'
  | 'stopLoss'
  | 'be'
  | 'tp1';
export type SortDirection = 'asc' | 'desc';
export type SortState = Readonly<{ key: SortKey; direction: SortDirection }> | null;

/** Klickzyklus: aufsteigend, absteigend, aus (Reihenfolge der API). */
export function nextSortState(current: SortState, key: SortKey): SortState {
  if (current === null || current.key !== key) return { key, direction: 'asc' };
  return current.direction === 'asc' ? { key, direction: 'desc' } : null;
}

type SortValue = string | number | null;

function numberOrNull(value: string | undefined): number | null {
  if (value === undefined) return null;
  const parsed = Number(value);
  return Number.isNaN(parsed) ? null : parsed;
}

function levelPrice(position: OpenPosition, kind: 'TWO_R' | 'THREE_R'): number | null {
  return numberOrNull(position.levels?.find((level) => level.kind === kind)?.price);
}

/** null = kein Wert; solche Zeilen stehen immer am Ende. */
function sortValue(position: OpenPosition, key: SortKey): SortValue {
  switch (key) {
    case 'symbol':
      return position.symbol;
    case 'value':
      return numberOrNull(position.positionValue);
    case 'margin':
      return numberOrNull(position.margin);
    case 'unrealizedProfit':
      return numberOrNull(position.unrealizedProfit);
    case 'realisedProfit':
      return numberOrNull(position.realisedProfit);
    case 'entry':
      return numberOrNull(position.avgOpenPrice);
    case 'stopLoss':
      return position.stopLoss === null ? null : numberOrNull(position.stopLoss);
    case 'be':
      return levelPrice(position, 'TWO_R');
    case 'tp1':
      return levelPrice(position, 'THREE_R');
  }
}

function compareValues(left: string | number, right: string | number): number {
  if (typeof left === 'string' && typeof right === 'string') return left.localeCompare(right, 'de');
  return Number(left) - Number(right);
}

/** 0 = hat Wert, 1 = Wert fehlt, 2 = Ursprungs-SL unbekannt; höhere Ränge stehen weiter hinten. */
function rankOf(position: OpenPosition, value: SortValue): number {
  if (position.initialStopLoss === null) return 2;
  return value === null ? 1 : 0;
}

/** Stabil sortiert, ohne die Eingabe zu ändern. Zeilen ohne Wert oder ohne bekannten SL ans Ende. */
export function sortPositions(
  positions: readonly OpenPosition[],
  state: SortState,
): OpenPosition[] {
  if (state === null) return [...positions];
  const factor = state.direction === 'asc' ? 1 : -1;
  const entries = positions.map((position) => {
    const value = sortValue(position, state.key);
    return { position, value, rank: rankOf(position, value) };
  });
  entries.sort((left, right) => {
    if (left.rank !== right.rank) return left.rank - right.rank;
    if (left.rank !== 0 || left.value === null || right.value === null) return 0;
    return factor * compareValues(left.value, right.value);
  });
  return entries.map((entry) => entry.position);
}
