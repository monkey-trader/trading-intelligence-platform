import { useEffect, useMemo, useState } from 'react';
import type {
  PositionPrice,
  PriceSnapshot,
  PriceStreamGateway,
  PriceStreamStatus,
} from '@/domain/price/PriceStream';

export type LivePrices = Readonly<{
  prices: Readonly<Record<string, PositionPrice>>;
  /** Stream ausgefallen oder PriceSnapshot.stale. */
  stale: boolean;
  /** Text für die aria-live-Region; wechselt nur zwischen Daten und veraltet. */
  announcement: string;
}>;

/** Abonniert den Kurs-Stream; der letzte bekannte Kurs bleibt bei Ausfall erhalten. */
export function useLivePrices(gateway: PriceStreamGateway): LivePrices {
  const [snapshot, setSnapshot] = useState<PriceSnapshot>();
  const [status, setStatus] = useState<PriceStreamStatus>('live');

  useEffect(() => gateway.subscribe({ onSnapshot: setSnapshot, onStatus: setStatus }), [gateway]);

  const prices = useMemo(
    () =>
      Object.fromEntries(
        (snapshot?.prices ?? []).map((entry) => [entry.positionId, entry] as const),
      ),
    [snapshot],
  );
  const stale = status === 'ausgefallen' || snapshot?.stale === true;
  const announcement = stale ? 'Kurse veraltet' : snapshot ? 'Kurse aktuell' : '';
  return { prices, stale, announcement };
}
