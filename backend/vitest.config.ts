import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      // Sprachneutrale BingX-Fixtures im Repo-Root, gemeinsam für TS- und Go-Backend.
      '@fixtures': fileURLToPath(new URL('../fixtures', import.meta.url)),
    },
  },
  test: {
    environment: 'node',
    include: ['src/**/*.test.ts'],
  },
});
