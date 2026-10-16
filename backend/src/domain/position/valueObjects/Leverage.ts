import { InvalidLeverageError } from '@/domain/position/errors/InvalidLeverageError';

/** Hebel einer Position, eine endliche Zahl größer als 0. */
export class Leverage {
  private constructor(readonly value: number) {}

  static create(rawLeverage: unknown): Leverage {
    if (typeof rawLeverage !== 'number' || !Number.isFinite(rawLeverage) || rawLeverage <= 0) {
      throw new InvalidLeverageError(`Ungültiger Hebel: ${String(rawLeverage)}`);
    }
    return new Leverage(rawLeverage);
  }
}
