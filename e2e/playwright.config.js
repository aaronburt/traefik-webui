const path = require('path')
const { defineConfig } = require('@playwright/test')

const mockPort = 18095
const appPort = 18081
const work = path.join(__dirname, '.tmp')

module.exports = defineConfig({
  testDir: '.',
  // One shared server whose first run creates the admin password, so tests run in order.
  workers: 1,
  fullyParallel: false,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: `http://127.0.0.1:${appPort}`,
    trace: 'retain-on-failure',
    // Locally, point at an installed Chromium instead of downloading one.
    launchOptions: process.env.CHROMIUM_PATH ? { executablePath: process.env.CHROMIUM_PATH } : {},
  },
  webServer: [
    {
      command: 'node mock-traefik.js',
      env: { MOCK_PORT: String(mockPort) },
      url: `http://127.0.0.1:${mockPort}/api/version`,
      reuseExistingServer: false,
    },
    {
      // The binary is built from the repository root: go build -o bin/traefik-webui ./cmd/traefik-webui
      command: `sh -c 'rm -rf ${work} && mkdir -p ${work}/dynamic ${work}/data && exec ../bin/traefik-webui'`,
      env: {
        TRAEFIK_API_URL: `http://127.0.0.1:${mockPort}`,
        CONFIG_DIR: path.join(work, 'dynamic'),
        DATA_DIR: path.join(work, 'data'),
        LISTEN_ADDR: `127.0.0.1:${appPort}`,
      },
      url: `http://127.0.0.1:${appPort}/healthz`,
      reuseExistingServer: false,
    },
  ],
})
