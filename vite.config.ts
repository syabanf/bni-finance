import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { VitePWA } from 'vite-plugin-pwa'
import { fileURLToPath, URL } from 'node:url'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: 'autoUpdate',
      includeAssets: ['favicon.svg', 'apple-touch-icon.png'],
      manifest: {
        name: 'Finance Hub BNI Indonesia',
        short_name: 'Finance Hub',
        description:
          'Sistem internal invoice dan iuran keanggotaan BNI (Business Networking International) Indonesia. Tidak terkait Bank BNI.',
        id: '/',
        start_url: '/',
        scope: '/',
        display: 'standalone',
        orientation: 'portrait',
        background_color: '#ffffff',
        theme_color: '#ffffff',
        lang: 'id',
        // Tanpa 'finance': bersama nama "BNI", kategori itu membuat aplikasi ini
        // terbaca sebagai aplikasi bank.
        categories: ['business', 'productivity'],
        icons: [
          { src: 'pwa-192x192.png', sizes: '192x192', type: 'image/png' },
          { src: 'pwa-512x512.png', sizes: '512x512', type: 'image/png' },
          { src: 'pwa-512x512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
        ],
      },
      workbox: {
        globPatterns: ['**/*.{js,css,html,svg,png,woff2}'],
        // Koleksi Konsol API hanya dibuka admin; jangan ikut diunduh semua orang.
        globIgnores: ['**/api-collection-*.js'],
        navigateFallback: 'index.html',
        cleanupOutdatedCaches: true,
      },
      devOptions: {
        // Enable the SW + manifest in `npm run dev` so the PWA is testable locally.
        enabled: true,
        type: 'module',
      },
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: process.env.PORT ? Number(process.env.PORT) : 5173,
    proxy: {
      '/api/bni-vm': {
        target: 'https://www.bni-vh.com/api/external/v1',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/bni-vm/, ''),
      },
    },
  },
})
