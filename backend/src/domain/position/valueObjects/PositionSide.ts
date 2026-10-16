import { InvalidPositionSideError } from '@/domain/position/errors/InvalidPositionSideError';

export type PositionSideValue = 'LONG' | 'SHORT';

/** Richtung einer Position. */
export class PositionSide {
  private constructor(readonly value: PositionSideValue) {}

  static create(rawSide: unknown): PositionSide {
    if (rawSide !== 'LONG' && rawSide !== 'SHORT') {
      throw new InvalidPositionSideError(
        `Ungültige Seite: ${typeof rawSide === 'string' ? `"${rawSide}"` : `Typ ${typeof rawSide}`}`,
      );
    }
    return new PositionSide(rawSide);
  }
}
