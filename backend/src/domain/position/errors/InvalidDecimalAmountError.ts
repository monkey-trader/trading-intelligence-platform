import { PositionDomainError } from '@/domain/position/errors/PositionDomainError';

/** Der Betrag ist kein Dezimal-String. */
export class InvalidDecimalAmountError extends PositionDomainError {}
