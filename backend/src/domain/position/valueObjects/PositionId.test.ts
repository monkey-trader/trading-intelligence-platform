import { describe, expect, it } from 'vitest';
import { PositionId } from '@/domain/position/valueObjects/PositionId';
import { InvalidPositionIdError } from '@/domain/position/errors/InvalidPositionIdError';
import { PositionDomainError } from '@/domain/position/errors/PositionDomainError';

describe('PositionId', () => {
  it('übernimmt eine 19-stellige ID exakt als String', () => {
    const positionId = PositionId.create('9000000000000000001');

    expect(positionId.value).toBe('9000000000000000001');
  });

  it('lehnt eine number ab, weil große IDs dabei Präzision verlieren', () => {
    const attempt = (): unknown => PositionId.create(Number.MAX_SAFE_INTEGER + 2);

    expect(attempt).toThrow(InvalidPositionIdError);
  });

  it.each(['', 'abc', '12a', '-1', '1.5', ' 123'])('lehnt ungültige ID "%s" ab', (rawId) => {
    expect(() => PositionId.create(rawId)).toThrow(InvalidPositionIdError);
  });

  it.each([undefined, null, {}, true])('lehnt Nicht-Strings ab (%s)', (rawId) => {
    expect(() => PositionId.create(rawId)).toThrow(InvalidPositionIdError);
  });

  it('wirft einen Fehler, der von PositionDomainError und Error erbt', () => {
    let caught: unknown;
    try {
      PositionId.create('kaputt');
    } catch (error) {
      caught = error;
    }

    expect(caught).toBeInstanceOf(InvalidPositionIdError);
    expect(caught).toBeInstanceOf(PositionDomainError);
    expect(caught).toBeInstanceOf(Error);
  });
});
