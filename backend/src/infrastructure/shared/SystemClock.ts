import type { Clock } from '@/domain/shared/interfaces/Clock';

/** Echte Systemzeit in Epoch-Millisekunden. */
export const systemClock: Clock = { now: () => Date.now() };
