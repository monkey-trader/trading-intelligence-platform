import { describe, expect, it } from 'vitest';
import {
  pick,
  requireBoolean,
  requireNumber,
  requireString,
} from '@/infrastructure/bingx/rawFields';
import { UnexpectedBingXResponseError } from '@/infrastructure/bingx/errors';

function captureError(action: () => unknown): unknown {
  try {
    action();
  } catch (error) {
    return error;
  }
  throw new Error('Es wurde ein Fehler erwartet, aber keiner geworfen.');
}

describe('pick', () => {
  it('liefert den Wert des ersten vorhandenen Kandidaten', () => {
    const raw = { avgOpenPrice: '2', avgPrice: '1' };

    expect(pick(raw, 'avgPrice', 'avgOpenPrice')).toBe('1');
  });

  it('überspringt fehlende Kandidaten', () => {
    const raw = { avgOpenPrice: '2' };

    expect(pick(raw, 'avgPrice', 'avgOpenPrice')).toBe('2');
  });

  it('liefert undefined, wenn kein Kandidat vorhanden ist', () => {
    expect(pick({ other: 1 }, 'avgPrice', 'avgOpenPrice')).toBeUndefined();
  });

  it('wertet null als vorhandenen Wert (nur undefined gilt als fehlend)', () => {
    const raw = { avgPrice: null, avgOpenPrice: '2' };

    expect(pick(raw, 'avgPrice', 'avgOpenPrice')).toBeNull();
  });
});

describe('requireString', () => {
  it('liefert den String des ersten vorhandenen Kandidaten', () => {
    expect(requireString({ positionAmt: '500' }, 'positionAmt', 'positionAmount')).toBe('500');
    expect(requireString({ positionAmount: '7' }, 'positionAmt', 'positionAmount')).toBe('7');
  });

  it.each([
    ['fehlt', {}],
    ['eine Zahl ist', { positionAmt: 500 }],
    ['null ist', { positionAmt: null }],
  ])('wirft UnexpectedBingXResponseError mit Feldname, wenn das Feld %s', (_description, raw) => {
    const caught = captureError(() => requireString(raw, 'positionAmt', 'positionAmount'));

    expect(caught).toBeInstanceOf(UnexpectedBingXResponseError);
    expect(caught instanceof Error ? caught.message : '').toContain('positionAmt');
    expect(caught instanceof Error ? caught.message : '').toContain('positionAmount');
  });
});

describe('requireNumber', () => {
  it('liefert die Zahl des ersten vorhandenen Kandidaten', () => {
    expect(requireNumber({ createTime: 1791500000000 }, 'createTime', 'openTime')).toBe(
      1791500000000,
    );
  });

  it.each([
    ['fehlt', {}],
    ['ein String ist', { leverage: '10' }],
  ])('wirft UnexpectedBingXResponseError mit Feldname, wenn das Feld %s', (_description, raw) => {
    const caught = captureError(() => requireNumber(raw, 'leverage'));

    expect(caught).toBeInstanceOf(UnexpectedBingXResponseError);
    expect(caught instanceof Error ? caught.message : '').toContain('leverage');
  });
});

describe('requireBoolean', () => {
  it('liefert den Boolean, auch wenn er false ist', () => {
    expect(requireBoolean({ isolated: false }, 'isolated')).toBe(false);
    expect(requireBoolean({ isolated: true }, 'isolated')).toBe(true);
  });

  it.each([
    ['fehlt', {}],
    ['ein String ist', { isolated: 'true' }],
  ])('wirft UnexpectedBingXResponseError mit Feldname, wenn das Feld %s', (_description, raw) => {
    const caught = captureError(() => requireBoolean(raw, 'isolated'));

    expect(caught).toBeInstanceOf(UnexpectedBingXResponseError);
    expect(caught instanceof Error ? caught.message : '').toContain('isolated');
  });
});
