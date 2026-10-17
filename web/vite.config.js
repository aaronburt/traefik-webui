import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// The build lands in internal/server/dist so Go can embed it.
export default defineConfig({
  plugins: [svelte()],
  build: { outDir: '../internal/server/dist', emptyOutDir: true },
  // Dev only: point the Vite dev server at a running traefik-webui, for example TRAEFIK_WEBUI_URL=http://localhost:8081.
  server: { proxy: { '/api': process.env.TRAEFIK_WEBUI_URL ?? 'http://localhost:8081' } },
})
