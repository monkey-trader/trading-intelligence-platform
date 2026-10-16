/** Port: merkt sich den Import-Fortschritt ("importiert bis") pro Schlüssel als Epoch-Millisekunden. */
export type ImportStateRepository = {
  get(key: string): Promise<number | undefined>;
  set(key: string, epochMs: number): Promise<void>;
  /** Entfernt einen Schlüssel, ein unbekannter Schlüssel ist kein Fehler. */
  delete(key: string): Promise<void>;
  /** Alle Schlüssel, die exakt mit `prefix` beginnen. */
  listKeys(prefix: string): Promise<string[]>;
};
