import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  base: './',
  build: { outDir: 'dist', emptyOutDir: true },
  server: { port: Number(process.env.WAILS_VITE_PORT ?? 9245), strictPort: true },
})
