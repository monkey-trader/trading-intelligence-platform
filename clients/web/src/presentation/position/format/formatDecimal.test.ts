import { describe, expect, it } from 'vitest';
import { formatDecimal } from '@/presentation/position/format/formatDecimal';

// de-CH gruppiert mit Apostroph; je nach ICU-Version ' (U+0027) oder ’ (U+2019).
const GROUP = "['’]";

describe('formatDecimal', () => {
  it('gruppiert Tausender im Format de-CH mit Punkt als Dezimaltrenner', () => {
    expect(formatDecimal('1234567.5')).toMatch(new RegExp(`^1${GROUP}234${GROUP}567\\.5$`));
  });

  it('zeigt ohne Optionen alle Nachkommastellen exakt und lässt Nullen am Ende weg', () => {
    expect(formatDecimal('0.9600')).toBe('0.96');
    expect(formatDecimal('0.00001234')).toBe('0.00001234');
    expect(formatDecimal('0.865')).toBe('0.865');
    expect(formatDecimal('500')).toBe('500');
  });

  it('zeigt ganze Zahlen ohne Nachkommastellen', () => {
    expect(formatDecimal('4150.00')).toMatch(new RegExp(`^4${GROUP}150$`));
  });

  it('formatiert sehr grosse Werte verlustfrei aus dem String (ohne Number)', () => {
    expect(formatDecimal('9007199254740993.25')).toMatch(
      new RegExp(`^9${GROUP}007${GROUP}199${GROUP}254${GROUP}740${GROUP}993\\.25$`),
    );
  });

  it('rundet mit festen Nachkommastellen kaufmännisch', () => {
    const twoDigits = { minimumFractionDigits: 2, maximumFractionDigits: 2 };
    expect(formatDecimal('25.0000', twoDigits)).toBe('25.00');
    expect(formatDecimal('1.005', twoDigits)).toBe('1.01');
    expect(formatDecimal('18.745', twoDigits)).toBe('18.75');
  });

  it('behält das Minuszeichen bei negativen Werten', () => {
    expect(
      formatDecimal('-6.2500', { minimumFractionDigits: 2, maximumFractionDigits: 2 }),
    ).toMatch(/^[-−]6\.25$/);
  });

  it('formatiert 0', () => {
    expect(formatDecimal('0')).toBe('0');
    expect(formatDecimal('0', { minimumFractionDigits: 2, maximumFractionDigits: 2 })).toBe('0.00');
  });
});

describe('formatDecimal – kein "-0.00"', () => {
  it.each([
    ['-0', '0.00'],
    ['-0.0000', '0.00'],
    ['-0.004', '0.00'],
    ['-0.005', '-0.01'],
    ['0.004', '0.00'],
  ])('formatiert %s mit zwei Stellen als %s', (value, expected) => {
    expect(formatDecimal(value, { minimumFractionDigits: 2, maximumFractionDigits: 2 })).toBe(
      expected,
    );
  });
});
