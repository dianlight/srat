/**
 * ESLint (flat config) for the SRAT frontend.
 *
 * Scoped to src TSX files only. Biome remains the primary linter; ESLint adds
 * testing-library-aware rules that Biome cannot express, plus the custom
 * src/testIds.ts registry rule. No `eslint:recommended` and no type-aware
 * rules, so the surface is intentionally tiny.
 */
import babelParser from "@babel/eslint-parser";
import reactHooks from "eslint-plugin-react-hooks";
import testingLibrary from "eslint-plugin-testing-library";
import srat from "./eslint/srat-plugin.js";

export default [
  {
    files: ["src/**/*.tsx"],
    ignores: [
      "src/store/sratApi.ts", // code-generated; formatted by rtk-query-codegen
      "src/mocks/**", // msw-auto-mock output
    ],
    languageOptions: {
      parser: babelParser,
      // Babel parses TSX without depending on the TypeScript runtime, so this
      // stays compatible with the repo's TypeScript 7.0 toolchain
      // (typescript-eslint cannot load against a TS 7 programmatic API yet).
      parserOptions: {
        requireConfigFile: false,
        babelOptions: {
          presets: ["@babel/preset-typescript", "@babel/preset-react"],
        },
      },
    },
    // The codebase carries inline eslint-disable-next-line directives for rules
    // we do not enable (e.g. react-hooks/exhaustive-deps, no-console); do not
    // flag those directives as unused.
    linterOptions: { reportUnusedDisableDirectives: "off" },
    plugins: {
      "react-hooks": reactHooks,
      srat,
      "testing-library": testingLibrary,
    },
    rules: {
      // Rule registered only so existing eslint-disable-next-line
      // react-hooks/exhaustive-deps directives in components stay valid.
      "react-hooks/exhaustive-deps": "off",
      // Custom: every data-testid / *ByTestId literal must be in src/testIds.ts
      "srat/registered-test-id": "error",
      // Project rule (see .opencode/instructions/frontend_test.instructions.md):
      // user-event only, never fireEvent.
      "testing-library/prefer-user-event": "error",
      // The `within(container)` / `container.firstChild` pattern is still used
      // across the suite; keep these visible as warnings until it is migrated.
      "testing-library/no-container": "warn",
      "testing-library/no-node-access": "warn",
    },
  },
];
