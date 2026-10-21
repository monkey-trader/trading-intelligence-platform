import { describe, expect, it } from 'vitest';
import { mapInitialStopLossProblem } from '@/presentation/position/initialStopLossErrorMapper';

// Erwartet: mapInitialStopLossProblem(problemType: InitialStopLossProblemType): string

describe('mapInitialStopLossProblem', () => {
  it('422: SL muss auf der Verlustseite liegen', () => {
    expect(mapInitialStopLossProblem('invalid-stop-loss')).toBe(
      'SL muss auf der Verlustseite liegen',
    );
  });

  it.each([
    ['not-found', /nicht (mehr )?offen/i],
    ['bad-request', /ungültig/i],
    ['backend-unreachable', /nicht erreichbar/i],
    ['forbidden-origin', /abgelehnt/i],
  ] as const)('%s hat eine verständliche Meldung', (problemType, pattern) => {
    expect(mapInitialStopLossProblem(problemType)).toMatch(pattern);
  });
});
