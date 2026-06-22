import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [tailwindcss(), sveltekit()],
  server: {
    port: 5180,
    host: true,
    strictPort: false,
    proxy: {
      '/api': {
        target: process.env.PUBLIC_API_TARGET || 'http://localhost:8080',
        changeOrigin: true,
      },
      '/auth/feishu': {
        target: process.env.PUBLIC_API_TARGET || 'http://localhost:8080',
        changeOrigin: true,
      },
      '/oauth2': {
        target: process.env.PUBLIC_API_TARGET || 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
});
