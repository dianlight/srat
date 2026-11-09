/**
 * msw-auto-mock emits invalid HTTP status codes for wildcard OpenAPI responses:
 * `{ status: 4 }` for `4XX` and `{ status: NaN }` for `default`. MSW then throws
 * `RangeError: Invalid status code: 4` from the interceptor, which surfaced as an
 * unhandled rejection once vitest stopped ignoring those.
 *
 * Run after `msw-auto-mock` (see `gen:mocks` in package.json) so a regeneration
 * cannot reintroduce the invalid codes.
 */
import { readFile, writeFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";

const handlersPath = fileURLToPath(
  new URL("../src/mocks/handlers.ts", import.meta.url),
);

const replacements = [
  ["{ status: 4 }", "{ status: 400 }"],
  ["{ status: NaN }", "{ status: 500 }"],
];

const source = await readFile(handlersPath, "utf8");
let output = source;

for (const [from, to] of replacements) {
  const count = output.split(from).length - 1;
  output = output.replaceAll(from, to);
  console.log(`fix-msw-mocks: ${from} -> ${to} (${count})`);
}

if (output === source) {
  console.log("fix-msw-mocks: nothing to change");
} else {
  await writeFile(handlersPath, output);
}
