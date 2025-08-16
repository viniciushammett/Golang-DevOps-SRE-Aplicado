import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/v1': 'http://localhost:8080',
      '/metrics': 'http://localhost:8080',
      '/healthz': 'http://localhost:8080'
    }
  },
  build: { outDir: 'dist' }
})