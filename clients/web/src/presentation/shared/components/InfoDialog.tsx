import { useEffect, useId, useRef, type ReactNode } from 'react';
import styles from '@/presentation/shared/components/InfoDialog.module.css';

type InfoDialogProps = Readonly<{
  title: string;
  isOpen: boolean;
  onClose: () => void;
  children: ReactNode;
}>;

/** Modaler Hinweisdialog (natives <dialog>): Esc, Klick auf den Hintergrund und "Schließen" schließen ihn. */
export function InfoDialog({ title, isOpen, onClose, children }: InfoDialogProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const titleId = useId();

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;
    if (isOpen && !dialog.open) {
      if (typeof dialog.showModal === 'function') dialog.showModal();
      else dialog.setAttribute('open', '');
    }
    if (!isOpen && dialog.open) {
      if (typeof dialog.close === 'function') dialog.close();
      else dialog.removeAttribute('open');
    }
  }, [isOpen]);

  return (
    <dialog
      ref={dialogRef}
      className={styles.dialog}
      aria-labelledby={titleId}
      onClose={onClose}
      onClick={(event) => {
        // Nur der Hintergrund (::backdrop) trifft das <dialog> selbst, Klicks im Inhalt nicht.
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div className={styles.content}>
        <h2 id={titleId} className={styles.title}>
          {title}
        </h2>
        <div className={styles.body}>{children}</div>
        <button type="button" className={styles.closeButton} onClick={onClose}>
          Schließen
        </button>
      </div>
    </dialog>
  );
}
