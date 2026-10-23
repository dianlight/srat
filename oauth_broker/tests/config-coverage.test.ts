import { describe, it, expect, vi } from "vitest";
import { getSessionTtlSeconds, isValidBrokerPublicUrl, loadProvidersConfig, getInstanceTtlSeconds, isAllowedSratCallbackUrl, isProductionEnv } from "../src/config.js";

describe("config.ts – patch 69% → 85% (17 miss +14 partials)", () => {
  it("getSessionTtlSeconds – unparseable and out of range branches (lines 40-49)", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    expect(getSessionTtlSeconds({ SESSION_TTL: "abc" })).toBe(600);
    expect(warn).toHaveBeenCalledWith(expect.stringContaining("unparseable"));
    warn.mockClear();
    expect(getSessionTtlSeconds({ SESSION_TTL: "0" })).toBe(60);
    expect(warn).toHaveBeenCalledWith(expect.stringContaining("out of range"));
    warn.mockClear();
    expect(getSessionTtlSeconds({ SESSION_TTL: "-5" })).toBe(60);
    expect(getSessionTtlSeconds({ SESSION_TTL: "10m" })).toBe(600);
    expect(getSessionTtlSeconds({ SESSION_TTL: "600s" })).toBe(600);
    expect(getSessionTtlSeconds({ SESSION_TTL: "  900  " })).toBe(900);
    warn.mockRestore();
  });

  it("isValidBrokerPublicUrl – false on invalid and missing", () => {
    expect(isValidBrokerPublicUrl("")).toBe(false);
    expect(isValidBrokerPublicUrl("not-a-url")).toBe(false);
    expect(isValidBrokerPublicUrl("https://example.com")).toBe(true);
    expect(isValidBrokerPublicUrl("http://localhost:3000")).toBe(true);
    expect(isValidBrokerPublicUrl("http://example.com")).toBe(false);
  });

  it("loadProvidersConfig – malformed BROKER_PROVIDERS_FILE JSON warn (line 130)", async () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const { writeFileSync, unlinkSync } = await import("node:fs");
    const tmp = "/tmp/bad-providers.json";
    writeFileSync(tmp, "{not-json", "utf-8");
    const cfg = loadProvidersConfig({ BROKER_PROVIDERS_FILE: tmp, BROKER_PROVIDERS_JSON: "{bad-json", DROPBOX_CLIENT_ID: "id", DROPBOX_CLIENT_SECRET: "sec" });
    expect(cfg.dropbox?.client_id).toBe("id");
    expect(warn).toHaveBeenCalled();
    try { unlinkSync(tmp); } catch {}
    // Also test inline JSON malformed
    const cfg2 = loadProvidersConfig({ BROKER_PROVIDERS_JSON: "{not-json" });
    expect(cfg2).toEqual({});
    warn.mockRestore();
  });

  it("getInstanceTtlSeconds – 1h, 60m, 3600s, plain, unparseable", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    expect(getInstanceTtlSeconds({ INSTANCE_TTL: "1h" })).toBe(3600);
    expect(getInstanceTtlSeconds({ INSTANCE_TTL: "60m" })).toBe(3600);
    expect(getInstanceTtlSeconds({ INSTANCE_TTL: "3600s" })).toBe(3600);
    expect(getInstanceTtlSeconds({ INSTANCE_TTL: "3600" })).toBe(3600);
    expect(getInstanceTtlSeconds({ INSTANCE_TTL: "abc" })).toBe(3600);
    expect(warn).toHaveBeenCalledWith(expect.stringContaining("unparseable"));
    warn.mockRestore();
    expect(getInstanceTtlSeconds({})).toBe(3600);
  });

  it("isAllowedSratCallbackUrl – invalid URL catch (line 265)", () => {
    expect(isAllowedSratCallbackUrl("not-a-url", "https://allowed.example.com/*")).toBe(false);
    expect(isAllowedSratCallbackUrl("https://allowed.example.com/cb", "https://allowed.example.com/*")).toBe(true);
    expect(isAllowedSratCallbackUrl("https://other.example.com/cb", "https://allowed.example.com/*")).toBe(false);
  });

  it("isProductionEnv", () => {
    expect(isProductionEnv({ BROKER_PUBLIC_URL: "https://production.example.com" })).toBe(true);
    expect(isProductionEnv({ ENV: "production" })).toBe(true);
    expect(isProductionEnv({ WORKERS_ENV: "production" })).toBe(true);
    expect(isProductionEnv({ BROKER_PUBLIC_URL: "https://dev.example.com" })).toBe(false);
  });
});
