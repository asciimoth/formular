import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/e2e",
  use: {
    browserName: "chromium",
    launchOptions: process.env.CHROME_BIN ? { executablePath: process.env.CHROME_BIN } : undefined
  },
  webServer: {
    command: "pnpm exec vite --host 127.0.0.1",
    port: 5173,
    reuseExistingServer: true
  }
});
