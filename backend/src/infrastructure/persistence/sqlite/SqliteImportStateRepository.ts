import type { DatabaseSync } from 'node:sqlite';
import type { ImportStateRepository } from '@/domain/position/interfaces/ImportStateRepository';

/** Import-Fortschritt in der Tabelle `import_state`. */
export class SqliteImportStateRepository implements ImportStateRepository {
  constructor(private readonly database: DatabaseSync) {}

  async get(key: string): Promise<number | undefined> {
    const row = this.database.prepare('SELECT value FROM import_state WHERE key = ?').get(key);
    if (row === undefined) return undefined;
    return Number(row.value);
  }

  async set(key: string, epochMs: number): Promise<void> {
    this.database
      .prepare(
        'INSERT INTO import_state (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value',
      )
      .run(key, epochMs);
  }

  async delete(key: string): Promise<void> {
    this.database.prepare('DELETE FROM import_state WHERE key = ?').run(key);
  }

  /** Präfix-Vergleich über substr statt LIKE, damit `_` und `%` keine Platzhalter sind. */
  async listKeys(prefix: string): Promise<string[]> {
    const rows = this.database
      .prepare('SELECT key FROM import_state WHERE substr(key, 1, length(?1)) = ?1 ORDER BY key')
      .all(prefix);
    return rows.map((row) => String(row.key));
  }
}
