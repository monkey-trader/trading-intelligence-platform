import { DecimalError } from '@/domain/shared/decimal/DecimalError';

/** Der Divisor einer Dezimal-Division ist numerisch 0. */
export class DivisionByZeroError extends DecimalError {
  constructor(message: string = 'Division durch 0.') {
    super(message);
  }
}
