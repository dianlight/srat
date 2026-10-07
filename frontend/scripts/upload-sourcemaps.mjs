#!/usr/bin/env bun
// Upload frontend sourcemaps to Sentry so minified rc frames resolve (#1344).
// Best-effort: exits 0 with a warning when Sentry secrets are missing (forks,
// local builds). Expects external *.js.map files produced by bun.build.ts
// production builds (sourcemap: "external").
//
// Env:
//   SENTRY_AUTH_TOKEN (required for upload)
//   SENTRY_ORG (required)
//   SENTRY_PROJECT_FRONTEND (preferred) or SENTRY_PROJECT (fallback)
//   VERSION / npm_package_version (release; defaults to package.json version)
// Args:
//   <staticDir> [--release <version>]  (default: ../backend/src/web/static)

import { existsSync, readdirSync } from "node:fs";
import path from "node:path";
import packageJson from "../package.json" with { type: "json" };

const args = process.argv.slice(2).filter((a) => a !== "--");
let staticDir = args[0] && !args[0].startsWith("--") ? args[0] : "../backend/src/web/static";
const releaseFlag = args.indexOf("--release");
const release =
  (releaseFlag !== -1 ? args[releaseFlag + 1] : undefined) ||
  process.env.VERSION ||
  process.env.npm_package_version ||
  packageJson.version;

const authToken = process.env.SENTRY_AUTH_TOKEN || "";
const org = process.env.SENTRY_ORG || "";
const project = process.env.SENTRY_PROJECT_FRONTEND || process.env.SENTRY_PROJECT || "";

if (!authToken || !org || !project) {
  console.warn(
    "[sentry-sourcemaps] Skipping upload: missing SENTRY_AUTH_TOKEN, SENTRY_ORG, or SENTRY_PROJECT_FRONTEND/SENTRY_PROJECT.",
  );
  process.exit(0);
}

const resolvedDir = path.isAbsolute(staticDir)
  ? staticDir
  : path.resolve(import.meta.dir, "..", staticDir);

if (!existsSync(resolvedDir)) {
  console.warn(`[sentry-sourcemaps] Skipping upload: dir not found: ${resolvedDir}`);
  process.exit(0);
}

const maps = readdirSync(resolvedDir, { recursive: true }).filter(
  (f) => typeof f === "string" && f.endsWith(".js.map"),
);

if (maps.length === 0) {
  console.warn(`[sentry-sourcemaps] No *.js.map files in ${resolvedDir}; nothing to upload.`);
  process.exit(0);
}

console.log(
  `[sentry-sourcemaps] Uploading ${maps.length} sourcemap(s) for release ${release} to ${org}/${project}...`,
);

const proc = Bun.spawnSync(
  [
    "bunx",
    "sentry-cli",
    "sourcemaps",
    "upload",
    "--org",
    org,
    "--project",
    project,
    "--release",
    String(release),
    resolvedDir,
  ],
  {
    env: { ...process.env, SENTRY_AUTH_TOKEN: authToken, SENTRY_ORG: org },
    stdio: ["ignore", "inherit", "inherit"],
  },
);

if (proc.exitCode !== 0) {
  console.error(`[sentry-sourcemaps] Upload failed with exit code ${proc.exitCode}`);
  process.exit(proc.exitCode);
}

console.log("[sentry-sourcemaps] Upload complete.");
