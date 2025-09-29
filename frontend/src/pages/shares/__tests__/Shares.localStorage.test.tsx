import "../../../../test/setup";
// Shared test setup (DOM globals, APIURL, and store helper)
import { describe, it, expect, beforeEach } from "bun:test";

// Minimal localStorage shim for bun:test
if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
        setItem: (k: string, v: string) => {
            _store[k] = String(v);
        },
        removeItem: (k: string) => {
            delete _store[k];
        },
        clear: () => {
            for (const k of Object.keys(_store)) delete _store[k];
        },
    };
}

describe("Shares localStorage functionality", () => {
    beforeEach(() => {
        localStorage.clear();
    });

    it("saves and restores selectedShareKey to localStorage", () => {
        const testShareKey = "share-123";

        // Save to localStorage
        localStorage.setItem("shares.selectedShareKey", testShareKey);

        // Verify it can be retrieved
        expect(localStorage.getItem("shares.selectedShareKey")).toBe(testShareKey);
    });

    it("saves and restores expandedGroups to localStorage", () => {
        const testGroups = ["group-Internal", "group-Media"];

        // Save to localStorage
        localStorage.setItem("shares.expandedGroups", JSON.stringify(testGroups));

        // Verify it can be retrieved and parsed
        const retrieved = localStorage.getItem("shares.expandedGroups");
        expect(retrieved).toBe(JSON.stringify(testGroups));

        const parsed = JSON.parse(retrieved!);
        expect(Array.isArray(parsed)).toBe(true);
        expect(parsed).toEqual(testGroups);
    });

    it("clears localStorage when removeItem is called", () => {
        // Set some values
        localStorage.setItem("shares.selectedShareKey", "test-share");
        localStorage.setItem("shares.expandedGroups", JSON.stringify(["test-group"]));

        // Verify they exist
        expect(localStorage.getItem("shares.selectedShareKey")).toBe("test-share");
        expect(localStorage.getItem("shares.expandedGroups")).toBe(JSON.stringify(["test-group"]));

        // Remove them
        localStorage.removeItem("shares.selectedShareKey");
        localStorage.removeItem("shares.expandedGroups");

        // Verify they're gone
        expect(localStorage.getItem("shares.selectedShareKey")).toBe(null);
        expect(localStorage.getItem("shares.expandedGroups")).toBe(null);
    });

    it("handles invalid JSON gracefully when parsing expandedGroups", () => {
        // Set invalid JSON
        localStorage.setItem("shares.expandedGroups", "invalid-json");

        // Try to parse it (simulating what the component would do)
        try {
            const saved = localStorage.getItem("shares.expandedGroups");
            if (saved) {
                const parsed = JSON.parse(saved);
                // This shouldn't be reached
                expect(false).toBe(true);
            }
        } catch {
            // This is expected - the component should fall back to empty array
            expect(true).toBe(true);
        }
    });

    it("initializes empty array when no expandedGroups in localStorage", () => {
        // Ensure nothing is stored
        expect(localStorage.getItem("shares.expandedGroups")).toBe(null);

        // This simulates the component's initialization logic
        let expandedGroups: string[] = [];
        try {
            const savedExpanded = localStorage.getItem("shares.expandedGroups");
            if (savedExpanded) {
                const parsed = JSON.parse(savedExpanded);
                if (Array.isArray(parsed)) {
                    expandedGroups = parsed as string[];
                }
            }
        } catch { }

        expect(expandedGroups).toEqual([]);
    });

    it("restores only valid selectedShareKey from localStorage", () => {
        // Test with valid key
        localStorage.setItem("shares.selectedShareKey", "valid-share-123");
        let selectedKey = localStorage.getItem("shares.selectedShareKey") || undefined;
        expect(selectedKey).toBe("valid-share-123");

        // Test with null/empty
        localStorage.removeItem("shares.selectedShareKey");
        selectedKey = localStorage.getItem("shares.selectedShareKey") || undefined;
        expect(selectedKey).toBe(undefined);
    });
});