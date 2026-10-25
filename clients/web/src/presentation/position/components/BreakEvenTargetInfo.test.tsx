import { render } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { BreakEvenTargetInfo } from '@/presentation/position/components/BreakEvenTargetInfo';

function renderedText(): string {
  const { container } = render(<BreakEvenTargetInfo />);
  return container.textContent ?? '';
}

describe('BreakEvenTargetInfo', () => {
  it('erklärt BE ab 2R als SL auf Break-even', () => {
    const text = renderedText();
    expect(text).toMatch(/BE/);
    expect(text).toMatch(/2R/);
    expect(text).toMatch(/Break-even/);
  });

  it('erklärt TP1 bei 3R mit 33 % nur als Richtwert', () => {
    const text = renderedText();
    expect(text).toMatch(/TP1/);
    expect(text).toMatch(/3R/);
    expect(text).toMatch(/33 %/);
    expect(text).toMatch(/Richtwert/);
  });

  it.each([['1R'], ['TP2'], ['TP3']])('erwähnt %s nicht mehr', (forbidden) => {
    expect(renderedText()).not.toContain(forbidden);
  });
});
