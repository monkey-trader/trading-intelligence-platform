import { describe, expect, it } from 'vitest';
import type { PositionChart, RLevel } from '@/domain/position/PositionChart';
import { xagLongChart, xagChartWithoutProtection } from '@/test/fixtures/positionChart';
import { buildPriceLines } from '@/presentation/position/chart/buildPriceLines';

// Entry 55.75; TP 63.36 (xagLongChart)
const levels: readonly RLevel[] = [
  { kind: 'ONE_R', rMultiple: '1', price: '57.00', percent: '0.0224' },
  { kind: 'TWO_R', rMultiple: '2', price: '59.00', percent: '0.0583' },
  { kind: 'THREE_R', rMultiple: '3', price: '61.00', percent: '0.0942' },
];

function withLevels(chart: PositionChart, value?: readonly RLevel[]): PositionChart {
  return { ...chart, levels: value };
}
const rLines = (chart: PositionChart) =>
  buildPriceLines(chart).filter((line) => ['r1', 'r2', 'r3'].includes(line.kind));

describe('buildPriceLines R-Linien', () => {
  it('zeichnet 1R, 2R und 3R mit Kind, Preis und Titel', () => {
    expect(rLines(withLevels(xagLongChart, levels))).toEqual([
      { kind: 'r1', price: 57, title: '1R' },
      { kind: 'r2', price: 59, title: 'BE' },
      { kind: 'r3', price: 61, title: '3R' },
    ]);
  });

  it('3R ohne TP-Nähe heißt "3R" und der echte TP bleibt eine eigene Linie', () => {
    const lines = buildPriceLines(withLevels(xagLongChart, levels));
    expect(lines.find((l) => l.kind === 'r3')?.title).toBe('3R');
    expect(lines.filter((l) => l.kind === 'target')).toHaveLength(1);
  });

  it('kein TP gesetzt: Titel "3R TP3"', () => {
    const lines = buildPriceLines(withLevels(xagChartWithoutProtection, levels));
    expect(lines.find((l) => l.kind === 'r3')?.title).toBe('3R TP3');
  });

  it('3R genau 0,1 % des Entry vom TP entfernt: eine Linie "TP = 3R" in TP-Farbe (kind target)', () => {
    const near: RLevel[] = [...levels.slice(0, 2), { ...levels[2]!, price: '63.36' }];
    const merged = buildPriceLines(withLevels(xagLongChart, near));
    expect(merged.filter((l) => l.kind === 'target' || l.kind === 'r3')).toEqual([
      { kind: 'target', price: 63.36, title: 'TP = 3R' },
    ]);
  });

  it('Grenze: Abstand = 0,1 % des Entry (0.05575) verschmilzt, +0.01 nicht', () => {
    const atLimit = withLevels(xagLongChart, [{ ...levels[2]!, price: '63.41575' }]);
    expect(buildPriceLines(atLimit).some((l) => l.title === 'TP = 3R')).toBe(true);
    const beyond = withLevels(xagLongChart, [{ ...levels[2]!, price: '63.42575' }]);
    expect(buildPriceLines(beyond).some((l) => l.title === 'TP = 3R')).toBe(false);
  });

  it('ohne levels (undefined oder leer) keine R-Linien', () => {
    expect(rLines(withLevels(xagLongChart, undefined))).toEqual([]);
    expect(rLines(withLevels(xagLongChart, []))).toEqual([]);
  });
});
