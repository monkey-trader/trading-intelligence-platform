import { useCallback, useState } from 'react';
import {
  nextSortState,
  type SortKey,
  type SortState,
} from '@/presentation/position/sorting/sortPositions';

/** Sortierzustand der Positionstabelle (nur in der Sitzung). */
export function useTableSort(): {
  sort: SortState;
  cycle: (key: SortKey) => void;
  select: (key: SortKey | null) => void;
  toggleDirection: () => void;
} {
  const [sort, setSort] = useState<SortState>(null);
  const cycle = useCallback(
    (key: SortKey) => setSort((current) => nextSortState(current, key)),
    [],
  );
  const select = useCallback(
    (key: SortKey | null) => setSort(key === null ? null : { key, direction: 'asc' }),
    [],
  );
  const toggleDirection = useCallback(
    () =>
      setSort((current) =>
        current === null
          ? null
          : { key: current.key, direction: current.direction === 'asc' ? 'desc' : 'asc' },
      ),
    [],
  );
  return { sort, cycle, select, toggleDirection };
}
