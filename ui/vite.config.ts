import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      // No changeOrigin: the proxied request keeps Host: localhost:3000, which
      // matches the browser's Origin, so `ailang server`'s same-origin policy
      // admits it (M-SERVER-ORIGIN-POLICY). changeOrigin: true rewrote Host to
      // :1957 and made every POST look cross-origin (403).
      '/api': {
        target: 'http://localhost:1957',
      },
      '/ws': {
        target: 'ws://localhost:1957',
        ws: true,
      },
    },
  },
});
