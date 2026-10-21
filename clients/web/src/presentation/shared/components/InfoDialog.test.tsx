import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { InfoDialog } from '@/presentation/shared/components/InfoDialog';

describe('InfoDialog', () => {
  it('zeigt Titel und Inhalt, wenn er offen ist', () => {
    render(
      <InfoDialog title="BE und Ziel" isOpen onClose={vi.fn()}>
        <p>Erklärung</p>
      </InfoDialog>,
    );

    expect(screen.getByRole('dialog', { name: 'BE und Ziel' })).toBeInTheDocument();
    expect(screen.getByText('Erklärung')).toBeInTheDocument();
  });

  it('zeigt nichts, solange er geschlossen ist', () => {
    render(
      <InfoDialog title="BE und Ziel" isOpen={false} onClose={vi.fn()}>
        <p>Erklärung</p>
      </InfoDialog>,
    );

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('ruft onClose über den Schließen-Knopf auf', async () => {
    const onClose = vi.fn();
    render(
      <InfoDialog title="BE und Ziel" isOpen onClose={onClose}>
        <p>Erklärung</p>
      </InfoDialog>,
    );

    await userEvent.click(screen.getByRole('button', { name: 'Schließen' }));

    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
