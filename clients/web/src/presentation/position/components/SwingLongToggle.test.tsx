import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { SwingLongGateway } from '@/domain/position/SwingLongGateway';
import { SwingLongToggle } from '@/presentation/position/components/SwingLongToggle';
import { suiLongPosition, swingLongPosition } from '@/test/fixtures/openPositions';

function makeGateway(overrides: Partial<SwingLongGateway> = {}): SwingLongGateway {
  return {
    setSwingLong: vi.fn().mockResolvedValue({ kind: 'ok' }),
    clearSwingLong: vi.fn().mockResolvedValue({ kind: 'ok' }),
    ...overrides,
  };
}

describe('SwingLongToggle', () => {
  it('setzt das Flag einer Trade-Position und meldet die Änderung', async () => {
    const gateway = makeGateway();
    const onChanged = vi.fn();
    render(<SwingLongToggle position={suiLongPosition} gateway={gateway} onChanged={onChanged} />);

    const toggle = screen.getByRole('switch', { name: 'Swing Long für SUI-USDT' });
    expect(toggle).toHaveAttribute('aria-checked', 'false');
    await userEvent.click(toggle);

    expect(gateway.setSwingLong).toHaveBeenCalledWith('9100000000000000001');
    expect(gateway.clearSwingLong).not.toHaveBeenCalled();
    await waitFor(() => expect(onChanged).toHaveBeenCalledTimes(1));
  });

  it('entfernt das Flag einer Swing-Long-Position', async () => {
    const gateway = makeGateway();
    render(<SwingLongToggle position={swingLongPosition} gateway={gateway} />);

    const toggle = screen.getByRole('switch', { name: 'Swing Long für ETH-USDT' });
    expect(toggle).toHaveAttribute('aria-checked', 'true');
    await userEvent.click(toggle);

    expect(gateway.clearSwingLong).toHaveBeenCalledWith('9100000000000000004');
    expect(gateway.setSwingLong).not.toHaveBeenCalled();
  });

  it('zeigt bei einem Problem die deutsche Meldung und meldet keine Änderung', async () => {
    const gateway = makeGateway({
      setSwingLong: vi
        .fn()
        .mockResolvedValue({ kind: 'problem', problemType: 'backend-unreachable' }),
    });
    const onChanged = vi.fn();
    render(<SwingLongToggle position={suiLongPosition} gateway={gateway} onChanged={onChanged} />);

    await userEvent.click(screen.getByRole('switch', { name: 'Swing Long für SUI-USDT' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(/Backend nicht erreichbar/);
    expect(onChanged).not.toHaveBeenCalled();
  });
});
