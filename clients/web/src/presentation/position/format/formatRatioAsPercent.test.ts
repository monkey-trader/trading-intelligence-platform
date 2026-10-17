import { describe, expect, it } from 'vitest';
import {
  formatRatioAsPercent,
  type RatioFormatOptions,
} from '@/presentation/position/format/formatRatioAsPercent';

// Zwischen Zahl und % darf ein (geschütztes) Leerzeichen stehen, muss aber nicht.
const PERCENT = '[\\s\\u00a0\\u202f]?%';

describe('formatRatioAsPercent', () => {
  it('zeigt ein Verhältnis als Prozent mit zwei Nachkommastellen', () => {
    expect(formatRatioAsPercent('0.05970626')).toMatch(new RegExp(`^5\\.97${PERCENT}$`));
    expect(formatRatioAsPercent('0.14356436')).toMatch(new RegExp(`^14\\.36${PERCENT}$`));
    expect(formatRatioAsPercent('0.5208')).toMatch(new RegExp(`^52\\.08${PERCENT}$`));
  });

  it('zeigt negative Verhältnisse mit Minuszeichen', () => {
    expect(formatRatioAsPercent('-0.0904')).toMatch(new RegExp(`^[-−]9\\.04${PERCENT}$`));
  });

  it('gruppiert grosse Prozentwerte im Format de-CH', () => {
    expect(formatRatioAsPercent('12.3456')).toMatch(new RegExp(`^1['’]234\\.56${PERCENT}$`));
  });

  it('zeigt 0 als 0.00 %', () => {
    expect(formatRatioAsPercent('0')).toMatch(new RegExp(`^0\\.00${PERCENT}$`));
  });

  it('zeigt einen Gedankenstrich, wenn kein Verhältnis vorliegt (null)', () => {
    expect(formatRatioAsPercent(null)).toBe('–');
  });
});

describe('formatRatioAsPercent – kein "-0.00 %"', () => {
  it('zeigt einen winzigen negativen Wert als 0.00 %', () => {
    expect(formatRatioAsPercent('-0.00004').replace(/\s/g, ' ')).toBe('0.00 %');
  });
});

describe('formatRatioAsPercent – explizites Vorzeichen (signDisplay: exceptZero)', () => {
  // \s deckt auch geschützte Leerzeichen (U+00A0, U+202F) ab.
  const signed: RatioFormatOptions = { signDisplay: 'exceptZero' };
  const normalizeSpaces = (text: string) => text.replace(/\s/g, ' ');

  it('zeigt positive Prozente mit "+"', () => {
    expect(normalizeSpaces(formatRatioAsPercent('0.5208', signed))).toBe('+52.08 %');
    expect(normalizeSpaces(formatRatioAsPercent('0.05970626', signed))).toBe('+5.97 %');
  });

  it('zeigt negative Prozente mit "-"', () => {
    expect(normalizeSpaces(formatRatioAsPercent('-0.0904', signed))).toBe('-9.04 %');
  });

  it('gruppiert grosse Prozentwerte im Format de-CH mit "+"', () => {
    expect(formatRatioAsPercent('12.3456', signed)).toMatch(
      new RegExp(`^\\+1['’]234\\.56${PERCENT}$`),
    );
  });

  it.each(['0', '-0', '0.00004', '-0.00004'])(
    'zeigt %s, gerundet 0, ohne Vorzeichen als 0.00 %%',
    (ratio) => {
      expect(normalizeSpaces(formatRatioAsPercent(ratio, signed))).toBe('0.00 %');
    },
  );

  it('setzt das Vorzeichen nach der Rundung: 0.00005 → +0.01 %', () => {
    expect(normalizeSpaces(formatRatioAsPercent('0.00005', signed))).toBe('+0.01 %');
    expect(normalizeSpaces(formatRatioAsPercent('-0.00005', signed))).toBe('-0.01 %');
  });

  it('zeigt weiterhin einen Gedankenstrich ohne Verhältnis (null)', () => {
    expect(formatRatioAsPercent(null, signed)).toBe('–');
  });

  it('bleibt ohne Option beim bisherigen Verhalten (kein "+")', () => {
    expect(normalizeSpaces(formatRatioAsPercent('0.14356436'))).toBe('14.36 %');
  });
});
