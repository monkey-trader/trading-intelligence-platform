import { describe, expect, it } from 'vitest';
import { createProblem, type ProblemType } from '@/presentation/http/problem';

describe('createProblem', () => {
  it.each([
    ['bad-request', 400],
    ['forbidden-host', 403],
    ['not-found', 404],
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

  it('liefert für not-found einen deutschen, nicht leeren Titel und nur type, title, status', () => {
    const problem = createProblem('not-found');

    expect(problem.title.trim().length).toBeGreaterThan(0);
    expect(Object.keys(problem).sort()).toEqual(['status', 'title', 'type']);
  });

  it('liefert für bad-request Status 400, einen deutschen Titel und nur type, title, status', () => {
    const problem = createProblem('bad-request');

    expect(problem).toEqual({ type: 'bad-request', title: expect.any(String), status: 400 });
    expect(Object.keys(problem).sort()).toEqual(['status', 'title', 'type']);
    expect(problem.title.trim().length).toBeGreaterThan(0);
    expect(problem.title).toMatch(/[äöüÄÖÜ]|\b(die|der|das|ist|nicht|ungültig|Anfrage)\b/i);
  });

  it('liefert bei jedem Aufruf ein neues Objekt', () => {
    expect(createProblem('internal')).not.toBe(createProblem('internal'));
  });
});
