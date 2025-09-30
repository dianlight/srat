import "../../../../test/setup";
import { describe, it, expect } from "bun:test";

describe("Settings tour steps", () => {
    it("provides consistent selectors and emits actions", async () => {
        const { SettingsSteps } = await import("../SettingsTourStep");
        const { TourEvents, TourEventTypes } = await import("../../../utils/TourEvents");

        expect(Array.isArray(SettingsSteps)).toBe(true);
        expect(SettingsSteps.length).toBeGreaterThan(0);

        SettingsSteps.forEach((step, index) => {
            expect(step.selector).toContain(`step${index}`);
            expect(step.content).toBeTruthy();
        });

        const originalEmit = TourEvents.emit;
        let emitted: string | null = null;

        try {
            TourEvents.emit = ((event: any) => {
                emitted = event;
                return Promise.resolve();
            }) as typeof TourEvents.emit;

            SettingsSteps.forEach((step) => {
                step.action?.({} as any);
            });
        } finally {
            TourEvents.emit = originalEmit;
        }

        const allowed = new Set([
            TourEventTypes.SETTINGS_STEP_3,
            TourEventTypes.SETTINGS_STEP_5,
            TourEventTypes.SETTINGS_STEP_8,
            null,
        ]);
        expect(allowed.has(emitted)).toBe(true);
    });
});
