/// <reference types="vitest/config" />
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig, loadEnv } from 'vite'
import { VitePWA } from 'vite-plugin-pwa'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, '..', '')
  return {
    envDir: '..',
    server: {
      // The dev server forwards /api to the Go backend, so a phone on the same
      // Wi-Fi only needs to reach this one port (no CORS, no second URL).
      proxy: { '/api': `http://localhost:${env.PORT || 8081}` },
    },
    plugins: [
      react(),
      tailwindcss(),
      VitePWA({
        registerType: 'autoUpdate',
        includeAssets: ['favicon.svg', 'apple-touch-icon.png'],
        manifest: {
          name: 'Travel Split',
          short_name: 'TravelSplit',
          description: 'หารค่าใช้จ่ายทริปกับเพื่อน',
          lang: 'th',
          theme_color: '#0f766e',
          background_color: '#ffffff',
          display: 'standalone',
          start_url: '/',
          icons: [
            { src: 'icon-192.png', sizes: '192x192', type: 'image/png', purpose: 'any' },
            { src: 'icon-512.png', sizes: '512x512', type: 'image/png', purpose: 'any' },
            { src: 'icon-maskable-512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
            { src: 'favicon.svg', sizes: 'any', type: 'image/svg+xml', purpose: 'any' },
          ],
        },
        workbox: {
          // API responses must stay fresh; only the app shell is precached.
          navigateFallbackDenylist: [/^\/api\//],
        },
      }),
    ],
    test: {
      environment: 'happy-dom',
      setupFiles: ['./src/test-setup.ts'],
      css: false,
    },
  }
})
