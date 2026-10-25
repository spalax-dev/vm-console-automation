import { defineConfig } from 'vite'
import { resolve } from 'node:path'

const __dirname = import.meta.dirname

export default defineConfig({
  root: './',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    rollupOptions: {
      input: {
        main: resolve(__dirname, 'index.html'),
      },
    },
  },
  server: {
    proxy: {
      '/api': {
        target: process.env.VMCONS_API ?? 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
