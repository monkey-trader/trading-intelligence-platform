import { PositionDomainError } from '@/domain/position/errors/PositionDomainError';

/** Der Hebel ist keine positive Zahl. */
export class InvalidLeverageError extends PositionDomainError {}
