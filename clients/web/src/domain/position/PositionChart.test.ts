import { describe, expect, it } from 'vitest';
import {
  CHART_INTERVALS,
  DEFAULT_CHART_INTERVAL,
  DEFAULT_CHART_LIMIT,
} from '@/domain/position/PositionChart';

describe('PositionChart (Domain)', () => {
  it('kennt genau die Intervalle des API-Vertrags in aufsteigender Länge', () => {
    expect(CHART_INTERVALS).toEqual(['5m', '15m', '1h', '4h', '1d']);
  });

  it('nutzt 15m als Standard-Intervall', () => {
    expect(DEFAULT_CHART_INTERVAL).toBe('15m');
  });

  it('lädt standardmäßig 300 Kerzen wie der API-Vertrag', () => {
    expect(DEFAULT_CHART_LIMIT).toBe(300);
  });
});
