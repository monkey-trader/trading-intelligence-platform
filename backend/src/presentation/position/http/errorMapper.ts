import {
  OpenPositionsUnavailableError,
  type OpenPositionsUnavailableReason,
} from '@/domain/position/errors/OpenPositionsUnavailableError';
import { createProblem, type Problem, type StatusOfProblem } from '@/presentation/http/problem';

/** Problem-Typen, die GET /positions/open aus dem Handler liefern kann. */
type OpenPositionsProblemType = OpenPositionsUnavailableReason | 'internal';

export type MappedOpenPositionsError = {
  status: StatusOfProblem<OpenPositionsProblemType>;
  problem: Problem;
  retryAfterSeconds?: number;
};

const RATE_LIMIT_RETRY_AFTER_SECONDS = 30;

/** Übersetzt einen Fehler beim Laden der offenen Positionen in HTTP-Status und Problem-Body. */
export function mapOpenPositionsError(error: unknown): MappedOpenPositionsError {
  const problemType: OpenPositionsProblemType =
    error instanceof OpenPositionsUnavailableError ? error.reason : 'internal';
  const problem = createProblem(problemType);
  if (problemType === 'rate-limited') {
    return {
      status: problem.status,
      problem,
      retryAfterSeconds: RATE_LIMIT_RETRY_AFTER_SECONDS,
    };
  }
  return { status: problem.status, problem };
}
