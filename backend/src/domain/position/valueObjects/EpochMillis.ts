import { InvalidTimestampError } from '@/domain/position/errors/InvalidTimestampError';

/** Zeitpunkt in Millisekunden seit 1970, ganzzahlig und größer als 0. */
export class EpochMillis {
  private constructor(readonly value: number) {}

  static create(rawTimestamp: unknown): EpochMillis {
    if (typeof rawTimestamp !== 'number' || !Number.isInteger(rawTimestamp) || rawTimestamp <= 0) {
      throw new InvalidTimestampError(`Ungültiger Zeitstempel: ${String(rawTimestamp)}`);
    }
    return new EpochMillis(rawTimestamp);
  }
}
