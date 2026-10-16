import { DatabaseSync } from 'node:sqlite';
import { SCHEMA_SQL } from '@/infrastructure/persistence/sqlite/schema';

/** Öffnet die SQLite-Datenbank (Datei oder `:memory:`) und legt das Schema an. */
export function openDatabase(databasePath: string): DatabaseSync {
  const database = new DatabaseSync(databasePath);
  database.exec('PRAGMA foreign_keys = ON;');
  database.exec(SCHEMA_SQL);
  return database;
}
