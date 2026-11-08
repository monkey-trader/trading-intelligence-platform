import type {
  SwingLongGateway,
  SwingLongProblemType,
  SwingLongResult,
} from '@/domain/position/SwingLongGateway';

const PROBLEM_CONTENT_TYPE = 'application/problem+json';
const KNOWN_PROBLEMS: readonly SwingLongProblemType[] = [
  'bad-request',
  'forbidden-origin',
  'internal',
];

/** Gateway auf PUT/DELETE /positions/{positionId}/swing-long. */
export function createHttpSwingLongGateway(
  fetchFn: typeof fetch = fetch,
  baseUrl = '/api',
): SwingLongGateway {
  const send = async (method: 'PUT' | 'DELETE', positionId: string): Promise<SwingLongResult> => {
    let response: Response;
    try {
      response = await fetchFn(
        `${baseUrl}/positions/${encodeURIComponent(positionId)}/swing-long`,
        {
          method,
          headers: { Accept: 'application/json' },
        },
      );
    } catch {
      return { kind: 'problem', problemType: 'backend-unreachable' };
    }
    if (response.ok) return { kind: 'ok' };
    const contentType = response.headers.get('Content-Type') ?? '';
    if (!contentType.toLowerCase().startsWith(PROBLEM_CONTENT_TYPE)) {
      return { kind: 'problem', problemType: 'backend-unreachable' };
    }
    return { kind: 'problem', problemType: await readProblemType(response) };
  };
  return {
    setSwingLong: (positionId) => send('PUT', positionId),
    clearSwingLong: (positionId) => send('DELETE', positionId),
  };
}

async function readProblemType(response: Response): Promise<SwingLongProblemType> {
  try {
    const body: unknown = await response.json();
    const type =
      typeof body === 'object' && body !== null && 'type' in body ? body.type : undefined;
    return KNOWN_PROBLEMS.find((known) => known === type) ?? 'invalid-body';
  } catch {
    return 'invalid-body';
  }
}
