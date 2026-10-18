import type { PositionChart } from '@/domain/position/PositionChart';
import type { OpenPositionsProblemType } from '@/domain/position/OpenPositionsResult';

/** Probleme beim Laden eines Charts: die der Positionsliste plus ungültige Anfrage und unbekannte Position. */
export type PositionChartProblemType = OpenPositionsProblemType | 'bad-request' | 'not-found';

export type PositionChartResult =
  | Readonly<{ kind: 'ok'; chart: PositionChart }>
  | Readonly<{
      kind: 'problem';
      problemType: PositionChartProblemType;
      /** Sekunden bis zum nächsten Versuch, nur wenn das Backend Retry-After schickt. */
      retryAfterSeconds?: number | undefined;
    }>;
