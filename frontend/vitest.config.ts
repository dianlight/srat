import { playwright } from '@vitest/browser-playwright';
import { defineConfig } from "vitest/config";

export default defineConfig({
    test: {
        environment: "happy-dom",
        globals: true,
        setupFiles: ["./test/happy-dom-setup.ts"],
        include: [
            "src/**/*.test.{ts,tsx}",
            "test/__tests__/**/*.test.{ts,tsx}",
        ],
        exclude: ["**/node_modules/**", "**/dist/**", "**/build/**"],
        // "forks" = one subprocess per file → strongest isolation for flaky
        // module/global state in the dev container. Slower, but more reliable
        // than thread workers for the migrated suite.
        pool: "forks",
        fileParallelism: false,
        testTimeout: 20000,
        hookTimeout: 20000,
        bail: 0,
        coverage: {
            provider: "v8",
            reporter: ["text", "lcov"],
            exclude: [
                "**/node_modules/**",
                "**/dist/**",
                "**/build/**",
                "**/coverage/**",
                "**/src/mocks/**",
                "**/macro/**",
                "**/.devcontainer/**",
                "src/store/sratApi.ts",
                "test/**",
            ],
        },
        browser: {
            provider: playwright(),
            instances: [{ browser: 'chromium' }]
        },
    },
});
