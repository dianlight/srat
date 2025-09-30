import "../../../../test/setup";
import { describe, it, expect } from "bun:test";

describe("Shares tour steps", () => {
    it("includes expected structure and actions", async () => {
        const { SharesSteps } = await import("../SharesTourStep");
        const { TourEvents, TourEventTypes } = await import("../../../utils/TourEvents");

        expect(Array.isArray(SharesSteps)).toBe(true);
        expect(SharesSteps.length).toBeGreaterThan(0);

        SharesSteps.forEach((step, index) => {
            expect(step.selector).toContain(`step${index}`);
            expect(step.content).toBeTruthy();
        });

        const originalEmit = TourEvents.emit;
        const seenEvents: string[] = [];

        try {
            TourEvents.emit = ((event: any) => {
                seenEvents.push(event);
                return Promise.resolve();
            }) as typeof TourEvents.emit;

            SharesSteps.forEach((step) => step.action?.({} as any));
        } finally {
            TourEvents.emit = originalEmit;
        }

        const allowed = new Set<string>([
            TourEventTypes.SHARES_STEP_3,
            TourEventTypes.SHARES_STEP_4,
        ]);

        expect(seenEvents.every((evt) => allowed.has(evt))).toBe(true);
    });
});
