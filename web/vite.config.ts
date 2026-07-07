import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

const apiTarget = process.env.PUBLIC_API_TARGET || process.env.VITE_API_TARGET || 'http://localhost:18080';

export default defineConfig({
  plugins: [react()],
  build: {
    chunkSizeWarningLimit: 1000,
    rollupOptions: {
      output: {
        manualChunks: {
          antd: ['antd'],
          i18n: ['i18next', 'react-i18next'],
        },
      },
    },
  },
  server: {
    port: 5180,
    host: true,
    strictPort: false,
    proxy: {
      '/api': {
        target: apiTarget,
        changeOrigin: true,
      },
      '/sapi': {
        target: apiTarget,
        changeOrigin: true,
      },
      '/auth/feishu': {
        target: apiTarget,
        changeOrigin: true,
      },
      '/oauth2': {
        target: apiTarget,
        changeOrigin: true,
      },
    },
  },
});
