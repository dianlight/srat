/**
 * Simple smoke test to verify MSW is properly integrated
 */

import "../testing"; // Ensure setup runs first
import { describe, it, expect } from "vitest";

describe("MSW Setup Smoke Test", () => {
	it("MSW server setup module loads", async () => {
		const { getMswServer } = await import("../testing");
		const server = await getMswServer();
		expect(server).toBeDefined();
		expect(server.listHandlers).toBeDefined();
	});

	it("has handlers registered", async () => {
		const { getMswServer } = await import("../testing");
		const server = await getMswServer();
		const handlers = server.listHandlers();
		expect(handlers.length).toBeGreaterThan(0);
	});
});
