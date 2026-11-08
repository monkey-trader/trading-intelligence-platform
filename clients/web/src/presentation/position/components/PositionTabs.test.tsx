import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { PositionTabs } from '@/presentation/position/components/PositionTabs';

describe('PositionTabs', () => {
  it('zeigt die Reiter Trades und Swing Long', () => {
    render(<PositionTabs active="trades" onChange={vi.fn()} />);

    expect(screen.getByRole('tab', { name: 'Trades' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'Swing Long' })).toBeInTheDocument();
  });

  it('markiert nur den aktiven Reiter als ausgewählt', () => {
    render(<PositionTabs active="swing-long" onChange={vi.fn()} />);

    expect(screen.getByRole('tab', { name: 'Swing Long' })).toHaveAttribute(
      'aria-selected',
      'true',
    );
    expect(screen.getByRole('tab', { name: 'Trades' })).toHaveAttribute('aria-selected', 'false');
  });

  it('meldet den gewählten Reiter', async () => {
    const onChange = vi.fn();
    render(<PositionTabs active="trades" onChange={onChange} />);

    await userEvent.click(screen.getByRole('tab', { name: 'Swing Long' }));

    expect(onChange).toHaveBeenCalledWith('swing-long');
  });
});
