import { describe, it, expect } from "vitest";
import { loadProvidersConfig, getProviderOrThrow, BUILTIN_PROVIDERS } from "../src/config.js";
import { writeFileSync, mkdirSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

describe("providers: file + env both", () => {
  it("supports BROKER_PROVIDERS_FILE", () => {
    const dir = join(tmpdir(), `broker-test-${Date.now()}`);
    mkdirSync(dir, { recursive: true });
    const file = join(dir, "providers.json");
    writeFileSync(
      file,
      JSON.stringify({
        gdrive: {
          client_id: "g-id",
          client_secret: "g-sec",
          authorize_url: "https://accounts.google.com/o/oauth2/v2/auth",
          token_url: "https://oauth2.googleapis.com/token",
          scopes: ["https://www.googleapis.com/auth/drive"],
          auth_params: { access_type: "offline", prompt: "consent" },
        },
      })
    );
    const cfg = loadProvidersConfig({ BROKER_PROVIDERS_FILE: file });
    expect(cfg.gdrive.client_id).toBe("g-id");
    expect(cfg.gdrive.scopes).toEqual(["https://www.googleapis.com/auth/drive"]);
    rmSync(dir, { recursive: true, force: true });
  });

  it("BROKER_PROVIDERS_JSON env merges and env wins on overlap", () => {
    const env = {
      DROPBOX_CLIENT_ID: "from-env-id",
      DROPBOX_CLIENT_SECRET: "from-env-sec",
      BROKER_PROVIDERS_JSON: JSON.stringify({
        dropbox: { client_id: "from-json-id", client_secret: "from-json-sec", authorize_url: "https://custom.example/auth" },
      }),
    };
    const cfg = loadProvidersConfig(env);
    // Shorthand DROPBOX_* merges last and wins
    expect(cfg.dropbox.client_id).toBe("from-env-id");
  });

  it("builtin defaults apply to dropbox", () => {
    expect(BUILTIN_PROVIDERS.dropbox.authorize_url).toBe("https://www.dropbox.com/oauth2/authorize");
    expect(BUILTIN_PROVIDERS.dropbox.token_url).toBe("https://api.dropboxapi.com/oauth2/token");
    const cfg = loadProvidersConfig({ DROPBOX_CLIENT_ID: "id", DROPBOX_CLIENT_SECRET: "sec" });
    const prov = getProviderOrThrow(cfg, "dropbox");
    expect(prov.authorize_url).toBe("https://www.dropbox.com/oauth2/authorize");
    expect(prov.token_url).toBe("https://api.dropboxapi.com/oauth2/token");
    expect(prov.auth_params?.token_access_type).toBe("offline");
  });

  it("unknown provider must supply authorize_url/token_url", () => {
    const cfg = loadProvidersConfig({
      BROKER_PROVIDERS_JSON: JSON.stringify({
        custom: { client_id: "id", client_secret: "sec" },
      }),
    });
    expect(() => getProviderOrThrow(cfg, "custom")).toThrow(/missing authorize_url/);
  });

  it("missing file is ignored (no throw)", () => {
    const cfg = loadProvidersConfig({ BROKER_PROVIDERS_FILE: "/nonexistent/path.json" });
    expect(Object.keys(cfg)).toHaveLength(0);
  });

  it("malformed JSON is ignored", () => {
    const cfg = loadProvidersConfig({ BROKER_PROVIDERS_JSON: "{not-json" });
    expect(Object.keys(cfg)).toHaveLength(0);
  });
});
