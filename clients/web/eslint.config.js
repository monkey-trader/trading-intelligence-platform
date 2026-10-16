import eslint from '@eslint/js';
import tseslint from 'typescript-eslint';
import reactHooks from 'eslint-plugin-react-hooks';
import { importX } from 'eslint-plugin-import-x';
import { createTypeScriptImportResolver } from 'eslint-import-resolver-typescript';
import prettierConfig from 'eslint-config-prettier';
import globals from 'globals';

export default tseslint.config(
  { ignores: ['dist/', 'coverage/', 'node_modules/', 'src/infrastructure/api/generated.ts'] },
  eslint.configs.recommended,
  tseslint.configs.recommended,
  {
    files: ['**/*.{ts,tsx,js}'],
    languageOptions: { globals: { ...globals.browser, ...globals.node } },
    plugins: { 'import-x': importX, 'react-hooks': reactHooks },
    settings: {
      'import-x/resolver-next': [createTypeScriptImportResolver({ project: './tsconfig.json' })],
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/consistent-type-imports': 'error',
      // Der Client spricht nur über die HTTP-API mit dem Backend, nie über dessen Code.
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['**/backend/**'],
              message: 'Nur über die HTTP-API, nie Backend-Code importieren.',
            },
          ],
        },
      ],
      // Onion-Architektur wie im Backend. Composition Root ist src/main.tsx.
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
              message: 'Application kennt nur Domain und Ports.',
            },
            {
              target: './src/presentation',
              from: './src/infrastructure',
              message: 'Presentation bekommt das Gateway injiziert (src/main.tsx).',
            },
          ],
        },
      ],
    },
  },
  {
    // Der Domain-Kern bleibt frei von React und Browser-I/O.
    files: ['src/domain/**/*.ts'],
    rules: {
      'no-restricted-globals': ['error', { name: 'fetch', message: 'Kein I/O im Domain-Kern.' }],
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            { group: ['react', 'react-dom', 'react/*'], message: 'Kein React im Domain-Kern.' },
          ],
        },
      ],
    },
  },
  prettierConfig,
);
