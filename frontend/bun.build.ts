//import copy from 'bun-copy-plugin';

import type { BuildConfig, BuildOutput, Subprocess } from "bun";
import { Glob } from "bun";
import { watch } from "node:fs";
import { parseArgs } from "node:util";
import App from "./src/index.html";
import { devInspectorPlugin } from "./inspector-plugin";

const { values } = parseArgs({
  args: Bun.argv,
  options: {
    watch: {
      type: "boolean",
      default: false,
      short: "w",
      description: "Watch for changes and rebuild automatically",
    },
    serve: {
      type: "boolean",
      short: "s",
      description: "Start the HTTP Server",
    },
    outDir: {
      type: "string",
      short: "o",
      default: "./out",
      description: "Specify the output directory (default: ./out)",
    },
    version: {
      type: "string",
      short: "v",
      description: "Build version (e.g. 1.0.0)",
    },
    apiUrl: {
      type: "string",
      short: "a",
      description:
        "API URL to use in the frontend (overrides API_URL env variable)",
    },
    noInspector: {
      type: "boolean",
      default: false,
      description:
        "Disable DevInspector (skip MCP server spawn & JSX instrumentation)",
    },
  },
  strict: true,
  allowPositionals: true,
});

const inspectorEnabled =
  !values.noInspector &&
  process.env.DEV_INSPECTOR_DISABLED !== "1" &&
  (values.watch || values.serve);

// Register the DevInspector JSX transform as a global Bun plugin so both
// Bun.serve() (static serve) and Bun.build() (watch mode) apply it.
// The plugin is a no-op outside .tsx/.jsx files and skips node_modules.
if (inspectorEnabled) {
  Bun.plugin(devInspectorPlugin);
  // Disable chrome devtools mcp by default to avoid hijacking chrome instances
  if (process.env.DEV_INSPECTOR_DISABLE_CHROME === undefined) {
    process.env.DEV_INSPECTOR_DISABLE_CHROME = "1";
  }
}

const buildConfig: BuildConfig = {
  entrypoints: ["src/index.html" /*, 'src/index.tsx' */],
  outdir: values.outDir, // Specify the output directory
  //experimentalCss: true,
  naming: {
    entry: "[dir]/[name].[ext]",
    chunk: "[name]-[hash].[ext]",
    asset: "[name].[ext]",
  },
  target: "browser",
  // Use inline sourcemaps during watch/serve for better DX, external in production to avoid bloating bundles
  sourcemap: "inline", // values.watch || values.serve ? "inline" : "external",
  minify: values.watch || values.serve ? false : true,
  // Avoid bundling optional prism-based highlighters that reference unsupported subpath exports
  external: [
    "refractor",
    "refractor/*",
    "react-syntax-highlighter/dist/esm/prism*",
  ],
  plugins: [
    //copy("src/index.html", "out/index.html")
    //  html({})
  ],
  tsconfig:
    values.watch || values.serve ? "tsconfig.json" : "tsconfig.prod.json",
  /*
	define: {
		"process.env.APIURL": APIURL,
		"process.env.NODE_ENV": values.watch || values.serve ? "'development'" : "'production'",
		"process.env.VITE_SENTRY_DSN": `"${process.env.VITE_SENTRY_DSN || 'disabled'}"`,
		"process.env.SERVER_EVENT_BACKEND": "'WS'", // SSE or WS
	},
	*/
};

// ─── DevInspector MCP server (standalone) ───────────────────────────────────
// Spawned in serve/watch mode so AI agents can connect to the MCP endpoint
// at http://localhost:6137/__mcp__/sse and the inspector UI/sidebar at
// http://localhost:6137/__inspector__/*.
// Requires the patched CLI to support --default-agent / --visible-agents.
const DEFAULT_AGENT = process.env.DEV_INSPECTOR_DEFAULT_AGENT ?? "Opencode";
const VISIBLE_AGENTS =
  process.env.DEV_INSPECTOR_VISIBLE_AGENTS ?? "Opencode,Claude Code";

function spawnInspectorServer(): Subprocess | undefined {
  if (!inspectorEnabled) return undefined;
  console.log("\n[dev-inspector] Starting standalone MCP server...");
  const server = Bun.spawn(
    [
      process.execPath,
      "x",
      "@mcpc-tech/unplugin-dev-inspector-mcp",
      "server",
      "--default-agent",
      DEFAULT_AGENT,
      "--visible-agents",
      VISIBLE_AGENTS,
    ],
    {
      cwd: import.meta.dir,
      stdin: "inherit",
      stdout: "inherit",
      stderr: "inherit",
      env: process.env,
    },
  );
  return server;
}

