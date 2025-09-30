import "../../../../test/setup";
import { describe, it, expect } from "bun:test";

describe("Volumes tour steps", () => {
    it("exposes selectors and emits volume events", async () => {
        const { VolumesSteps } = await import("../VolumesTourStep");
        const { TourEvents, TourEventTypes } = await import("../../../utils/TourEvents");

        expect(Array.isArray(VolumesSteps)).toBe(true);
        expect(VolumesSteps.length).toBeGreaterThan(0);

        VolumesSteps.forEach((step, index) => {
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

            VolumesSteps.forEach((step) => step.action?.({} as any));
        } finally {
            TourEvents.emit = originalEmit;
        }

        const allowed = new Set<string>([
            TourEventTypes.VOLUMES_STEP_3,
            TourEventTypes.VOLUMES_STEP_4,
            TourEventTypes.VOLUMES_STEP_5,
        ]);

        expect(seenEvents.every((evt) => allowed.has(evt))).toBe(true);
    });
});
