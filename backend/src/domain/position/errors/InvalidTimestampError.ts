import { PositionDomainError } from '@/domain/position/errors/PositionDomainError';

/** Der Zeitstempel ist keine positive ganze Zahl in Millisekunden. */
export class InvalidTimestampError extends PositionDomainError {}
