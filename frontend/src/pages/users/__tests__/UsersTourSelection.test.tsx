import { describe, expect, it } from "bun:test";
import "../../../../test/setup";

describe("Users tour selection", () => {
    it("prefers a non-admin user for guided tour selection", async () => {
        const { getTourTargetUser } = await import("../Users");

        const target = getTourTargetUser([
            { username: "admin", is_admin: true } as any,
            { username: "alice", is_admin: false } as any,
        ]);

        expect(target?.username).toBe("alice");
    });
});
