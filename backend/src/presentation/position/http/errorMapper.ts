import {
  OpenPositionsUnavailableError,
  type OpenPositionsUnavailableReason,
} from '@/domain/position/errors/OpenPositionsUnavailableError';

export type ProblemType = OpenPositionsUnavailableReason | 'internal';

/** HTTP-Status, mit denen ein Problem ausgeliefert wird (siehe docs/openapi.yaml). */
export type ProblemStatus = 500 | 502 | 503 | 504;

/** Problem-Objekt nach docs/openapi.yaml. Enthält nie interne Details oder den cause. */
export type Problem = {
  type: ProblemType;
  title: string;
  status: ProblemStatus;
  detail?: string;
};

export type MappedOpenPositionsError = {
  status: ProblemStatus;
  problem: Problem;
  retryAfterSeconds?: number;
};

const RATE_LIMIT_RETRY_AFTER_SECONDS = 30;

const PROBLEMS: Readonly<Record<ProblemType, Readonly<{ status: ProblemStatus; title: string }>>> =
  {
    'not-configured': { status: 503, title: 'BingX-Zugang ist nicht konfiguriert.' },
    'rate-limited': { status: 503, title: 'BingX ist ausgelastet, bitte später erneut versuchen.' },
    unreachable: { status: 504, title: 'BingX ist nicht erreichbar.' },
    'upstream-error': { status: 502, title: 'BingX hat mit einem Fehler geantwortet.' },
    'invalid-response': { status: 502, title: 'BingX hat eine ungültige Antwort geliefert.' },
    internal: { status: 500, title: 'Interner Fehler.' },
  };

/** Übersetzt einen Fehler beim Laden der offenen Positionen in HTTP-Status und Problem-Body. */
export function mapOpenPositionsError(error: unknown): MappedOpenPositionsError {
  const problemType: ProblemType =
    error instanceof OpenPositionsUnavailableError ? error.reason : 'internal';
  const { status, title } = PROBLEMS[problemType];
  const problem: Problem = { type: problemType, title, status };
  if (problemType === 'rate-limited') {
    return { status, problem, retryAfterSeconds: RATE_LIMIT_RETRY_AFTER_SECONDS };
  }
  return { status, problem };
}
