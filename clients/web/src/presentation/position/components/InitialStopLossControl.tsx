import { useEffect, useRef, useState, type FormEvent, type KeyboardEvent } from 'react';
import type { InitialStopLossGateway } from '@/domain/position/InitialStopLossGateway';
import type { OpenPosition } from '@/domain/position/OpenPosition';
import { mapInitialStopLossProblem } from '@/presentation/position/initialStopLossErrorMapper';
import { validateInitialStopLoss } from '@/presentation/position/initialStopLossValidation';
import styles from '@/presentation/position/components/OpenPositionsTable.module.css';

type InitialStopLossControlProps = Readonly<{
  position: OpenPosition;
  gateway: InitialStopLossGateway;
  onChanged?: (() => void) | undefined;
}>;

/** Manuelles Eintragen (bei unbekanntem Risiko) und Zurücksetzen des ursprünglichen SL. */
export function InitialStopLossControl({
  position,
  gateway,
  onChanged,
}: InitialStopLossControlProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [rawValue, setRawValue] = useState('');
  const [errorMessage, setErrorMessage] = useState<string>();
  const [isBusy, setIsBusy] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const inputId = `initial-sl-${position.positionId}-${position.side}`;
  const hasKnownRisk = (position.levels?.length ?? 0) > 0;

  useEffect(() => {
    if (isEditing) inputRef.current?.focus();
  }, [isEditing]);

  const close = () => {
    setIsEditing(false);
    setRawValue('');
    setErrorMessage(undefined);
  };

  const save = async (event: FormEvent) => {
    event.preventDefault();
    const validationMessage = validateInitialStopLoss(
      position.side,
      position.avgOpenPrice,
      rawValue,
    );
    if (validationMessage) {
      setErrorMessage(validationMessage);
      return;
    }
    setIsBusy(true);
    const result = await gateway.setInitialStopLoss(position.positionId, rawValue.trim());
    setIsBusy(false);
    if (result.kind === 'problem') {
      setErrorMessage(mapInitialStopLossProblem(result.problemType));
      return;
    }
    close();
    onChanged?.();
  };

  const reset = async () => {
    setIsBusy(true);
    const result = await gateway.clearInitialStopLoss(position.positionId);
    setIsBusy(false);
    if (result.kind === 'problem') {
      setErrorMessage(mapInitialStopLossProblem(result.problemType));
      return;
    }
    setErrorMessage(undefined);
    onChanged?.();
  };

  const handleKeyDown = (event: KeyboardEvent) => {
    if (event.key === 'Escape') close();
  };

  const errorAlert = errorMessage && (
    <p role="alert" className={styles.slError}>
      {errorMessage}
    </p>
  );

  if (position.initialStopLossSource === 'MANUAL') {
    return (
      <div className={styles.slControl}>
        <div className={styles.manualRow}>
          <span className={styles.manualBadge}>manuell</span>
          <button type="button" className={styles.slButton} disabled={isBusy} onClick={reset}>
            Zurücksetzen
          </button>
        </div>
        {errorAlert}
      </div>
    );
  }

  if (hasKnownRisk) return null;

  return (
    <div className={styles.slControl}>
      {isEditing ? (
        <form className={styles.slForm} onSubmit={save} onKeyDown={handleKeyDown}>
          <label htmlFor={inputId} className={styles.slLabel}>
            Ursprünglicher SL
          </label>
          <input
            ref={inputRef}
            id={inputId}
            type="text"
            inputMode="decimal"
            className={styles.slInput}
            value={rawValue}
            onChange={(event) => setRawValue(event.target.value)}
          />
          <button
            type="submit"
            className={`${styles.slButton} ${styles.slButtonPrimary}`}
            disabled={isBusy}
          >
            Speichern
          </button>
          <button type="button" className={styles.slButton} onClick={close}>
            Abbrechen
          </button>
        </form>
      ) : (
        <button
          type="button"
          className={styles.slButton}
          aria-label={`Ursprünglichen SL eintragen für ${position.symbol}`}
          onClick={() => setIsEditing(true)}
        >
          SL eintragen
        </button>
      )}
      {errorAlert}
    </div>
  );
}
