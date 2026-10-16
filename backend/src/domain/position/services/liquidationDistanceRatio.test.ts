import { describe, expect, it } from 'vitest';
import { liquidationDistanceRatio } from '@/domain/position/services/liquidationDistanceRatio';
import { InvalidDecimalError } from '@/domain/shared/decimal/InvalidDecimalError';

describe('liquidationDistanceRatio', () => {
  it('berechnet LONG als (mark − liq) / mark mit 8 Nachkommastellen', () => {
    // (81365.0 − 76507) / 81365.0 = 4858 / 81365 = 0.0597062619… → 0.05970626
    expect(liquidationDistanceRatio('LONG', '81365.0', '76507')).toBe('0.05970626');
  });

  it('rundet LONG kaufmännisch auf 8 Stellen', () => {
    // (1.0100 − 0.865) / 1.0100 = 0.1435643564… → 0.14356436
    expect(liquidationDistanceRatio('LONG', '1.0100', '0.865')).toBe('0.14356436');
  });

  it('berechnet SHORT als (liq − mark) / mark', () => {
    // (4410.5 − 4175.00) / 4175.00 = 0.0564071856… → 0.05640719
    expect(liquidationDistanceRatio('SHORT', '4175.00', '4410.5')).toBe('0.05640719');
  });

  it('liefert für LONG einen negativen Abstand, wenn Mark unter dem Liquidationspreis liegt', () => {
    // (70000 − 76507) / 70000 = −0.0929571428… → −0.09295714
    expect(liquidationDistanceRatio('LONG', '70000', '76507')).toBe('-0.09295714');
  });

  it('liefert für SHORT einen negativen Abstand, wenn Mark über dem Liquidationspreis liegt', () => {
    // (4410.5 − 4500) / 4500 = −0.0198888… → −0.01988889
    expect(liquidationDistanceRatio('SHORT', '4500', '4410.5')).toBe('-0.01988889');
  });

  it('liefert "0", wenn Mark genau auf dem Liquidationspreis liegt', () => {
    expect(liquidationDistanceRatio('LONG', '100.0', '100')).toBe('0');
    expect(liquidationDistanceRatio('SHORT', '100', '100.00')).toBe('0');
  });

  it.each(['LONG', 'SHORT'] as const)(
    'liefert null bei %s ohne Liquidationspreis ("0")',
    (side) => {
      expect(liquidationDistanceRatio(side, '81365.0', '0')).toBeNull();
    },
  );

  it('behandelt einen numerisch nullwertigen Liquidationspreis ("0.000") ebenfalls als keinen', () => {
    expect(liquidationDistanceRatio('LONG', '81365.0', '0.000')).toBeNull();
  });

  it.each(['LONG', 'SHORT'] as const)('liefert null bei %s mit Mark-Preis "0"', (side) => {
    expect(liquidationDistanceRatio(side, '0', '76507')).toBeNull();
  });

  it('wirft InvalidDecimalError bei ungültigem Preis', () => {
    expect(() => liquidationDistanceRatio('LONG', 'abc', '76507')).toThrow(InvalidDecimalError);
    expect(() => liquidationDistanceRatio('SHORT', '81365.0', '1e3')).toThrow(InvalidDecimalError);
  });

  it.each(['-1', '-0.5'])(
    'liefert null bei negativem Mark-Preis %s (kein sinnvoller Abstand)',
    (markPrice) => {
      expect(liquidationDistanceRatio('LONG', markPrice, '1')).toBeNull();
      expect(liquidationDistanceRatio('SHORT', markPrice, '1')).toBeNull();
    },
  );

  it('liefert null bei negativem Liquidationspreis (ungültige Angabe von BingX)', () => {
    expect(liquidationDistanceRatio('LONG', '100', '-5')).toBeNull();
  });
});
