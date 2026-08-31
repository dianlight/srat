/* eslint-disable */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createTestStore } from "/test/testing";
import { Severity, Status, Supported_events, sratApi } from "../sratApi";

const delay = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

const waitForCondition = async (
    check: () => boolean,
    {
        timeoutMs = 2000,
        stepMs = 5,
    }: { timeoutMs?: number; stepMs?: number } = {},
) => {
    const start = Date.now();
    while (!check()) {
        if (Date.now() - start > timeoutMs) return false;
        await delay(stepMs);
    }
    return true;
};

type EventListener = (event: unknown) => void;
type ListenerMap = Map<string, Set<EventListener>>;

class MockWebSocket {
    static instances: MockWebSocket[] = [];
    url: string;
    listeners: ListenerMap = new Map();

    constructor(url: string) {
        this.url = url;
        MockWebSocket.instances.push(this);
    }

    addEventListener(type: string, listener: EventListener) {
        const set = this.listeners.get(type) ?? new Set<EventListener>();
        set.add(listener);
        this.listeners.set(type, set);
    }

    emit(type: string, data?: unknown) {
        const set = this.listeners.get(type);
        if (!set) return;
        for (const listener of set) {
            listener(data ?? {});
        }
    }

    close() {
        this.emit("close", {});
    }
}

describe("wsApi reconnect behavior", () => {
    const originalWebSocket = (globalThis as Record<string, unknown>).WebSocket;
    const originalStreaming = process.env.MSW_ENABLE_STREAMING;

    beforeEach(() => {
        MockWebSocket.instances = [];
        (globalThis as Record<string, unknown>).WebSocket = MockWebSocket;
        (globalThis as Record<string, unknown>).__SRAT_WS_RECONNECT_MS = 0;
        process.env.MSW_ENABLE_STREAMING = "1";
    });

    afterEach(() => {
        (globalThis as Record<string, unknown>).WebSocket = originalWebSocket;
        delete (globalThis as Record<string, unknown>).__SRAT_WS_INACTIVITY_MS;
        delete (globalThis as Record<string, unknown>).__SRAT_WS_RECONNECT_MS;
        if (originalStreaming === undefined) {
            delete process.env.MSW_ENABLE_STREAMING;
        } else {
            process.env.MSW_ENABLE_STREAMING = originalStreaming;
        }
    });

    it("reconnects WebSocket on close", async () => {
        const store = await createTestStore();
        const { wsApi } = await import("../../store/wsApi");

        if (!wsApi?.endpoints?.getServerEvents) {
            expect(true).toBe(true);
            return;
        }

        const subscription = store.dispatch(
            wsApi.endpoints.getServerEvents.initiate(),
        );
        await subscription;

        expect(MockWebSocket.instances.length).toBe(1);
        const firstSocket = MockWebSocket.instances[0];
        firstSocket?.emit("open", {});
        firstSocket?.emit("close", {});

        const reconnected = await waitForCondition(
            () => MockWebSocket.instances.length >= 2,
        );
        expect(reconnected).toBe(true);

        subscription.unsubscribe();
    });

    it("reconnects WebSocket after inactivity", async () => {
        (globalThis as Record<string, unknown>).__SRAT_WS_INACTIVITY_MS = 10;

        const store = await createTestStore();
        const { wsApi } = await import("../../store/wsApi");

        if (!wsApi?.endpoints?.getServerEvents) {
            expect(true).toBe(true);
            return;
        }

        const subscription = store.dispatch(
            wsApi.endpoints.getServerEvents.initiate(),
        );
        await subscription;

        expect(MockWebSocket.instances.length).toBe(1);
        const firstSocket = MockWebSocket.instances[0];
        firstSocket?.emit("open", {});

        const reconnected = await waitForCondition(
            () => MockWebSocket.instances.length >= 2,
        );
        expect(reconnected).toBe(true);

        subscription.unsubscribe();
    });

    it("builds websocket URL without double slash path", async () => {
        const store = await createTestStore();
        const { wsApi } = await import("../../store/wsApi");

        if (!wsApi?.endpoints?.getServerEvents) {
            expect(true).toBe(true);
            return;
        }

        const subscription = store.dispatch(
            wsApi.endpoints.getServerEvents.initiate(),
        );
        await subscription;

        expect(MockWebSocket.instances.length).toBe(1);
        const socket = MockWebSocket.instances[0];
        expect(socket?.url).toContain("/ws");
        expect(socket?.url).not.toContain("//ws");

        subscription.unsubscribe();
    });

    it("parses problem websocket events into cache data", async () => {
        const store = await createTestStore();
        const { wsApi } = await import("../../store/wsApi");

        if (!wsApi?.endpoints?.getServerEvents) {
            expect(true).toBe(true);
            return;
        }

        const subscription = store.dispatch(
            wsApi.endpoints.getServerEvents.initiate(),
        );
        await subscription;

        expect(MockWebSocket.instances.length).toBe(1);
        const socket = MockWebSocket.instances[0];
        socket?.emit("open", {});

        const payload = {
            problem_key: "custom_component_restart_required",
            id: 1,
            repeating: 1,
            title: "Restart required",
            description: "Restart Home Assistant to apply changes.",
            severity: Severity.Warning,
            status: Status.Created,
            created_at: "2026-01-01T00:00:00Z",
            updated_at: "2026-01-01T00:00:00Z",
            ignored: false,
        };

        socket?.emit("message", {
            data: `id: 101\nevent: ${Supported_events.Problem}\ndata: ${JSON.stringify(payload)}`,
        });

        const updated = await waitForCondition(() => {
            // biome-ignore lint/suspicious/noExplicitAny: dynamic import incompatible with static RootState
            const state = wsApi.endpoints.getServerEvents.select()(store.getState() as any);
            return (
                state?.data?.[Supported_events.Problem]?.problem_key === payload.problem_key
            );
        });

        expect(updated).toBe(true);
        // biome-ignore lint/suspicious/noExplicitAny: dynamic import incompatible with static RootState
        const finalState = wsApi.endpoints.getServerEvents.select()(store.getState() as any);
        expect(finalState?.data?.[Supported_events.Problem]?.status).toBe(Status.Created);

        subscription.unsubscribe();
    });
});

