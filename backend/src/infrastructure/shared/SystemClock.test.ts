import { describe, expect, it } from 'vitest';
import { systemClock } from '@/infrastructure/shared/SystemClock';
import type { Clock } from '@/domain/shared/interfaces/Clock';

describe('systemClock', () => {
  it('erfüllt den Clock-Port', () => {
    const clock: Clock = systemClock;

    expect(typeof clock.now).toBe('function');
  });

  it('liefert die aktuelle Zeit in Epoch-Millisekunden nahe Date.now', () => {
    const before = Date.now();
    const now = systemClock.now();
    const after = Date.now();

    expect(Number.isInteger(now)).toBe(true);
    expect(now).toBeGreaterThanOrEqual(before);
    expect(now).toBeLessThanOrEqual(after);
  });
});
