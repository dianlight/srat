import { existsSync, readFileSync } from "node:fs";
import type { ProviderConfig, ProvidersConfig } from "./types.js";

export type BrokerEnv = {
  BROKER_PUBLIC_URL: string;
  BROKER_API_TOKEN: string;
  BROKER_PROVIDERS_FILE?: string;
  BROKER_PROVIDERS_JSON?: string;
  DROPBOX_CLIENT_ID?: string;
  DROPBOX_CLIENT_SECRET?: string;
  SESSION_TTL?: string;
  PORT?: string;
  BROKER_DISABLE_AUTH?: string;
  BROKER_ALLOWED_CALLBACK_PATTERNS?: string;
};

/** Built-in provider defaults (only credentials required from config). */
export const BUILTIN_PROVIDERS: Record<string, Partial<ProviderConfig>> = {
  dropbox: {
    authorize_url: "https://www.dropbox.com/oauth2/authorize",
    token_url: "https://api.dropboxapi.com/oauth2/token",
    auth_params: { token_access_type: "offline" },
  },
};

/**
 * Determines the session lifetime from the `SESSION_TTL` environment value.
 *
 * @param env - Environment variables containing the optional session TTL
 * @returns The session lifetime in seconds, with a default of 600 seconds and a minimum of 60 seconds
 */
export function getSessionTtlSeconds(env: Record<string, string | undefined>): number {
  const raw = env.SESSION_TTL?.trim();
  if (!raw) return 600;
  // Accept "10m", "600", "600s"
  let n: number;
  if (/^\d+$/.test(raw)) n = parseInt(raw, 10);
  else if (/^\d+m$/.test(raw)) n = parseInt(raw.replace("m", ""), 10) * 60;
  else if (/^\d+s$/.test(raw)) n = parseInt(raw.replace("s", ""), 10);
  else {
    const parsed = parseInt(raw, 10);
    if (Number.isNaN(parsed)) {
      console.warn(`[broker] SESSION_TTL unparseable "${raw}" – falling back to 600s`);
      return 600;
    }
    n = parsed;
  }
  if (Number.isNaN(n) || n <= 0) {
    console.warn(`[broker] SESSION_TTL "${raw}" out of range – using 60s minimum`);
    return 60;
  }
  return Math.max(n, 60);
}

/**
 * Gets the configured broker public URL without surrounding whitespace or trailing slashes.
 *
 * @param env - Environment variables containing the broker public URL
 * @returns The normalized broker public URL, or an empty string when none is configured
 */
export function getBrokerPublicUrl(env: Record<string, string | undefined>): string {
  const raw = (env.BROKER_PUBLIC_URL || "").trim().replace(/\/+$/, "");
  return raw;
}

/**
 * Determines whether a URL uses HTTP and targets a loopback host.
 *
 * @param url - The URL to evaluate
 * @returns `true` if the URL targets localhost or a loopback IP address over HTTP, `false` otherwise.
 */
function isLoopbackBrokerUrl(url: URL): boolean {
  if (url.protocol !== "http:") return false;
  const host = url.hostname.toLowerCase();
  return host === "localhost" || host === "127.0.0.1" || host === "::1" || host === "[::1]";
}

/**
 * Determines whether a broker public URL is valid.
 *
 * @param raw - The URL to validate
 * @returns `true` if the URL uses HTTPS or targets a loopback address over HTTP, `false` otherwise
 */
export function isValidBrokerPublicUrl(raw: string): boolean {
  if (!raw) return false;
  try {
    const u = new URL(raw);
    if (u.protocol === "https:") return true;
    if (isLoopbackBrokerUrl(u)) return true;
    return false;
  } catch {
    return false;
  }
}

/**
 * Validates and retrieves the broker's public URL.
 *
 * @param env - Environment variables containing `BROKER_PUBLIC_URL`
 * @returns The normalized broker public URL
 * @throws If `BROKER_PUBLIC_URL` is missing or invalid
 */
export function getBrokerPublicUrlOrThrow(env: Record<string, string | undefined>): string {
  const raw = getBrokerPublicUrl(env);
  if (!raw) throw new Error("BROKER_PUBLIC_URL is not configured");
  if (!isValidBrokerPublicUrl(raw)) throw new Error("BROKER_PUBLIC_URL must be an absolute https URL (loopback http allowed for dev)");
  return raw;
}

/**
 * Loads provider configurations from environment-specified sources and applies built-in defaults.
 *
 * @param env - Environment variables containing provider configuration sources and Dropbox credentials
 * @returns The merged provider configuration
 */
