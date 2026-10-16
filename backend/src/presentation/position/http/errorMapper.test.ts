import { describe, expect, it } from 'vitest';
import { mapOpenPositionsError } from '@/presentation/position/http/errorMapper';
import { OpenPositionsUnavailableError } from '@/domain/position/errors/OpenPositionsUnavailableError';
import type { OpenPositionsUnavailableReason } from '@/domain/position/errors/OpenPositionsUnavailableError';

const PROBLEM_FIELDS = ['type', 'title', 'status', 'detail'];
const SECRET_CAUSE_TEXT = 'geheim-xyz';

describe('mapOpenPositionsError', () => {
  it.each([
    ['not-configured', 503],
    ['rate-limited', 503],
    ['unreachable', 504],
    ['upstream-error', 502],
    ['invalid-response', 502],
  ] as const)(
    'bildet reason %s auf Status %i ab',
    (reason: OpenPositionsUnavailableReason, expectedStatus) => {
      const mapped = mapOpenPositionsError(new OpenPositionsUnavailableError(reason));

      expect(mapped.status).toBe(expectedStatus);
      expect(mapped.problem.type).toBe(reason);
      expect(mapped.problem.status).toBe(expectedStatus);
      expect(typeof mapped.problem.title).toBe('string');
      expect(mapped.problem.title.length).toBeGreaterThan(0);
    },
  );

  it('setzt retryAfterSeconds 30 nur bei rate-limited', () => {
    const mapped = mapOpenPositionsError(new OpenPositionsUnavailableError('rate-limited'));

    expect(mapped.retryAfterSeconds).toBe(30);
  });

  it.each(['not-configured', 'unreachable', 'upstream-error', 'invalid-response'] as const)(
    'setzt kein retryAfterSeconds bei %s',
    (reason) => {
      const mapped = mapOpenPositionsError(new OpenPositionsUnavailableError(reason));

      expect(mapped.retryAfterSeconds).toBeUndefined();
    },
  );

  it('bildet einen beliebigen anderen Fehler auf 500 internal ab', () => {
    const mapped = mapOpenPositionsError(new Error('Programmierfehler geheim-abc'));

    expect(mapped.status).toBe(500);
    expect(mapped.problem.type).toBe('internal');
    expect(mapped.problem.status).toBe(500);
    expect(mapped.retryAfterSeconds).toBeUndefined();
    expect(JSON.stringify(mapped.problem)).not.toContain('geheim-abc');
  });

  it('bildet auch Nicht-Error-Werte auf 500 internal ab', () => {
    const mapped = mapOpenPositionsError('kaputt');

    expect(mapped.status).toBe(500);
    expect(mapped.problem.type).toBe('internal');
  });

  it('liefert ein problem, das dem Problem-Schema entspricht (keine weiteren Felder)', () => {
    const errors: unknown[] = [
      new OpenPositionsUnavailableError('not-configured'),
      new OpenPositionsUnavailableError('rate-limited'),
      new OpenPositionsUnavailableError('unreachable'),
      new OpenPositionsUnavailableError('upstream-error'),
      new OpenPositionsUnavailableError('invalid-response'),
      new Error('irgendwas'),
    ];

    for (const error of errors) {
      const { problem } = mapOpenPositionsError(error);
      for (const fieldName of Object.keys(problem)) {
        expect(PROBLEM_FIELDS).toContain(fieldName);
      }
      if (problem.detail !== undefined) expect(typeof problem.detail).toBe('string');
    }
  });

  it('gibt den Text von cause nicht im problem preis', () => {
    const cause = new Error(`BingX-Fehler mit ${SECRET_CAUSE_TEXT}`);

    const mapped = mapOpenPositionsError(
      new OpenPositionsUnavailableError('upstream-error', cause),
    );

    expect(JSON.stringify(mapped.problem)).not.toContain(SECRET_CAUSE_TEXT);
  });
});
