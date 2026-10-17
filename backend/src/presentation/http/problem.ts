import type { OpenPositionsUnavailableReason } from '@/domain/position/errors/OpenPositionsUnavailableError';

/** Problem-Typen laut docs/openapi.yaml. */
export type ProblemType =
  'bad-request' | 'forbidden-host' | 'not-found' | OpenPositionsUnavailableReason | 'internal';

/** HTTP-Status, mit denen ein Problem ausgeliefert wird (siehe docs/openapi.yaml). */
export type ProblemStatus = 400 | 403 | 404 | 500 | 502 | 503 | 504;

/** Problem-Objekt nach docs/openapi.yaml. Enthält nie interne Details oder den cause. */
export type Problem = {
  type: ProblemType;
  title: string;
  status: ProblemStatus;
  detail?: string;
};

export const PROBLEM_CONTENT_TYPE = 'application/problem+json';

const PROBLEMS = {
  'bad-request': { status: 400, title: 'Die Anfrage ist ungültig.' },
  'forbidden-host': { status: 403, title: 'Der Host-Header ist nicht erlaubt.' },
  'not-found': { status: 404, title: 'Die angefragte Ressource gibt es nicht.' },
  'not-configured': { status: 503, title: 'BingX-Zugang ist nicht konfiguriert.' },
  'rate-limited': { status: 503, title: 'BingX ist ausgelastet, bitte später erneut versuchen.' },
  unreachable: { status: 504, title: 'BingX ist nicht erreichbar.' },
  'upstream-error': { status: 502, title: 'BingX hat mit einem Fehler geantwortet.' },
  'invalid-response': { status: 502, title: 'BingX hat eine ungültige Antwort geliefert.' },
  internal: { status: 500, title: 'Interner Fehler.' },
} as const satisfies Readonly<
  Record<ProblemType, Readonly<{ status: ProblemStatus; title: string }>>
>;

/** HTTP-Status, der fest zu einem Problem-Typ gehört (z.B. 404 für 'not-found'). */
export type StatusOfProblem<Type extends ProblemType> = (typeof PROBLEMS)[Type]['status'];

/**
 * Baut ein neues Problem-Objekt mit festem Titel und Status zum Typ. Der Status ist auf die
 * Typen eingeschränkt, damit Routen nur Status liefern, die ihr Response-Schema kennt.
 */
export function createProblem<Type extends ProblemType>(
  problemType: Type,
): Problem & { type: Type; status: StatusOfProblem<Type> } {
  const { status, title }: { status: StatusOfProblem<Type>; title: string } = PROBLEMS[problemType];
  return { type: problemType, title, status };
}

/** JSON-Schema für Fastify, entspricht `Problem` in docs/openapi.yaml. */
export const PROBLEM_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['type', 'title', 'status'],
  properties: {
    type: { type: 'string', enum: Object.keys(PROBLEMS) },
    title: { type: 'string' },
    status: { type: 'integer' },
    detail: { type: 'string' },
  },
} as const;
