import { useState } from 'react';
import type { OpenPosition } from '@/domain/position/OpenPosition';
import type { SwingLongGateway } from '@/domain/position/SwingLongGateway';
import { mapSwingLongProblem } from '@/presentation/position/swingLongErrorMapper';
import styles from '@/presentation/position/components/SwingLongToggle.module.css';

type SwingLongToggleProps = Readonly<{
  position: OpenPosition;
  gateway: SwingLongGateway;
  onChanged?: (() => void) | undefined;
}>;

/** Schalter pro Zeile: Position als Swing Long markieren bzw. Markierung entfernen. */
export function SwingLongToggle({ position, gateway, onChanged }: SwingLongToggleProps) {
  const [errorMessage, setErrorMessage] = useState<string>();
  const [isBusy, setIsBusy] = useState(false);

  const toggle = async () => {
    setIsBusy(true);
    setErrorMessage(undefined);
    try {
      const result = position.swingLong
        ? await gateway.clearSwingLong(position.positionId)
        : await gateway.setSwingLong(position.positionId);
      if (result.kind === 'problem') setErrorMessage(mapSwingLongProblem(result.problemType));
      else onChanged?.();
    } finally {
      setIsBusy(false);
    }
  };

  return (
    <span className={styles.wrapper}>
      <button
        type="button"
        role="switch"
        aria-checked={position.swingLong}
        aria-label={`Swing Long für ${position.symbol}`}
        disabled={isBusy}
        className={styles.toggle}
        onClick={() => void toggle()}
      >
        {position.swingLong && (
          <span aria-hidden="true" className={styles.mark}>
            ✓{' '}
          </span>
        )}
        Swing Long
      </button>
      {errorMessage && (
        <span role="alert" className={styles.error}>
          {errorMessage}
        </span>
      )}
    </span>
  );
}
