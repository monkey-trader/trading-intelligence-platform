import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { AppHeader } from '@/presentation/shared/components/AppHeader';

describe('AppHeader', () => {
  it('rendert ein Banner mit dem App-Titel', () => {
    render(<AppHeader />);

    const banner = screen.getByRole('banner');
    expect(banner).toHaveTextContent('Trading Intelligence Platform');
  });
});