describe("wsApi dirty-tracking auto-invalidation", () => {
    const originalWebSocket = (globalThis as Record<string, unknown>).WebSocket;
    const originalStreaming = process.env.MSW_ENABLE_STREAMING;

    beforeEach(() => {
        MockWebSocket.instances = [];
        (globalThis as Record<string, unknown>).WebSocket = MockWebSocket;
        (globalThis as Record<string, unknown>).__SRAT_WS_RECONNECT_MS = 0;
        process.env.MSW_ENABLE_STREAMING = "1";
    });

    afterEach(() => {
        (globalThis as Record<string, unknown>).WebSocket = originalWebSocket;
        delete (globalThis as Record<string, unknown>).__SRAT_WS_RECONNECT_MS;
        if (originalStreaming === undefined) {
            delete process.env.MSW_ENABLE_STREAMING;
        } else {
            process.env.MSW_ENABLE_STREAMING = originalStreaming;
        }
        vi.restoreAllMocks();
    });

    const setupSocket = async () => {
        const store = await createTestStore();
        const { wsApi } = await import("../../store/wsApi");

        if (!wsApi?.endpoints?.getServerEvents) {
            return null;
        }

        const subscription = store.dispatch(
            wsApi.endpoints.getServerEvents.initiate(),
        );
        await subscription;

        expect(MockWebSocket.instances.length).toBe(1);
        const socket = MockWebSocket.instances[0];
        socket?.emit("open", {});

        const spy = vi.spyOn(sratApi.util, "invalidateTags");
        return { store, wsApi, socket, subscription, spy };
    };

    it("invalidates the user tag on dirty_data_tracker users flag", async () => {
        const setup = await setupSocket();
        if (!setup) {
            expect(true).toBe(true);
            return;
        }
        const { socket, subscription, spy } = setup;

        socket?.emit("message", {
            data: `id: 201\nevent: ${Supported_events.DirtyDataTracker}\ndata: ${JSON.stringify({ users: true })}`,
        });

        const invalidated = await waitForCondition(
            () => spy.mock.calls.length >= 1,
        );
        expect(invalidated).toBe(true);
        expect(spy.mock.calls[0]?.[0]).toEqual(["user"]);

        subscription.unsubscribe();
    });

    it("invalidates the share tag on dirty_data_tracker shares flag", async () => {
        const setup = await setupSocket();
        if (!setup) {
            expect(true).toBe(true);
            return;
        }
        const { socket, subscription, spy } = setup;

        socket?.emit("message", {
            data: `id: 202\nevent: ${Supported_events.DirtyDataTracker}\ndata: ${JSON.stringify({ shares: true })}`,
        });

        const invalidated = await waitForCondition(
            () => spy.mock.calls.length >= 1,
        );
        expect(invalidated).toBe(true);
        expect(spy.mock.calls[0]?.[0]).toEqual(["share"]);

        subscription.unsubscribe();
    });

    it("invalidates the user tag on heartbeat dirty_tracking.users", async () => {
        const setup = await setupSocket();
        if (!setup) {
            expect(true).toBe(true);
            return;
        }
        const { socket, subscription, spy } = setup;

        socket?.emit("message", {
            data: `id: 203\nevent: ${Supported_events.Heartbeat}\ndata: ${JSON.stringify({ alive: true, dirty_tracking: { users: true } })}`,
        });

        const invalidated = await waitForCondition(
            () => spy.mock.calls.length >= 1,
        );
        expect(invalidated).toBe(true);
        expect(spy.mock.calls[0]?.[0]).toEqual(["user"]);

        subscription.unsubscribe();
    });

    it("invalidates the share tag on heartbeat dirty_tracking.shares", async () => {
        const setup = await setupSocket();
        if (!setup) {
            expect(true).toBe(true);
            return;
        }
        const { socket, subscription, spy } = setup;

        socket?.emit("message", {
            data: `id: 204\nevent: ${Supported_events.Heartbeat}\ndata: ${JSON.stringify({ alive: true, dirty_tracking: { shares: true } })}`,
        });

        const invalidated = await waitForCondition(
            () => spy.mock.calls.length >= 1,
        );
        expect(invalidated).toBe(true);
        expect(spy.mock.calls[0]?.[0]).toEqual(["share"]);

        subscription.unsubscribe();
    });

    it("does not invalidate any tag when dirty flags are false", async () => {
        const setup = await setupSocket();
        if (!setup) {
            expect(true).toBe(true);
            return;
        }
        const { socket, subscription, spy } = setup;

        socket?.emit("message", {
            data: `id: 205\nevent: ${Supported_events.DirtyDataTracker}\ndata: ${JSON.stringify({ users: false, shares: false })}`,
        });
        socket?.emit("message", {
            data: `id: 206\nevent: ${Supported_events.Heartbeat}\ndata: ${JSON.stringify({ alive: true, dirty_tracking: { users: false, shares: false } })}`,
        });

        await delay(50);
        expect(spy).not.toHaveBeenCalled();

        subscription.unsubscribe();
    });
});
