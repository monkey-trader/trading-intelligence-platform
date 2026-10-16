/** Basis aller Fehler der Dezimal-Arithmetik im Shared-Kern. */
export class DecimalError extends Error {
  constructor(message: string) {
    super(message);
    this.name = new.target.name;
  }
}
