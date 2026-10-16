import { describe, expect, it } from 'vitest';
import { pnlTone } from '@/presentation/position/format/pnlTone';

describe('pnlTone', () => {
  it('ist positive bei Werten grösser als 0', () => {
    expect(pnlTone('25.0000')).toBe('positive');
    expect(pnlTone('0.00000001')).toBe('positive');
    expect(pnlTone('18.75')).toBe('positive');
  });

  it('ist negative bei Werten kleiner als 0', () => {
    expect(pnlTone('-6.2500')).toBe('negative');
    expect(pnlTone('-0.00000001')).toBe('negative');
  });

  it.each(['0', '0.00', '-0', '-0.000', '0.0000'])('ist neutral bei %s', (zeroValue) => {
    expect(pnlTone(zeroValue)).toBe('neutral');
  });
});

describe('pnlTone mit angezeigter Genauigkeit', () => {
  it.each([
    ['-0.004', 2, 'neutral'],
    ['0.004', 2, 'neutral'],
    ['-0.005', 2, 'negative'],
    ['0.005', 2, 'positive'],
    ['-0.00004', 4, 'neutral'],
    ['-12.5', 2, 'negative'],
  ] as const)('%s auf %i Stellen ist %s', (value, fractionDigits, expected) => {
    expect(pnlTone(value, fractionDigits)).toBe(expected);
  });

  it('ohne Genauigkeit zählt jedes Vorzeichen', () => {
    expect(pnlTone('-0.004')).toBe('negative');
  });
});