export function loadProvidersConfig(env: Record<string, string | undefined>): ProvidersConfig {
  const cfg: ProvidersConfig = {};

  // 1) From file if BROKER_PROVIDERS_FILE set
  const filePath = env.BROKER_PROVIDERS_FILE?.trim();
  if (filePath) {
    try {
      if (existsSync(filePath)) {
        const raw = readFileSync(filePath, "utf-8");
        const parsed = JSON.parse(raw) as ProvidersConfig;
        for (const [k, v] of Object.entries(parsed)) {
          cfg[k] = { ...v };
        }
      }
    } catch (e) {
      console.warn(`[broker] malformed BROKER_PROVIDERS_FILE JSON ignored: ${(e as Error).message}`);
    }
  }

  // 2) From inline JSON env (Workers secret/KV alternative)
  const inlineJson = env.BROKER_PROVIDERS_JSON?.trim();
  if (inlineJson) {
    try {
      const parsed = JSON.parse(inlineJson) as ProvidersConfig;
      for (const [k, v] of Object.entries(parsed)) {
        cfg[k] = { ...cfg[k], ...v };
      }
    } catch (e) {
      console.warn(`[broker] malformed BROKER_PROVIDERS_JSON ignored: ${(e as Error).message}`);
    }
  }

  // 3) Shorthand DROPBOX_* env
  const dropboxId = env.DROPBOX_CLIENT_ID?.trim();
  const dropboxSecret = env.DROPBOX_CLIENT_SECRET?.trim();
  if (dropboxId || dropboxSecret) {
    cfg.dropbox = {
      ...cfg.dropbox,
      client_id: dropboxId || cfg.dropbox?.client_id || "",
      client_secret: dropboxSecret || cfg.dropbox?.client_secret || "",
    };
  }

  // 4) Apply built-in defaults for known providers where missing authorize_url/token_url/auth_params
  for (const [name, defaults] of Object.entries(BUILTIN_PROVIDERS)) {
    if (cfg[name]) {
      const existing = cfg[name];
      cfg[name] = {
        ...existing,
        auth_params: { ...defaults.auth_params, ...existing.auth_params },
      };
      if (!cfg[name].authorize_url) cfg[name].authorize_url = defaults.authorize_url;
      if (!cfg[name].token_url) cfg[name].token_url = defaults.token_url;
    }
  }

  return cfg;
}

/**
 * Retrieves a fully configured provider by name.
 *
 * @param providers - The available provider configurations
 * @param name - The provider name to retrieve
 * @returns The provider configuration with client credentials and authorization and token URLs
 * @throws If the provider is unknown or missing required credentials or URLs
 */
export function getProviderOrThrow(
  providers: ProvidersConfig,
  name: string
): ProviderConfig & { authorize_url: string; token_url: string } {
  const p = providers[name];
  if (!p) throw new Error(`unknown provider: ${name}`);
  if (!p.client_id || !p.client_secret) throw new Error(`provider ${name} missing client_id/client_secret`);
  if (!p.authorize_url) throw new Error(`provider ${name} missing authorize_url`);
  if (!p.token_url) throw new Error(`provider ${name} missing token_url`);
  return p as ProviderConfig & { authorize_url: string; token_url: string };
}

/** Max srat_callback_url length to prevent header overflow / DoS (also applied to BROKER_PUBLIC_URL). */
export const MAX_CALLBACK_URL_LENGTH = 2048;

/** Max instance_id length (client-provided opaque id, e.g. HA uuid). */
export const MAX_INSTANCE_ID_LENGTH = 128;

/** Default instance registration TTL: 1h (3600s) per spec – not configurable to avoid weakening security. */
export const INSTANCE_TTL_SECONDS = 3600;

/**
 * Determines the instance registration lifetime.
 * Fixed to 1h per spec; reads INSTANCE_TTL env only as override for tests.
 */
export function getInstanceTtlSeconds(env: Record<string, string | undefined>): number {
  const raw = env.INSTANCE_TTL?.trim();
  if (!raw) return INSTANCE_TTL_SECONDS;
  // Accept "1h", "60m", "3600", "3600s"
  if (/^\d+h$/.test(raw)) return Math.max(parseInt(raw, 10) * 3600, 60);
  if (/^\d+m$/.test(raw)) return Math.max(parseInt(raw.replace("m", ""), 10) * 60, 60);
  if (/^\d+s$/.test(raw)) return Math.max(parseInt(raw.replace("s", ""), 10), 60);
  if (/^\d+$/.test(raw)) return Math.max(parseInt(raw, 10), 60);
  console.warn(`[broker] INSTANCE_TTL unparseable "${raw}" – falling back to ${INSTANCE_TTL_SECONDS}s`);
  return INSTANCE_TTL_SECONDS;
}

/**
 * Convert a glob pattern (e.g. "https://*.example.com/*") to RegExp.
 * Supports * (any chars except / for host part is relaxed) – deliberately simple, not shell glob.
 */
export function globToRegExp(glob: string): RegExp {
  // Escape regex specials except * and ?
  const escaped = glob.replace(/([.+^${}()|[\]\\])/g, "\\$1");
  const regexStr = "^" + escaped.replace(/\*/g, ".*").replace(/\?/g, ".") + "$";
  return new RegExp(regexStr);
}

export function parseAllowedCallbackPatterns(raw: string | undefined): string[] {
  if (!raw) return [];
  return raw
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
}

export function isAllowedSratCallbackUrl(raw: string, allowlistRaw: string | undefined): boolean {
  const patterns = parseAllowedCallbackPatterns(allowlistRaw);
  if (patterns.length === 0) return true; // permissive default (any https) – isValid already checked
  try {
    // URL must be valid first
    const u = new URL(raw);
    // Try each glob
    for (const pat of patterns) {
      // If pattern looks like a plain host (no scheme), treat as "*://host/*"
      let glob = pat;
      if (!glob.includes("://")) {
        glob = `https://${glob}`;
        if (!glob.endsWith("*") && !glob.includes("/")) glob += "/*";
      }
      const re = globToRegExp(glob);
      if (re.test(raw) || re.test(u.origin + u.pathname)) return true;
      // Also test against origin only for simple host patterns
      if (re.test(u.origin) || re.test(u.host) || re.test(u.hostname)) return true;
    }
    return false;
  } catch {
    return false;
  }
}

export function isProductionEnv(env: Record<string, string | undefined>): boolean {
  const url = (env.BROKER_PUBLIC_URL || "").toLowerCase();
  return url.includes("production") || (env.ENV || "").toLowerCase() === "production" || (env.WORKERS_ENV || "").toLowerCase() === "production";
}
