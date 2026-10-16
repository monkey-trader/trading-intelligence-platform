import { PositionDomainError } from '@/domain/position/errors/PositionDomainError';

/** Die Seite ist weder LONG noch SHORT. */
export class InvalidPositionSideError extends PositionDomainError {}
