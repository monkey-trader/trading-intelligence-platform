import type { OpenPositionsSnapshot } from '@/domain/position/OpenPosition';

/** Problem-Typen des Backends (laut API-Vertrag). */
export type BackendProblemType =
  | 'forbidden-host'
  | 'not-configured'
  | 'rate-limited'
  | 'unreachable'
  | 'upstream-error'
  | 'invalid-response'
  | 'internal';

/**
 * Alle Probleme, die der Client unterscheidet:
 * - backend-unreachable: Backend antwortet nicht (oder nicht mit problem+json).
 * - invalid-body: Antwort passt nicht zum API-Vertrag.
 */
export type OpenPositionsProblemType = BackendProblemType | 'backend-unreachable' | 'invalid-body';

export type OpenPositionsResult =
  | Readonly<{ kind: 'ok'; snapshot: OpenPositionsSnapshot }>
  | Readonly<{
      kind: 'problem';
      problemType: OpenPositionsProblemType;
      /** Sekunden bis zum nächsten Versuch, nur wenn das Backend Retry-After schickt. */
      retryAfterSeconds?: number;
    }>;
