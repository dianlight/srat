/**
 * Simple smoke test to verify MSW is properly integrated
 */

import "../setup"; // Ensure setup runs first
import { describe, it, expect } from "bun:test";

describe("MSW Setup Smoke Test", () => {
	it("MSW server setup module loads", async () => {
		const { getMswServer } = await import("../bun-setup");
		const server = await getMswServer();
		expect(server).toBeDefined();
		expect(server.listHandlers).toBeDefined();
	});

	it("has handlers registered", async () => {
		const { getMswServer } = await import("../bun-setup");
		const server = await getMswServer();
		const handlers = server.listHandlers();
		expect(handlers.length).toBeGreaterThan(0);
	});
});
