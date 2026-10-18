import { fileURLToPath } from 'node:url';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

// BACKEND_URL stellt den Dev-Proxy auf ein anderes Backend um (z.B. Go auf Port 3100).
const BACKEND_URL = process.env.BACKEND_URL ?? 'http://127.0.0.1:3000';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    // Nur lokal: das Backend liefert Kontodaten ohne Authentifizierung.
    host: '127.0.0.1',
    port: 5173,
    strictPort: true,
    proxy: {
      // changeOrigin setzt den Host-Header auf 127.0.0.1:3000, sonst lehnt der Host-Check ab.
      '/api': {
        target: BACKEND_URL,
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
});
