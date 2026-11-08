import { describe, expect, it } from 'vitest';
import type { SwingLongProblemType } from '@/domain/position/SwingLongGateway';
import { mapSwingLongProblem } from '@/presentation/position/swingLongErrorMapper';

const TYPES: readonly SwingLongProblemType[] = [
  'bad-request',
  'forbidden-origin',
  'internal',
  'backend-unreachable',
  'invalid-body',
];

describe('mapSwingLongProblem', () => {
  it.each(TYPES)('liefert für %s eine nicht leere Meldung', (type) => {
    expect(mapSwingLongProblem(type).trim()).not.toBe('');
  });

  it('nennt bei nicht erreichbarem Backend den Server', () => {
    expect(mapSwingLongProblem('backend-unreachable')).toMatch(/Backend nicht erreichbar/);
  });

  it('unterscheidet die Meldungen je Problem', () => {
    expect(new Set(TYPES.map(mapSwingLongProblem)).size).toBe(TYPES.length);
  });
});
