import { describe, expect, it } from 'vitest';
import type { OpenPositionsProblemType } from '@/domain/position/OpenPositionsResult';
import { mapOpenPositionsProblem } from '@/presentation/position/errorMapper';

const ALL_PROBLEM_TYPES: readonly OpenPositionsProblemType[] = [
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

describe('mapOpenPositionsProblem', () => {
  it('meldet fehlende BingX-Zugangsdaten bei not-configured', () => {
    expect(mapOpenPositionsProblem('not-configured')).toBe(
      'BingX-Zugang fehlt (BX_KEY/BX_SECRET im Backend setzen).',
    );
  });

  it('nennt bei rate-limited die Wartezeit aus Retry-After', () => {
    expect(mapOpenPositionsProblem('rate-limited', 30)).toBe(
      'BingX ausgelastet, neuer Versuch in 30 s.',
    );
    expect(mapOpenPositionsProblem('rate-limited', 5)).toBe(
      'BingX ausgelastet, neuer Versuch in 5 s.',
    );
  });

  it('meldet rate-limited auch ohne Retry-After verständlich', () => {
    const message = mapOpenPositionsProblem('rate-limited');
    expect(message).toMatch(/^BingX ausgelastet/);
    expect(message).not.toMatch(/undefined|NaN/);
  });

  it('weist bei backend-unreachable auf den lokalen Backend-Start hin', () => {
    expect(mapOpenPositionsProblem('backend-unreachable')).toBe(
      'Backend nicht erreichbar (läuft der Server aus backend/?).',
    );
  });

  it('meldet die abgelehnte Host-Prüfung bei forbidden-host', () => {
    expect(mapOpenPositionsProblem('forbidden-host')).toBe(
      'Zugriff vom Backend abgelehnt (Host-Prüfung).',
    );
  });

  it('meldet bei unreachable, dass BingX nicht erreichbar ist', () => {
    expect(mapOpenPositionsProblem('unreachable')).toMatch(/BingX.*nicht erreichbar/);
  });

  it('nennt BingX bei upstream-error und invalid-response', () => {
    expect(mapOpenPositionsProblem('upstream-error')).toMatch(/BingX/);
    expect(mapOpenPositionsProblem('invalid-response')).toMatch(/BingX/);
  });

  it('nennt das Backend bei internal und invalid-body', () => {
    expect(mapOpenPositionsProblem('internal')).toMatch(/Backend/);
    expect(mapOpenPositionsProblem('invalid-body')).toMatch(/Backend/);
  });

  it.each(ALL_PROBLEM_TYPES)('liefert für %s eine nicht leere Meldung', (problemType) => {
    expect(mapOpenPositionsProblem(problemType).trim()).not.toBe('');
  });

  it('liefert für jeden Problem-Typ eine eigene Meldung', () => {
    const messages = ALL_PROBLEM_TYPES.map((problemType) => mapOpenPositionsProblem(problemType));
    expect(new Set(messages).size).toBe(ALL_PROBLEM_TYPES.length);
  });

  it('ignoriert Retry-After bei anderen Typen als rate-limited', () => {
    expect(mapOpenPositionsProblem('unreachable', 30)).toBe(mapOpenPositionsProblem('unreachable'));
  });
});
