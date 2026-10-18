import { describe, expect, it } from 'vitest';
import type { PositionChartProblemType } from '@/domain/position/PositionChartResult';
import { mapOpenPositionsProblem } from '@/presentation/position/errorMapper';
import { mapPositionChartProblem } from '@/presentation/position/chartErrorMapper';

describe('mapPositionChartProblem', () => {
  it('meldet eine unbekannte oder geschlossene Position', () => {
    expect(mapPositionChartProblem('not-found')).toMatch(/position/i);
    expect(mapPositionChartProblem('not-found')).toMatch(
      /nicht gefunden|nicht mehr offen|geschlossen/i,
    );
  });

  it('meldet eine ungültige Anfrage', () => {
    expect(mapPositionChartProblem('bad-request')).not.toBe('');
  });

  it('nutzt für die übrigen Probleme dieselben Meldungen wie die Positionsliste', () => {
    expect(mapPositionChartProblem('upstream-error')).toBe(
      mapOpenPositionsProblem('upstream-error'),
    );
    expect(mapPositionChartProblem('backend-unreachable')).toBe(
      mapOpenPositionsProblem('backend-unreachable'),
    );
  });

  it('nennt bei rate-limited die Wartezeit', () => {
    expect(mapPositionChartProblem('rate-limited', 12)).toBe(
      'BingX ausgelastet, neuer Versuch in 12 s.',
    );
  });

  it('liefert für jeden Problem-Typ eine nicht leere Meldung', () => {
    const problemTypes: readonly PositionChartProblemType[] = [
      'bad-request',
      'not-found',
      'forbidden-host',
      'not-configured',
      'rate-limited',
      'unreachable',
      'upstream-error',
      'invalid-response',
      'internal',
      'backend-unreachable',
      'invalid-body',
    ];
    for (const problemType of problemTypes) {
      expect(mapPositionChartProblem(problemType)).not.toBe('');
    }
  });
});
