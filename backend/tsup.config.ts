import { defineConfig } from 'tsup';

// Bündelt src/main.ts und den Import-CLI nach dist/ und löst dabei den @/-Alias aus tsconfig.json auf.
export default defineConfig({
  entry: ['src/main.ts', 'src/cli/importPositions.ts'],
  outDir: 'dist',
  format: ['esm'],
  platform: 'node',
  target: 'node24',
  clean: true,
  sourcemap: true,
  // node:-Präfix behalten: `node:sqlite` gibt es nur mit Präfix, ohne würde `sqlite` als npm-Paket gesucht.
  removeNodeProtocol: false,
});
