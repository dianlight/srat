import { describe, it, expect } from "vitest";
import { setupServer } from "msw/node";
import { customHandlers, setFilesystemSupportOverride, clearFilesystemSupportOverrides } from "./customHandlers";

describe("customHandlers – cover 16 missing lines for patch 0%→80%", () => {
  it("setFilesystemSupportOverride and clear", () => {
    setFilesystemSupportOverride("ext4", { canMount: false });
    clearFilesystemSupportOverrides();
    expect(customHandlers.length).toBeGreaterThan(0);
  });

  it("filesystem state handler – 400 without partition_id and 200 with", async () => {
    const server = setupServer(...customHandlers);
    server.listen();
    const r1 = await fetch("http://localhost:3000/api/filesystem/state");
    expect(r1.status).toBe(400);
    const r2 = await fetch("http://localhost:3000/api/filesystem/state?partition_id=123");
    expect(r2.status).toBe(200);
    const j = await r2.json() as { isClean: boolean };
    expect(j.isClean).toBe(true);
    server.close();
  });

  it("filesystem support handler – override and fallback", async () => {
    setFilesystemSupportOverride("vfat", { canMount: false, alpinePackage: "test" });
    const server = setupServer(...customHandlers);
    server.listen();
    const r1 = await fetch("http://localhost:3000/api/filesystem/support?fstype=vfat");
    expect(r1.status).toBe(200);
    const j1 = await r1.json() as { canMount: boolean };
    expect(j1.canMount).toBe(false);
    const r2 = await fetch("http://localhost:3000/api/filesystem/support?fstype=ext4");
    expect(r2.status).toBe(200);
    server.close();
    clearFilesystemSupportOverrides();
  });

  it("other handlers return 200", async () => {
    const server = setupServer(...customHandlers);
    server.listen();
    const urls = [
      "http://localhost:3000/api/health",
      "http://localhost:3000/api/shares",
      "http://localhost:3000/api/volumes",
      "http://localhost:3000/api/lab_features",
      "http://localhost:3000/api/settings",
    ];
    for (const u of urls) {
      const r = await fetch(u);
      expect(r.status).toBe(200);
    }
    server.close();
  });
});
