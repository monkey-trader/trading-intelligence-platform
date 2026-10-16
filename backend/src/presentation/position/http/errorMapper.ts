import { OpenPositionsUnavailableError } from '@/domain/position/errors/OpenPositionsUnavailableError';
import {
  createProblem,
  type Problem,
  type ProblemStatus,
  type ProblemType,
} from '@/presentation/http/problem';

export type MappedOpenPositionsError = {
  status: ProblemStatus;
  problem: Problem;
  retryAfterSeconds?: number;
};

const RATE_LIMIT_RETRY_AFTER_SECONDS = 30;

/** Übersetzt einen Fehler beim Laden der offenen Positionen in HTTP-Status und Problem-Body. */
export function mapOpenPositionsError(error: unknown): MappedOpenPositionsError {
  const problemType: ProblemType =
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
