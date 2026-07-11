/**
 * Bun onLoad plugin for dev-inspector-mcp.
 *
 * Injects `data-insp-path="file:line:col:tag"` attributes into every JSX/TSX
 * element at build time, enabling accurate click-to-source in the inspector UI.
 *
 * The key insight: transformCode() only *adds* attributes — it does not compile
 * JSX away.  Returning { loader: "tsx" } tells Bun to still run its own JSX
 * transpilation on the augmented source, which is exactly what we want.
 */

import type { BunPlugin } from "bun";
// @ts-ignore — no bundled types, but the CJS default export is the transformCode fn
import { transformCode } from "@code-inspector/core";

export const devInspectorPlugin: BunPlugin = {
  name: "dev-inspector-mcp",
  setup(build) {
    // Mirror the filter from src/utils/file-type-detector.ts:7
    // Also handles plain .ts/.js so non-JSX files pass through untouched.
    build.onLoad({ filter: /\.(jsx|tsx|js|ts)$/ }, async (args) => {
      // Never transform installed packages
      if (args.path.includes("node_modules")) {
        return undefined; // let Bun handle it normally
      }

      const source = await Bun.file(args.path).text();

      // Only attempt the JSX transform on files that contain JSX syntax.
      // transformCode is fast and handles non-JSX files gracefully, but the
      // quick-bailout avoids unnecessary Babel parses on plain .ts files.
      const isJsx = args.path.endsWith(".jsx") || args.path.endsWith(".tsx");
      const loader = isJsx ? "tsx" : ("ts" as "tsx" | "ts");

      if (!isJsx) {
        // Plain TypeScript — no JSX to instrument, pass through.
        return { contents: source, loader };
      }

      try {
        // transformCode returns a Promise in @code-inspector/core >=1.4.x
        const transformed = await transformCode({
          content: source,
          filePath: args.path,
          fileType: "jsx", // @code-inspector/core uses "jsx" for both jsx & tsx
          escapeTags: [],  // no tags to skip
          pathType: "absolute",
        });

        return { contents: transformed, loader };
      } catch (err) {
        // On transform failure fall back to unmodified source so the app still works.
        console.warn(`[dev-inspector] Transform failed for ${args.path}:`, err);
        return { contents: source, loader };
      }
    });
  },
};
