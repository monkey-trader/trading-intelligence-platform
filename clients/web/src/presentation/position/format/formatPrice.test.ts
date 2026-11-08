import { describe, expect, it } from 'vitest';
import {
  formatChange24h,
  formatDistance,
  formatPrice,
} from '@/presentation/position/format/formatPrice';

describe('formatPrice', () => {
  it.each([
    ['0.0000123', '0.0000123'],
    ['123456.75', "123'456.75"],
    ['1.50', '1.5'],
  ])('%s -> %s (exakt, de-CH)', (input, expected) => {
    expect(formatPrice(input)).toBe(expected);
  });
  it('null -> –', () => expect(formatPrice(null)).toBe('–'));
});

describe('formatChange24h', () => {
  it('positiv mit +, Prozentzeichen und positiver Tönung', () => {
    const result = formatChange24h('1.3');
    expect(result.text).toMatch(/^\+1\.3\d*\u00a0?%$/);
    expect(result.tone).toBe('positive');
  });
  it('negativ mit Minus und negativer Tönung', () => {
    const result = formatChange24h('-0.5');
    expect(result.text).toMatch(/^[-−]0\.5\d*\u00a0?%$/);
    expect(result.tone).toBe('negative');
  });
  it('0 ist neutral und ohne Vorzeichen', () => {
    const result = formatChange24h('0');
    expect(result.text).not.toMatch(/[+\-−]/);
    expect(result.tone).toBe('neutral');
  });
  it('null -> – neutral', () => {
    expect(formatChange24h(null)).toEqual({ text: '–', tone: 'neutral' });
  });
});

describe('formatDistance', () => {
  it('positiv: Vorzeichen + und "offen"', () => {
    const result = formatDistance('2.5');
    expect(result.text).toMatch(/^\+2\.5\d*\u00a0?%$/);
    expect(result.label).toBe('offen');
  });
  it('negativ: Minus und "erreicht"', () => {
    const result = formatDistance('-1.2');
    expect(result.text).toMatch(/^[-−]1\.2\d*\u00a0?%$/);
    expect(result.label).toBe('erreicht');
  });
  it('genau 0 gilt als erreicht', () => {
    expect(formatDistance('0').label).toBe('erreicht');
  });
  it('null -> – ohne Label', () => {
    expect(formatDistance(null)).toEqual({ text: '–', label: null });
  });
});
