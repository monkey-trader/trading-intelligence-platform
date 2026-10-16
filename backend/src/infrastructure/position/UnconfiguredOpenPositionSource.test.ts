import { describe, expect, it } from 'vitest';
import { UnconfiguredOpenPositionSource } from '@/infrastructure/position/UnconfiguredOpenPositionSource';
import { OpenPositionsUnavailableError } from '@/domain/position/errors/OpenPositionsUnavailableError';

async function captureRejection(action: () => Promise<unknown>): Promise<unknown> {
  try {
    await action();
  } catch (error) {
    return error;
  }
  throw new Error('Es wurde ein Fehler erwartet, aber keiner geworfen.');
}

describe('UnconfiguredOpenPositionSource', () => {
  it('wirft OpenPositionsUnavailableError mit reason not-configured', async () => {
    const source = new UnconfiguredOpenPositionSource();

    const caught = await captureRejection(() => source.fetchOpenPositions());

    expect(caught).toBeInstanceOf(OpenPositionsUnavailableError);
    expect(caught instanceof OpenPositionsUnavailableError ? caught.reason : undefined).toBe(
      'not-configured',
    );
  });

  it('wirft bei jedem Aufruf erneut', async () => {
    const source = new UnconfiguredOpenPositionSource();

    await expect(source.fetchOpenPositions()).rejects.toBeInstanceOf(OpenPositionsUnavailableError);
    await expect(source.fetchOpenPositions()).rejects.toBeInstanceOf(OpenPositionsUnavailableError);
  });
});
