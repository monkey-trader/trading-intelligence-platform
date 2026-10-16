/** Basis aller fachlichen Fehler rund um Positionen. Erlaubt `instanceof PositionDomainError`. */
export class PositionDomainError extends Error {
  constructor(message: string) {
    super(message);
    this.name = new.target.name;
  }
}
