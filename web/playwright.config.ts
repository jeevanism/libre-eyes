import { defineConfig, devices } from '@playwright/test'

const liveBaseURL = process.env.LIBREEYES_E2E_BASE_URL

export default defineConfig({
  testDir: './tests',
  use: {
    baseURL: liveBaseURL ?? 'http://127.0.0.1:4173',
    trace: 'retain-on-failure',
    launchOptions: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH
      ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH }
      : {},
  },
  // Live API tests attach to `make dev`; ordinary browser tests use an isolated preview.
  ...(liveBaseURL ? {} : {
    webServer: {
      command: 'npm run build && npm run preview -- --host 127.0.0.1 --port 4173',
      url: 'http://127.0.0.1:4173',
      reuseExistingServer: true,
    },
  }),
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
    {
      name: 'tablet-chromium',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 820, height: 1180 },
        hasTouch: true,
      },
    },
  ],
})
