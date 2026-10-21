import { describe, expect, it } from 'vitest';
import { validateInitialStopLoss } from '@/presentation/position/initialStopLossValidation';

// Erwartet: validateInitialStopLoss(side: 'LONG' | 'SHORT', entry: string, raw: string): string | undefined
// (undefined = gültig, sonst Fehlertext). Eingabe wird getrimmt; Komma als Dezimaltrenner ist ungültig.

describe('validateInitialStopLoss', () => {
  it.each([
    ['LONG', '0.9600', '0.9'],
    ['LONG', '0.9600', ' 0.9599 '],
    ['SHORT', '0.9600', '1.02'],
  ] as const)('%s Entry %s, SL %s ist gültig', (side, entry, raw) => {
    expect(validateInitialStopLoss(side, entry, raw)).toBeUndefined();
  });

  it.each([
    ['leer', ''],
    ['nur Leerzeichen', '   '],
    ['keine Zahl', 'abc'],
    ['Komma', '0,9'],
    ['Exponent', '1e-1'],
    ['null', '0'],
    ['negativ', '-1'],
  ])('LONG: %s wird abgelehnt', (_label, raw) => {
    expect(validateInitialStopLoss('LONG', '0.9600', raw)).toEqual(expect.any(String));
  });

  it.each([
    ['LONG', '0.9600', '0.96'],
    ['LONG', '0.9600', '1'],
    ['SHORT', '0.9600', '0.96'],
    ['SHORT', '0.9600', '0.9'],
  ] as const)('%s Entry %s: SL %s liegt nicht auf der Verlustseite', (side, entry, raw) => {
    expect(validateInitialStopLoss(side, entry, raw)).toBe('SL muss auf der Verlustseite liegen');
  });
});
