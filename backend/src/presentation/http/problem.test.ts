import { describe, expect, it } from 'vitest';
import { createProblem, type ProblemType } from '@/presentation/http/problem';

describe('createProblem', () => {
  it.each([
    ['forbidden-host', 403],
    ['not-configured', 503],
    ['rate-limited', 503],
    ['unreachable', 504],
    ['upstream-error', 502],
    ['invalid-response', 502],
    ['internal', 500],
  ] as const)(
    'baut für %s ein Problem mit Status %i',
    (problemType: ProblemType, expectedStatus) => {
      const problem = createProblem(problemType);

      expect(problem.type).toBe(problemType);
      expect(problem.status).toBe(expectedStatus);
      expect(typeof problem.title).toBe('string');
      expect(problem.title.length).toBeGreaterThan(0);
    },
  );

  it('enthält nur type, title und status (keine internen Details)', () => {
    expect(Object.keys(createProblem('forbidden-host')).sort()).toEqual([
      'status',
      'title',
      'type',
    ]);
  });

  it('liefert bei jedem Aufruf ein neues Objekt', () => {
    expect(createProblem('internal')).not.toBe(createProblem('internal'));
  });
});
