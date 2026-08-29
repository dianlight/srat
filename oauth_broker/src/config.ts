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
};

/** Built-in provider defaults (only credentials required from config). */
export const BUILTIN_PROVIDERS: Record<string, Partial<ProviderConfig>> = {
  dropbox: {
    authorize_url: "https://www.dropbox.com/oauth2/authorize",
    token_url: "https://api.dropboxapi.com/oauth2/token",
    auth_params: { token_access_type: "offline" },
  },
};

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
    n = Number.isNaN(parsed) ? 600 : parsed;
  }
  if (Number.isNaN(n) || n <= 0) return 60;
  return Math.max(n, 60);
}

export function getBrokerPublicUrl(env: Record<string, string | undefined>): string {
  const raw = (env.BROKER_PUBLIC_URL || "").trim().replace(/\/+$/, "");
  return raw;
}

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
    } catch {
      // Best-effort: malformed JSON yields empty, caller will error on missing provider
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
    } catch {
      // ignore malformed inline json
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
