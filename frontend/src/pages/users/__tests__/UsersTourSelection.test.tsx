import { describe, expect, it } from "vitest";

describe("Users tour selection", () => {
    it("prefers a non-admin user for guided tour selection", async () => {
        const { getTourTargetUser } = await import("../tourSelection");

        const target = getTourTargetUser([
            { username: "admin", is_admin: true } as any,
            { username: "alice", is_admin: false } as any,
        ]);

        expect(target?.username).toBe("alice");
    });
});
