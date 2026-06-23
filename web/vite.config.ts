import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

const apiTarget = process.env.PUBLIC_API_TARGET || process.env.VITE_API_TARGET || 'http://localhost:18080';

export default defineConfig({
  plugins: [tailwindcss(), sveltekit()],
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
