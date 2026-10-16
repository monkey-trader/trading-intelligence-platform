import { describe, expect, it } from 'vitest';
import { formatRatioAsPercent } from '@/presentation/position/format/formatRatioAsPercent';

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
