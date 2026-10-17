import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// The build lands in internal/server/dist so Go can embed it.
export default defineConfig({
  plugins: [svelte()],
  build: { outDir: '../internal/server/dist', emptyOutDir: true },
  server: { proxy: { '/api': 'http://localhost:8081' } },
})
