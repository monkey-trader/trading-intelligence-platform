import { PositionDomainError } from '@/domain/position/errors/PositionDomainError';

/** Die Schließzeit liegt vor der Eröffnungszeit. */
export class CloseBeforeOpenError extends PositionDomainError {}
