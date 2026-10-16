import { describe, expect, it } from 'vitest';
import { EpochMillis } from '@/domain/position/valueObjects/EpochMillis';
import { InvalidTimestampError } from '@/domain/position/errors/InvalidTimestampError';
import { PositionDomainError } from '@/domain/position/errors/PositionDomainError';

describe('EpochMillis', () => {
  it('übernimmt einen gültigen Zeitstempel in Millisekunden', () => {
    expect(EpochMillis.create(1780000000000).value).toBe(1780000000000);
  });

  it.each([0, -1, 1.5, Number.NaN, Number.POSITIVE_INFINITY])(
    'lehnt ungültigen Zeitstempel %s ab',
    (rawTimestamp) => {
      expect(() => EpochMillis.create(rawTimestamp)).toThrow(InvalidTimestampError);
    },
  );

  it('lehnt Strings ab', () => {
    expect(() => EpochMillis.create('1780000000000')).toThrow(InvalidTimestampError);
  });

  it('wirft einen Fehler, der von PositionDomainError erbt', () => {
    let caught: unknown;
    try {
      EpochMillis.create(0);
    } catch (error) {
      caught = error;
    }

    expect(caught).toBeInstanceOf(InvalidTimestampError);
    expect(caught).toBeInstanceOf(PositionDomainError);
  });
});
