/**
 * Custom ESLint rules for the SRAT frontend.
 *
 * `registered-test-id` enforces the src/testIds.ts convention:
 * - Every `data-testid` attribute value must be registered, unless it is a
 *   library-generated PascalCase id (MUI icons) or a throwaway `mock-*`
 *   fixture id used by test mock renderers.
 * - Every `*ByTestId(...)` query argument must satisfy the same rule, so a
 *   renamed/removed registry entry surfaces as a lint error in the tests.
 */
import { readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

// The registry is TypeScript, so we extract string literals textually instead
// of importing it at lint time. Values in testIds.ts are plain kebab-case
// strings, so this regex is safe.
const registrySource = readFileSync(
  join(dirname(fileURLToPath(import.meta.url)), "..", "src", "testIds.ts"),
  "utf8",
);
const registeredIds = new Set(
  [...registrySource.matchAll(/"([a-z0-9-]+)"/g)].map((m) => m[1]),
);

const isLibraryId = (id) => /^[A-Z][A-Za-z0-9]*$/.test(id); // MUI auto-testids
const isMockFixtureId = (id) => id.startsWith("mock-");
const isKebabCase = (id) => /^[a-z0-9]+(-[a-z0-9]+)*$/.test(id);

const BY_TEST_ID_CALL = /^(get|find|query|getAll|findAll|queryAll)ByTestId$/;

export default {
  rules: {
    "registered-test-id": {
      meta: {
        type: "problem",
        docs: { description: "Test ids must be registered in src/testIds.ts" },
        messages: {
          unregistered:
            '"{{ id }}" is not in src/testIds.ts — register it before using it.',
          notKebab:
            'Test id "{{ id }}" must be kebab-case, e.g. "page-component-element".',
        },
      },
      create(context) {
        const checkId = (id, node) => {
          if (registeredIds.has(id)) return;
          if (isLibraryId(id) || isMockFixtureId(id)) return;
          context.report({ node, messageId: "unregistered", data: { id } });
        };

        return {
          JSXAttribute(node) {
            if (node.name.name !== "data-testid") return;
            if (node.value?.type !== "Literal" || typeof node.value.value !== "string") {
              return;
            }
            const id = node.value.value;
            checkId(id, node);
            if (!isKebabCase(id) && !isLibraryId(id) && !isMockFixtureId(id)) {
              context.report({ node, messageId: "notKebab", data: { id } });
            }
          },
          CallExpression(node) {
            const callee = node.callee;
            const fnName =
              callee?.type === "Identifier" ? callee.name : callee?.property?.name;
            if (typeof fnName !== "string" || !BY_TEST_ID_CALL.test(fnName)) return;
            const arg = node.arguments[0];
            if (arg?.type === "Literal" && typeof arg.value === "string") {
              checkId(arg.value, arg);
            }
          },
        };
      },
    },
  },
};
