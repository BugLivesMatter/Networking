import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/auth': 'http://localhost:4200',
      '/categories': 'http://localhost:4200',
      '/products': 'http://localhost:4200',
      '/files': 'http://localhost:4200',
      '/profile': 'http://localhost:4200',
      '/health': 'http://localhost:4200',
    },
  },
})
