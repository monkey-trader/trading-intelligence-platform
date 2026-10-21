import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { AppHeader } from '@/presentation/shared/components/AppHeader';

describe('AppHeader', () => {
  it('rendert ein Banner mit dem App-Titel', () => {
    render(<AppHeader />);

    const banner = screen.getByRole('banner');
    expect(banner).toHaveTextContent('Trading Intelligence Platform');
  });

  it('bietet die Design-Wahl System, Hell und Dunkel an', () => {
    render(<AppHeader />);

    const select = screen.getByRole('combobox', { name: 'Design' });
    expect(select).toHaveValue('system');
    expect(screen.getAllByRole('option').map((option) => option.textContent)).toEqual([
      'System',
      'Hell',
      'Dunkel',
    ]);
  });
});
