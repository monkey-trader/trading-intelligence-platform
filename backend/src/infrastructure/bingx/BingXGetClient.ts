/** Query-Parameter eines BingX-Requests. Reihenfolge bleibt erhalten. */
export type BingXQueryParams = Readonly<Record<string, string | number>>;

/** Minimale Sicht auf den BingX-Client, die die Sources brauchen. Liefert das Feld `data`. */
export type BingXGetClient = {
  get(path: string, params: BingXQueryParams): Promise<unknown>;
};
