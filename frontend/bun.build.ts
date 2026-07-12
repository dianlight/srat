//import copy from 'bun-copy-plugin';

import type { BuildConfig, BuildOutput } from "bun";
import { Glob } from "bun";
import path from "node:path";
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

// Eagerly import the config-updater BEFORE registering the global Bun plugin so
// the runtime import() isn't intercepted by the plugin's onLoad callback (which
// would cause "onLoad() expects an object returned" errors for module graph imports).
// @ts-ignore — internal dist imports (no public API for these)
const configUpdaterModule = inspectorEnabled
  ? await import(
      path.join(
        import.meta.dir,
        "node_modules/@mcpc-tech/unplugin-dev-inspector-mcp/dist/config-updater.js",
      ),
    ).catch((err: unknown) => {
      console.warn("[dev-inspector] configUpdater not available:", err);
      return undefined;
    })
  : undefined;

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

// ─── DevInspector MCP server (in-process) ────────────────────────────────────
// Instead of spawning the CLI server (which can't intercept stdio properly),
// we import the internal config-updater and run the server in-process, hooking
// console methods to capture output into the stdioLogs store for the inspector
// UI (GET /__inspector__/stdio).
//
// In Bun, console.log does NOT go through process.stdout.write (unlike Node.js),
// so we hook console methods directly rather than stream writes.

const DEFAULT_AGENT = process.env.DEV_INSPECTOR_DEFAULT_AGENT ?? "Opencode";
const VISIBLE_AGENTS =
  process.env.DEV_INSPECTOR_VISIBLE_AGENTS ?? "Opencode,Claude Code";

let inspectorServerCtx: {
  host: string;
  port: number;
  server: any;
} | undefined;

function spawnFallback() {
  const fallbackServer = Bun.spawn(
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
      env: { ...process.env, DEV_INSPECTOR_DISABLE_CHROME: "1" },
    },
  );
  inspectorServerCtx = {
    host: "localhost",
    port: 6137,
    server: fallbackServer,
  };
}

async function startInspectorServer() {
  if (!inspectorEnabled) return;

  const configUpdater = configUpdaterModule;
  if (!configUpdater) {
    console.warn(
      "[dev-inspector] configUpdater not loaded, falling back to spawned server",
    );
    spawnFallback();
    return;
  }

  try {
    const {
      a: addStdioLog,
      i: setupMcpMiddleware,
      n: setupAcpMiddleware,
      o: getDefaultPort,
      r: setupInspectorMiddleware,
      s: startStandaloneServer,
      t: updateMcpConfigs,
    } = configUpdater;

    // ─── Stdio interceptor ──────────────────────────────────────────────────────
    // In Bun, console.log does NOT go through process.stdout.write (unlike Node.js),
    // so we hook console methods directly to capture output into the stdioLogs store.

    const origLog = console.log;
    const origWarn = console.warn;
    const origError = console.error;
    const origInfo = console.info;
    const origDebug = console.debug;

    const capture = (
      stream: "stdout" | "stderr",
      original: typeof console.log,
    ) =>
      (...args: unknown[]) => {
        try {
          const text = args
            .map((a) => (typeof a === "string" ? a : JSON.stringify(a)))
            .join(" ");
          addStdioLog(stream, text);
        } catch {
          // swallow – never break the user's console
        }
        return original.apply(console, args);
      };

    console.log = capture("stdout", origLog) as typeof console.log;
    console.info = capture("stdout", origInfo) as typeof console.info;
    console.debug = capture("stdout", origDebug) as typeof console.debug;
    console.warn = capture("stderr", origWarn) as typeof console.warn;
    console.error = capture("stderr", origError) as typeof console.error;

    // ─── Start server in-process ────────────────────────────────────────────────

    const { server, host: inspectorHost, port: inspectorPort } =
      await startStandaloneServer({
        port: getDefaultPort(),
        host: "localhost",
      });

    const serverContext = {
      host: inspectorHost,
      port: inspectorPort,
      disableChrome: true,
    };

    await setupMcpMiddleware(server, serverContext);
    setupAcpMiddleware(server, serverContext, {});
    setupInspectorMiddleware(server, {
      disableChrome: true,
      defaultAgent: DEFAULT_AGENT,
      visibleAgents: VISIBLE_AGENTS.split(",")
        .map((a) => a.trim())
        .filter(Boolean),
    });

    const displayHost =
      inspectorHost === "0.0.0.0" ? "localhost" : inspectorHost;
    const mcpUrl = `http://${displayHost}:${inspectorPort}/__mcp__/sse`;

    await updateMcpConfigs(process.cwd(), mcpUrl, {});

    inspectorServerCtx = { host: displayHost, port: inspectorPort, server };

    console.log(`  MCP       → ${mcpUrl}`);
    console.log(`  Inspector → http://${displayHost}:${inspectorPort}/__inspector__/sidebar`);
    console.log(`  Agents    → ${VISIBLE_AGENTS} (default: ${DEFAULT_AGENT})`);
  } catch (err) {
    console.warn(
      "[dev-inspector] Failed to start in-process server:",
      err,
    );
    spawnFallback();
  }
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
    await startInspectorServer();
    // Give the server a moment to bind before the browser connects.
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
    const inspCtx = inspectorServerCtx;
    const inspHost = inspCtx?.host ?? "localhost";
    const inspPort = inspCtx?.port ?? 6137;
    console.log(`\n${"─".repeat(54)}`);
    console.log(`  App       → http://localhost:${app.port}`);
    console.log(`  HMR       → native (via Bun dev server)`);
    console.log(`  MCP       → http://${inspHost}:${inspPort}/__mcp__/sse`);
    console.log(`  Inspector → http://${inspHost}:${inspPort}/__inspector__/sidebar`);
    if (inspectorEnabled) {
      console.log(
        `  Agents    → ${VISIBLE_AGENTS} (default: ${DEFAULT_AGENT})`,
      );
    }
    console.log(`${"─".repeat(54)}\n`);
    console.log("  Ctrl+C to stop.\n");
    process.on("SIGINT", () => {
      console.log("\n[dev] Shutting down...");
      app.stop(true);
      process.exit(0);
    });
    process.on("SIGTERM", () => {
      app.stop(true);
      process.exit(0);
    });
  } else if (values.watch) {
    await startInspectorServer();
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
      process.exit(0);
    });
    process.on("SIGTERM", () => {
      srcwatch.close();
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
