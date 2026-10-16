import { DecimalError } from '@/domain/shared/decimal/DecimalError';

/** Eingabe ist keine Dezimalzahl im Format `^-?\d+(\.\d+)?$` oder die Skala ist ungültig. */
export class InvalidDecimalError extends DecimalError {}
