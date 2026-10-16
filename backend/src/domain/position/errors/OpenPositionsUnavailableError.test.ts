import { describe, expect, it } from 'vitest';
import { OpenPositionsUnavailableError } from '@/domain/position/errors/OpenPositionsUnavailableError';
import type { OpenPositionsUnavailableReason } from '@/domain/position/errors/OpenPositionsUnavailableError';

const ALL_REASONS: readonly OpenPositionsUnavailableReason[] = [
  'not-configured',
  'rate-limited',
  'unreachable',
  'upstream-error',
  'invalid-response',
];

describe('OpenPositionsUnavailableError', () => {
  it.each(ALL_REASONS)('behält reason "%s" bei und ist ein Error', (reason) => {
    const error = new OpenPositionsUnavailableError(reason);

    expect(error).toBeInstanceOf(OpenPositionsUnavailableError);
    expect(error).toBeInstanceOf(Error);
    expect(error.reason).toBe(reason);
  });

  it('behält die Ursache als cause bei', () => {
    const originalError = new Error('Ursprünglicher Adapterfehler');

    const error = new OpenPositionsUnavailableError('upstream-error', originalError);

    expect(error.reason).toBe('upstream-error');
    expect(error.cause).toBe(originalError);
  });

  it('hat ohne Ursache kein cause', () => {
    const error = new OpenPositionsUnavailableError('not-configured');

    expect(error.cause).toBeUndefined();
  });

  it('setzt name auf den Klassennamen', () => {
    expect(new OpenPositionsUnavailableError('unreachable').name).toBe(
      'OpenPositionsUnavailableError',
    );
  });
});
