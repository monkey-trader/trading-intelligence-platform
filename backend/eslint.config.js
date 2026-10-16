import eslint from '@eslint/js';
import tseslint from 'typescript-eslint';
import { importX } from 'eslint-plugin-import-x';
import { createTypeScriptImportResolver } from 'eslint-import-resolver-typescript';
import prettierConfig from 'eslint-config-prettier';
import globals from 'globals';

export default tseslint.config(
  { ignores: ['dist/', 'coverage/', 'node_modules/'] },
  eslint.configs.recommended,
  tseslint.configs.recommended,
  {
    files: ['**/*.ts', '**/*.js'],
    languageOptions: { globals: globals.node },
    plugins: { 'import-x': importX },
    settings: {
      'import-x/resolver-next': [createTypeScriptImportResolver({ project: './tsconfig.json' })],
    },
    rules: {
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/consistent-type-imports': 'error',
      // Onion-Architektur: Abhängigkeiten zeigen nur nach innen zum Domain-Kern.
      // Der Resolver löst @/-Alias und relative Pfade gleich auf.
      'import-x/no-restricted-paths': [
        'error',
        {
          zones: [
            {
              target: './src/domain',
              from: ['./src/application', './src/infrastructure', './src/presentation'],
              message: 'Domain darf nur von sich selbst abhängen.',
            },
            {
              target: './src/application',
              from: ['./src/infrastructure', './src/presentation'],
              message: 'Application kennt nur Domain und Ports, keine Adapter.',
            },
            {
              target: './src/presentation',
              from: './src/infrastructure',
              message: 'Presentation spricht über Use Cases, nicht direkt mit Infrastructure.',
            },
          ],
        },
      ],
    },
  },
  {
    // Der Domain-Kern bleibt frei von Node-APIs, HTTP-Framework und Datenbank.
    files: ['src/domain/**/*.ts'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            { group: ['node:*'], message: 'Keine Node-APIs im Domain-Kern.' },
            { group: ['fastify', 'fastify/*', '@fastify/*'], message: 'Kein HTTP im Domain-Kern.' },
            {
              group: ['*sqlite*', 'better-sqlite3', 'kysely', 'drizzle-orm', 'drizzle-orm/*'],
              message: 'Keine Datenbank im Domain-Kern.',
            },
          ],
        },
      ],
    },
  },
  prettierConfig,
);
