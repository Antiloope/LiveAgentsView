import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules/pixi.js') || id.includes('node_modules/@pixi/')) return 'pixi'
        },
      },
    },
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8420',
    },
  },
})
