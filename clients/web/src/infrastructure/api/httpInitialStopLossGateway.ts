import type {
  InitialStopLossGateway,
  InitialStopLossProblemType,
  InitialStopLossResult,
} from '@/domain/position/InitialStopLossGateway';

const PROBLEM_CONTENT_TYPE = 'application/problem+json';
const KNOWN_PROBLEM_TYPES: readonly InitialStopLossProblemType[] = [
  'invalid-stop-loss',
  'not-found',
  'bad-request',
  'forbidden-origin',
  'not-configured',
  'rate-limited',
  'unreachable',
  'upstream-error',
  'invalid-response',
  'internal',
];

/** Gateway auf PUT/DELETE /positions/{positionId}/initial-stop-loss des eigenen Backends. */
export function createHttpInitialStopLossGateway(
  fetchFn: typeof fetch = fetch,
  baseUrl = '/api',
): InitialStopLossGateway {
  const urlFor = (positionId: string) =>
    `${baseUrl}/positions/${encodeURIComponent(positionId)}/initial-stop-loss`;

  return {
    async setInitialStopLoss(positionId, value) {
      const response = await send(fetchFn, urlFor(positionId), {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
        body: JSON.stringify({ initialStopLoss: value }),
      });
      if (!(response instanceof Response)) return response;
      if (!response.ok) return readProblem(response);
      return (await hasValidBody(response)) ? { kind: 'ok' } : problem('invalid-body');
    },
    async clearInitialStopLoss(positionId) {
      const response = await send(fetchFn, urlFor(positionId), { method: 'DELETE' });
      if (!(response instanceof Response)) return response;
      return response.ok ? { kind: 'ok' } : readProblem(response);
    },
  };
}

async function send(
  fetchFn: typeof fetch,
  url: string,
  init: RequestInit,
): Promise<Response | InitialStopLossResult> {
  try {
    return await fetchFn(url, init);
  } catch {
    return problem('backend-unreachable');
  }
}

function problem(problemType: InitialStopLossProblemType): InitialStopLossResult {
  return { kind: 'problem', problemType };
}

async function hasValidBody(response: Response): Promise<boolean> {
  try {
    const body: unknown = await response.json();
    return (
      typeof body === 'object' &&
      body !== null &&
      'positionId' in body &&
      typeof body.positionId === 'string'
    );
  } catch {
    return false;
  }
}

async function readProblem(response: Response): Promise<InitialStopLossResult> {
  const contentType = (response.headers.get('Content-Type') ?? '').toLowerCase();
  // Fehler ohne problem+json kommen nicht von unserem Backend (z.B. Proxy ohne Backend).
  if (!contentType.startsWith(PROBLEM_CONTENT_TYPE)) return problem('backend-unreachable');
  try {
    const body: unknown = await response.json();
    const type =
      typeof body === 'object' && body !== null && 'type' in body ? body.type : undefined;
    const known = KNOWN_PROBLEM_TYPES.find((candidate) => candidate === type);
    return problem(known ?? 'invalid-body');
  } catch {
    return problem('invalid-body');
  }
}
