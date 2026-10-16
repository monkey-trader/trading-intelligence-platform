import { describe, expect, it } from 'vitest';
import { Leverage } from '@/domain/position/valueObjects/Leverage';
import { InvalidLeverageError } from '@/domain/position/errors/InvalidLeverageError';
import { PositionDomainError } from '@/domain/position/errors/PositionDomainError';

describe('Leverage', () => {
  it.each([1, 7, 15, 2.5])('akzeptiert Hebel %s', (rawLeverage) => {
    expect(Leverage.create(rawLeverage).value).toBe(rawLeverage);
  });

  it.each([0, -1, Number.NaN])('lehnt ungültigen Hebel %s ab', (rawLeverage) => {
    expect(() => Leverage.create(rawLeverage)).toThrow(InvalidLeverageError);
  });

  it('lehnt Strings ab', () => {
    expect(() => Leverage.create('7')).toThrow(InvalidLeverageError);
  });

  it('wirft einen Fehler, der von PositionDomainError erbt', () => {
    let caught: unknown;
    try {
      Leverage.create(0);
    } catch (error) {
      caught = error;
    }

    expect(caught).toBeInstanceOf(InvalidLeverageError);
    expect(caught).toBeInstanceOf(PositionDomainError);
  });
});
