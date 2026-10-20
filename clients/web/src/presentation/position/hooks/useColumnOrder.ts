import { useCallback, useState } from 'react';
import {
  COLUMN_ORDER_STORAGE_KEY,
  DEFAULT_COLUMN_ORDER,
  isDefaultColumnOrder,
  moveColumn,
  sanitizeColumnOrder,
  type ColumnKey,
} from '@/presentation/position/columnOrder';

type StorageLike = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>;

function defaultStorage(): StorageLike | undefined {
  try {
    return window.localStorage;
  } catch {
    return undefined;
  }
}

function loadOrder(storage: StorageLike | undefined): ColumnKey[] {
  try {
    const raw = storage?.getItem(COLUMN_ORDER_STORAGE_KEY);
    if (raw === null || raw === undefined) return [...DEFAULT_COLUMN_ORDER];
    const parsed: unknown = JSON.parse(raw);
    return sanitizeColumnOrder(parsed);
  } catch {
    return [...DEFAULT_COLUMN_ORDER];
  }
}

/** Spaltenreihenfolge mit Persistenz; Speicherfehler werden bewusst ignoriert. */
export function useColumnOrder(storage: StorageLike | undefined = defaultStorage()): {
  order: readonly ColumnKey[];
  move: (fromKey: ColumnKey, toKey: ColumnKey) => void;
  reset: () => void;
  isDefault: boolean;
} {
  const [order, setOrder] = useState<readonly ColumnKey[]>(() => loadOrder(storage));

  const move = useCallback(
    (fromKey: ColumnKey, toKey: ColumnKey) => {
      const next = moveColumn(order, fromKey, toKey);
      setOrder(next);
      try {
        storage?.setItem(COLUMN_ORDER_STORAGE_KEY, JSON.stringify(next));
      } catch {
        // Speicher voll oder gesperrt: Reihenfolge gilt nur für diese Sitzung.
      }
    },
    [order, storage],
  );

  const reset = useCallback(() => {
    setOrder([...DEFAULT_COLUMN_ORDER]);
    try {
      storage?.removeItem(COLUMN_ORDER_STORAGE_KEY);
    } catch {
      // Gesperrter Speicher: Zustand ist trotzdem zurückgesetzt.
    }
  }, [storage]);

  return { order, move, reset, isDefault: isDefaultColumnOrder(order) };
}
