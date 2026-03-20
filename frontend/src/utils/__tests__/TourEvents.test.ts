import { beforeEach, describe, expect, it, spyOn } from "bun:test";
import "../../../test/setup";
import { TourEvents, TourEventTypes } from "../TourEvents";

describe("TourEvents", () => {
    beforeEach(() => {
        TourEvents.clearListeners();
    });

    it("emits payload to listeners and returns unsubscribe", async () => {
        const payload = document.createElement("div");
        let received: Element | null = null;

        const unsubscribe = TourEvents.on(TourEventTypes.USERS_STEP_3, (element) => {
            received = element;
        });

        await TourEvents.emit(TourEventTypes.USERS_STEP_3, payload);

        expect(received === payload).toBe(true);
        expect(typeof unsubscribe).toBe("function");
    });

    it("supports once listeners that fire only one time", async () => {
        let calls = 0;
        const payload = document.createElement("span");

        TourEvents.once(TourEventTypes.SHARES_STEP_3, () => {
            calls += 1;
        });

        await TourEvents.emit(TourEventTypes.SHARES_STEP_3, payload);
        await TourEvents.emit(TourEventTypes.SHARES_STEP_3, payload);

        expect(calls).toBe(1);
    });

    it("removes listeners via off", async () => {
        let calls = 0;
        const listener = () => {
            calls += 1;
        };

        TourEvents.on(TourEventTypes.SETTINGS_STEP_5, listener);
        TourEvents.off(TourEventTypes.SETTINGS_STEP_5, listener);

        await TourEvents.emit(TourEventTypes.SETTINGS_STEP_5, document.createElement("div"));

        expect(calls).toBe(0);
    });

    it("clears listeners for a single event", async () => {
        let usersCalls = 0;
        let sharesCalls = 0;

        TourEvents.on(TourEventTypes.USERS_STEP_3, () => {
            usersCalls += 1;
        });
        TourEvents.on(TourEventTypes.SHARES_STEP_4, () => {
            sharesCalls += 1;
        });

        TourEvents.clearListeners(TourEventTypes.USERS_STEP_3);

        await TourEvents.emit(TourEventTypes.USERS_STEP_3, document.createElement("div"));
        await TourEvents.emit(TourEventTypes.SHARES_STEP_4, document.createElement("div"));

        expect(usersCalls).toBe(0);
        expect(sharesCalls).toBe(1);
    });

    it("does not throw when a listener errors", async () => {
        const warnSpy = spyOn(console, "warn").mockImplementation(() => undefined);

        TourEvents.on(TourEventTypes.VOLUMES_STEP_3, () => {
            throw new Error("boom");
        });

        await TourEvents.emit(TourEventTypes.VOLUMES_STEP_3, document.createElement("div"));

        expect(warnSpy).toHaveBeenCalled();
        warnSpy.mockRestore();
    });
});
