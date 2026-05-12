import react from "@vitejs/plugin-react";
import { playwright } from "@vitest/browser-playwright";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [react()],
  test: {
    setupFiles: ["./test/browser-setup.ts"],
    browser: {
      enabled: true,
      headless: true,
      provider: playwright(),
      // https://vitest.dev/config/browser/playwright
      instances: [
        {
          browser: "chromium",
          headless: true,
        },
      ],
    },
  },
});
