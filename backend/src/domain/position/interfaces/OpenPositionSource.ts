import type { OpenPosition } from '@/domain/position/entities/OpenPosition';

/** Port: liefert die aktuell offenen Positionen. */
export type OpenPositionSource = {
  fetchOpenPositions(): Promise<OpenPosition[]>;
};
