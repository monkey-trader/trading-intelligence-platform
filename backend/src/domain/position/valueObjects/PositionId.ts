import { InvalidPositionIdError } from '@/domain/position/errors/InvalidPositionIdError';

const DIGITS_ONLY = /^\d+$/;

/** Eindeutige BingX-Positions-ID. Immer ein String, weil 19-stellige IDs als number Präzision verlieren. */
export class PositionId {
  private constructor(readonly value: string) {}

  static create(rawId: unknown): PositionId {
    if (typeof rawId !== 'string' || !DIGITS_ONLY.test(rawId)) {
      throw new InvalidPositionIdError(`Ungültige Positions-ID: ${describe(rawId)}`);
    }
    return new PositionId(rawId);
  }
}

function describe(rawValue: unknown): string {
  return typeof rawValue === 'string' ? `"${rawValue}"` : `Typ ${typeof rawValue}`;
}
