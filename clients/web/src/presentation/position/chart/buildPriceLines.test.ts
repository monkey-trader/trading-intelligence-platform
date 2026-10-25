import { describe, expect, it } from 'vitest';
import type { PositionChart, RLevel } from '@/domain/position/PositionChart';
import { xagLongChart, xagChartWithoutProtection } from '@/test/fixtures/positionChart';
import { buildPriceLines } from '@/presentation/position/chart/buildPriceLines';

// Entry 55.75; TP 63.36 (xagLongChart)
const twoR: RLevel = { kind: 'TWO_R', rMultiple: '2', price: '59.00', percent: '0.0583' };
const threeR: RLevel = { kind: 'THREE_R', rMultiple: '3', price: '61.00', percent: '0.0942' };
const merged: RLevel = { ...threeR, price: '63.36', coincidesWithTakeProfit: true };
const separate: RLevel = { ...threeR, coincidesWithTakeProfit: false };

function withLevels(chart: PositionChart, value?: readonly RLevel[]): PositionChart {
  return { ...chart, levels: value };
}
const rLines = (chart: PositionChart) =>
  buildPriceLines(chart).filter((line) => ['r1', 'r2', 'r3'].includes(line.kind));

describe('buildPriceLines R-Linien', () => {
  it('zeichnet BE (2R) und TP1 (3R), keine 1R-Linie', () => {
    expect(rLines(withLevels(xagLongChart, [twoR, separate]))).toEqual([
      { kind: 'r2', price: 59, title: 'BE' },
      { kind: 'r3', price: 61, title: 'TP1' },
    ]);
    expect(
      buildPriceLines(withLevels(xagLongChart, [twoR, separate])).map((l) => l.kind),
    ).not.toContain('r1');
  });

  it('coincidesWithTakeProfit=true: nur der zusammenfallende TP entfällt, ein weiter entfernter bleibt', () => {
    const farTakeProfit = { orderId: 'far', triggerPrice: '80.00', quantity: '1', distanceRatio: '0.4' };
    const chart = {
      ...withLevels(xagLongChart, [twoR, merged]),
      takeProfit: [farTakeProfit, ...xagLongChart.takeProfit],
    };
    const targets = buildPriceLines(chart).filter((line) => line.kind === 'target');
    expect(targets.map((line) => line.price).sort()).toEqual([63.36, 80]);
    expect(targets.some((line) => line.title === 'TP = 3R' && line.price === 63.36)).toBe(true);
  });

  it('coincidesWithTakeProfit=false: TP-Linie der Order bleibt eigene Linie', () => {
    const lines = buildPriceLines(withLevels(xagLongChart, [twoR, separate]));
    expect(lines.filter((l) => l.kind === 'target')).toHaveLength(1);
    expect(lines.some((l) => l.title === 'TP = 3R')).toBe(false);
  });

  it('coincidesWithTakeProfit fehlt: getrennte Linien', () => {
    const lines = buildPriceLines(withLevels(xagLongChart, [twoR, threeR]));
    expect(lines.filter((l) => l.kind === 'target')).toHaveLength(1);
    expect(lines.find((l) => l.kind === 'r3')?.title).toBe('TP1');
  });

  it('coincidesWithTakeProfit=true: eine gemeinsame Linie "TP = 3R" in TP-Farbe, TP-Order-Linie entfällt', () => {
    const lines = buildPriceLines(withLevels(xagLongChart, [twoR, merged]));
    expect(lines.filter((l) => l.kind === 'target' || l.kind === 'r3')).toEqual([
      { kind: 'target', price: 63.36, title: 'TP = 3R' },
    ]);
  });

  it('Merge folgt allein dem Flag des Backends, nicht dem Abstand', () => {
    const farButFlagged: RLevel = { ...threeR, coincidesWithTakeProfit: true };
    const lines = buildPriceLines(withLevels(xagLongChart, [farButFlagged]));
    expect(lines.some((l) => l.title === 'TP = 3R')).toBe(true);
    const nearButUnflagged: RLevel = { ...merged, coincidesWithTakeProfit: false };
    expect(
      buildPriceLines(withLevels(xagLongChart, [nearButUnflagged])).some(
        (l) => l.title === 'TP = 3R',
      ),
    ).toBe(false);
  });

  it('kein TP gesetzt: 3R heißt "TP1"', () => {
    const lines = buildPriceLines(withLevels(xagChartWithoutProtection, [twoR, separate]));
    expect(lines.find((l) => l.kind === 'r3')?.title).toBe('TP1');
  });

  it('ohne levels (undefined oder leer) keine R-Linien', () => {
    expect(rLines(withLevels(xagLongChart, undefined))).toEqual([]);
    expect(rLines(withLevels(xagLongChart, []))).toEqual([]);
  });
});
