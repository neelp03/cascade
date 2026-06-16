import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@cascade/types': path.resolve(__dirname, '../../packages/types/src/index.ts'),
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api/ingest': {
        target: 'http://localhost:8080',
        rewrite: (p) => p.replace(/^\/api\/ingest/, ''),
      },
      '/api/query': {
        target: 'http://localhost:8081',
        rewrite: (p) => p.replace(/^\/api\/query/, ''),
      },
    },
  },
});
