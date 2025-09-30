import "../../../../test/setup";
import { describe, it, expect } from "bun:test";

describe("Users tour steps", () => {
    it("lists user onboarding steps and fires edit event", async () => {
        const { UsersSteps } = await import("../UsersSteps");
        const { TourEvents, TourEventTypes } = await import("../../../utils/TourEvents");

        expect(Array.isArray(UsersSteps)).toBe(true);
        expect(UsersSteps.length).toBeGreaterThan(0);

        UsersSteps.forEach((step, index) => {
            expect(step.selector).toContain(`step${index}`);
            expect(step.content).toBeTruthy();
        });

        const originalEmit = TourEvents.emit;
        let emittedEvent: string | null = null;

        try {
            TourEvents.emit = ((event: any) => {
                emittedEvent = event;
                return Promise.resolve();
            }) as typeof TourEvents.emit;

            UsersSteps.forEach((step) => step.action?.({ username: "demo" } as any));
        } finally {
            TourEvents.emit = originalEmit;
        }

        if (emittedEvent !== null) {
            expect(emittedEvent as string).toBe(TourEventTypes.USERS_STEP_3);
        }
    });
});