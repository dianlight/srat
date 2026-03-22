import { afterEach, beforeEach, describe, expect, it } from "bun:test";
import "../../../test/setup";
import { createTestStore } from "../../../test/setup";

const delay = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

const waitForCondition = async (
    check: () => boolean,
    {
        timeoutMs = 200,
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

    beforeEach(() => {
        MockWebSocket.instances = [];
        (globalThis as Record<string, unknown>).WebSocket = MockWebSocket;
        (globalThis as Record<string, unknown>).__SRAT_WS_RECONNECT_MS = 0;
    });

    afterEach(() => {
        (globalThis as Record<string, unknown>).WebSocket = originalWebSocket;
        delete (globalThis as Record<string, unknown>).__SRAT_WS_INACTIVITY_MS;
        delete (globalThis as Record<string, unknown>).__SRAT_WS_RECONNECT_MS;
    });

    it("reconnects WebSocket on close", async () => {
        const store = await createTestStore();
        const { wsApi } = await import("../../store/wsApi");

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
});