async function build(): Promise<BuildOutput | undefined> {
  process.env.VERSION = values.version || process.env.VERSION || "dev";
  process.env.API_URL = values.apiUrl || process.env.API_URL || "";
  console.log("Build Frontend with:");
  console.log("\tVersion: ", process.env.VERSION || "not provided");
  console.log("\tNode Environment: ", process.env.NODE_ENV || "not provided");
  console.log("\tAPI URL: ", process.env.API_URL || "not provided");
  console.log("\tSentry DSN: ", process.env.VITE_SENTRY_DSN || "not provided");
  console.log("\tDevInspector: ", inspectorEnabled ? "enabled" : "disabled");
  let mcpServer: Subprocess | undefined = undefined;
  //console.log("\tSentry DSN: ", process.env.VITE_SENTRY_DSN || "not provided");
  if (!values.serve && !values.watch) {
    console.log(`\tMode: Build ${import.meta.dir}/src -> ${values.outDir}`);
    return Bun.build(buildConfig).then((result) => {
      if (!result.success) {
        console.error("Build failed");
        for (const message of result.logs) {
          // Bun will pretty print the message object
          console.error(message);
        }
      }
      return result;
    });
  } else if (values.serve) {
    mcpServer = spawnInspectorServer();
    // Give the MCP server a moment to bind before the browser connects.
    await Bun.sleep(1500);
    console.log(`\tMode: Serve ${values.outDir}`);
    const app = Bun.serve({
      routes: {
        "/*": App,
      },
      port: 3080,
      idleTimeout: 60,
      development: {
        chromeDevToolsAutomaticWorkspaceFolders: true,
        console: true,
        hmr: true,
      },
    });
    console.log(`\n${"─".repeat(54)}`);
    console.log(`  App       → http://localhost:${app.port}`);
    console.log(`  HMR       → native (via Bun dev server)`);
    console.log(`  MCP       → http://localhost:6137/__mcp__/sse`);
    console.log(`  Inspector → http://localhost:6137/__inspector__/sidebar`);
    if (inspectorEnabled) {
      console.log(
        `  Agents    → ${VISIBLE_AGENTS} (default: ${DEFAULT_AGENT})`,
      );
    }
    console.log(`${"─".repeat(54)}\n`);
    console.log("  Ctrl+C to stop.\n");
    process.on("SIGINT", () => {
      console.log("\n[dev] Shutting down...");
      mcpServer?.kill();
      app.stop(true);
      process.exit(0);
    });
    process.on("SIGTERM", () => {
      mcpServer?.kill();
      app.stop(true);
      process.exit(0);
    });
  } else if (values.watch) {
    mcpServer = spawnInspectorServer();
    await Bun.sleep(1500);
    console.log(`\tMode: Watch ${import.meta.dir}/src -> ${values.outDir}`);
    async function rebuild(event: string, filename: string | null) {
      console.log(`Detected ${event} in ${filename}`);

      // Only clean up old index files if the output directory exists
      try {
        const glob = new Glob(`index-*`);
        for await (const file of glob.scan(values.outDir)) {
          console.log(`D ${values.outDir}/${file}`);
          Bun.file(`${values.outDir}/${file}`)
            .delete()
            .catch((_err) => {});
        }
      } catch (_err) {
        // Directory might not exist yet on first build
      }

      Bun.build(buildConfig).then((result) => {
        if (!result.success) {
          console.error("Build failed");
          for (const message of result.logs) {
            // Bun will pretty print the message object
            console.error(message);
          }
        }
        return result;
      });
      console.log("\tReBuild complete ✅");
    }

    // Debounce to avoid hammering rebuilds on bulk saves.
    let debounceTimer: ReturnType<typeof setTimeout> | undefined;
    const triggerRebuild = (event: string, filename: string | null) => {
      if (debounceTimer) clearTimeout(debounceTimer);
      debounceTimer = setTimeout(() => rebuild(event, filename), 100);
    };

    const srcwatch = watch(
      `${import.meta.dir}/src`,
      { recursive: true },
      async (event, filename) => {
        triggerRebuild(event, filename);
      },
    );

    process.on("SIGINT", () => {
      console.log("\n[dev] Shutting down...");
      srcwatch.close();
      mcpServer?.kill();
      process.exit(0);
    });
    process.on("SIGTERM", () => {
      srcwatch.close();
      mcpServer?.kill();
      process.exit(0);
    });
    rebuild("initial build", null);
  }
  return undefined;
}

await build();
console.log(
  `Build complete ✅ [:${values.watch ? "👁️:watching" : values.serve ? "🌐:serving" : "🧻:build"}]`,
);
