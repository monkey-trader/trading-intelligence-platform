import { PositionDomainError } from '@/domain/position/errors/PositionDomainError';

/** Das Symbol hat nicht die Form BASIS-QUOTE (z.B. BTC-USDT). */
export class InvalidTradingSymbolError extends PositionDomainError {}
