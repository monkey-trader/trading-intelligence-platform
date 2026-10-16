import { describe, expect, it } from 'vitest';
import { PositionSide } from '@/domain/position/valueObjects/PositionSide';
import { InvalidPositionSideError } from '@/domain/position/errors/InvalidPositionSideError';
import { PositionDomainError } from '@/domain/position/errors/PositionDomainError';

describe('PositionSide', () => {
  it.each(['LONG', 'SHORT'] as const)('akzeptiert "%s"', (rawSide) => {
    expect(PositionSide.create(rawSide).value).toBe(rawSide);
  });

  it.each(['long', 'short', 'BOTH', '', 'BUY'])('lehnt ungültige Seite "%s" ab', (rawSide) => {
    expect(() => PositionSide.create(rawSide)).toThrow(InvalidPositionSideError);
  });

  it('lehnt Nicht-Strings ab', () => {
    expect(() => PositionSide.create(1)).toThrow(InvalidPositionSideError);
  });

  it('wirft einen Fehler, der von PositionDomainError erbt', () => {
    let caught: unknown;
    try {
      PositionSide.create('BOTH');
    } catch (error) {
      caught = error;
    }

    expect(caught).toBeInstanceOf(InvalidPositionSideError);
    expect(caught).toBeInstanceOf(PositionDomainError);
  });
});
