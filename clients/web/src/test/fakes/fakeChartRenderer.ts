import { vi } from 'vitest';
import type { ChartRenderer } from '@/presentation/position/chart/ChartRenderer';

/** Fake für den ChartRenderer-Port: keine Canvas nötig, alle Aufrufe sind Spies. */
export function createFakeChartRenderer() {
  const renderer = {
    setCandles: vi.fn<ChartRenderer['setCandles']>(),
    setPriceLines: vi.fn<ChartRenderer['setPriceLines']>(),
    setMarkers: vi.fn<ChartRenderer['setMarkers']>(),
    destroy: vi.fn<ChartRenderer['destroy']>(),
  } satisfies ChartRenderer;
  return renderer;
}

export type FakeChartRenderer = ReturnType<typeof createFakeChartRenderer>;
