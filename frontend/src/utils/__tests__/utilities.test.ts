import { describe, it, expect } from "bun:test";

describe("TourEvents utility", () => {
    it("exports TourEvents class", async () => {
        const { TourEvents } = await import("../TourEvents");
        expect(TourEvents).toBeTruthy();
    });

    it("exports TourEventTypes enum", async () => {
        const { TourEventTypes } = await import("../TourEvents");
        expect(TourEventTypes).toBeTruthy();
    });

    it("can create TourEvents instance", async () => {
        const { TourEvents } = await import("../TourEvents");
        const instance = new TourEvents();
        expect(instance).toBeTruthy();
    });

    it("can emit tour events", async () => {
        const { TourEvents, TourEventTypes } = await import("../TourEvents");
        const instance = new TourEvents();
        
        // This should not throw
        instance.emit(TourEventTypes.DASHBOARD_STEP_1);
        expect(instance).toBeTruthy();
    });

    it("can listen to tour events", async () => {
        const { TourEvents, TourEventTypes } = await import("../TourEvents");
        const instance = new TourEvents();
        
        let eventReceived = false;
        instance.on(TourEventTypes.DASHBOARD_STEP_1, () => {
            eventReceived = true;
        });
        
        instance.emit(TourEventTypes.DASHBOARD_STEP_1);
        expect(eventReceived).toBeTruthy();
    });

    it("can remove event listeners", async () => {
        const { TourEvents, TourEventTypes } = await import("../TourEvents");
        const instance = new TourEvents();
        
        let callCount = 0;
        const handler = () => { callCount++; };
        
        instance.on(TourEventTypes.DASHBOARD_STEP_1, handler);
        instance.emit(TourEventTypes.DASHBOARD_STEP_1);
        
        instance.off(TourEventTypes.DASHBOARD_STEP_1, handler);
        instance.emit(TourEventTypes.DASHBOARD_STEP_1);
        
        // Should only be called once (before removal)
        expect(callCount).toBe(1);
    });

    it("supports multiple event listeners", async () => {
        const { TourEvents, TourEventTypes } = await import("../TourEvents");
        const instance = new TourEvents();
        
        let count1 = 0;
        let count2 = 0;
        
        instance.on(TourEventTypes.DASHBOARD_STEP_1, () => { count1++; });
        instance.on(TourEventTypes.DASHBOARD_STEP_1, () => { count2++; });
        
        instance.emit(TourEventTypes.DASHBOARD_STEP_1);
        
        expect(count1).toBe(1);
        expect(count2).toBe(1);
    });

    it("has all expected tour event types", async () => {
        const { TourEventTypes } = await import("../TourEvents");
        
        // Check that common tour event types exist
        expect(TourEventTypes.DASHBOARD_STEP_1).toBeTruthy();
        expect(TourEventTypes.SHARES_STEP_1).toBeTruthy();
        expect(TourEventTypes.VOLUMES_STEP_1).toBeTruthy();
        expect(TourEventTypes.SETTINGS_STEP_1).toBeTruthy();
        expect(TourEventTypes.USERS_STEP_1).toBeTruthy();
    });
});

describe("volumes utils", () => {
    it("exports decodeEscapeSequence function", async () => {
        const { decodeEscapeSequence } = await import("../../pages/volumes/utils");
        expect(typeof decodeEscapeSequence).toBe("function");
    });

    it("exports generateSHA1Hash function", async () => {
        const { generateSHA1Hash } = await import("../../pages/volumes/utils");
        expect(typeof generateSHA1Hash).toBe("function");
    });

    it("decodeEscapeSequence handles normal strings", async () => {
        const { decodeEscapeSequence } = await import("../../pages/volumes/utils");
        const result = decodeEscapeSequence("test");
        expect(result).toBe("test");
    });

    it("decodeEscapeSequence handles escaped characters", async () => {
        const { decodeEscapeSequence } = await import("../../pages/volumes/utils");
        const result = decodeEscapeSequence("test\\x20space");
        expect(result).toBeTruthy();
    });

    it("generateSHA1Hash generates hash from string", async () => {
        const { generateSHA1Hash } = await import("../../pages/volumes/utils");
        const hash = generateSHA1Hash("test");
        expect(hash).toBeTruthy();
        expect(typeof hash).toBe("string");
        expect(hash.length).toBe(40); // SHA1 hash is 40 characters
    });

    it("generateSHA1Hash generates different hashes for different inputs", async () => {
        const { generateSHA1Hash } = await import("../../pages/volumes/utils");
        const hash1 = generateSHA1Hash("test1");
        const hash2 = generateSHA1Hash("test2");
        expect(hash1).not.toBe(hash2);
    });

    it("generateSHA1Hash generates same hash for same input", async () => {
        const { generateSHA1Hash } = await import("../../pages/volumes/utils");
        const hash1 = generateSHA1Hash("test");
        const hash2 = generateSHA1Hash("test");
        expect(hash1).toBe(hash2);
    });
});

describe("shares utils", () => {
    it("exports utility functions", async () => {
        const utils = await import("../../pages/shares/utils");
        expect(utils).toBeTruthy();
    });
});

describe("location state", () => {
    it("exports TabIDs enum", async () => {
        const { TabIDs } = await import("../../store/locationState");
        expect(TabIDs).toBeTruthy();
        expect(TabIDs.DASHBOARD).toBeTruthy();
        expect(TabIDs.VOLUMES).toBeTruthy();
        expect(TabIDs.SHARES).toBeTruthy();
        expect(TabIDs.USERS).toBeTruthy();
        expect(TabIDs.SETTINGS).toBeTruthy();
    });

    it("exports LocationState type", async () => {
        const locationState = await import("../../store/locationState");
        expect(locationState).toBeTruthy();
    });
});

describe("error slice", () => {
    it("exports error slice", async () => {
        const errorSlice = await import("../../store/errorSlice");
        expect(errorSlice.default).toBeTruthy();
    });

    it("error slice has setError action", async () => {
        const { setError } = await import("../../store/errorSlice");
        expect(typeof setError).toBe("function");
    });

    it("error slice has clearError action", async () => {
        const { clearError } = await import("../../store/errorSlice");
        expect(typeof clearError).toBe("function");
    });

    it("can dispatch setError action", async () => {
        const { setError } = await import("../../store/errorSlice");
        const action = setError("Test error");
        expect(action.type).toBe("error/setError");
        expect(action.payload).toBe("Test error");
    });

    it("can dispatch clearError action", async () => {
        const { clearError } = await import("../../store/errorSlice");
        const action = clearError();
        expect(action.type).toBe("error/clearError");
    });
});

describe("store configuration", () => {
    it("exports store", async () => {
        const { default: store } = await import("../../store/store");
        expect(store).toBeTruthy();
    });

    it("store has getState method", async () => {
        const { default: store } = await import("../../store/store");
        expect(typeof store.getState).toBe("function");
    });

    it("store has dispatch method", async () => {
        const { default: store } = await import("../../store/store");
        expect(typeof store.dispatch).toBe("function");
    });

    it("store has subscribe method", async () => {
        const { default: store } = await import("../../store/store");
        expect(typeof store.subscribe).toBe("function");
    });

    it("can get initial state", async () => {
        const { default: store } = await import("../../store/store");
        const state = store.getState();
        expect(state).toBeTruthy();
    });
});
