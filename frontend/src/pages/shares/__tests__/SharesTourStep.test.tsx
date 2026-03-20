import { describe, expect, it } from "bun:test";
import "../../../../test/setup";

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

    it("picks the first available share for guided tour actions", async () => {
        const { getTourTargetShare } = await import("../Shares");

        const target = getTourTargetShare({
            media: { name: "Media", usage: "general" } as any,
            backups: { name: "Backups", usage: "backup" } as any,
        });

        expect(target?.[0]).toBe("media");
        expect(target?.[1].name).toBe("Media");
    });
});
