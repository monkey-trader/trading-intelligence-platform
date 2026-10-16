import { PositionDomainError } from '@/domain/position/errors/PositionDomainError';

/** Die Positions-ID ist kein String aus Ziffern. */
export class InvalidPositionIdError extends PositionDomainError {}
