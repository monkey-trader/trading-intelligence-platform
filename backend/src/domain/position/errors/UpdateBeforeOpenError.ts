import { PositionDomainError } from '@/domain/position/errors/PositionDomainError';

/** Die letzte Änderung liegt vor der Eröffnungszeit. */
export class UpdateBeforeOpenError extends PositionDomainError {}
