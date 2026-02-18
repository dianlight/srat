/*
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

type EventListener = (event: any) => void;

type ListenerMap = Map<string, Set<EventListener>>;

class MockEventSource {
    static instances: MockEventSource[] = [];
    url: string;
    withCredentials?: boolean;
    listeners: ListenerMap = new Map();
    closed = false;

    constructor(url: string, options?: { withCredentials?: boolean }) {
        this.url = url;
        this.withCredentials = options?.withCredentials;
        MockEventSource.instances.push(this);
    }

    addEventListener(type: string, listener: EventListener) {
        const set = this.listeners.get(type) ?? new Set<EventListener>();
        set.add(listener);
        this.listeners.set(type, set);
    }

    removeEventListener(type: string, listener: EventListener) {
        const set = this.listeners.get(type);
        if (set) set.delete(listener);
    }

    emit(type: string, data?: any) {
        const set = this.listeners.get(type);
        if (!set) return;
        for (const listener of set) {
            listener(data ?? { data: "{}" });
        }
    }

    close() {
        this.closed = true;
    }
}

class MockWebSocket {
    static instances: MockWebSocket[] = [];
    url: string;
    listeners: ListenerMap = new Map();
    closed = false;

    constructor(url: string) {
        this.url = url;
        MockWebSocket.instances.push(this);
    }

    addEventListener(type: string, listener: EventListener) {
        const set = this.listeners.get(type) ?? new Set<EventListener>();
        set.add(listener);
        this.listeners.set(type, set);
    }

    removeEventListener(type: string, listener: EventListener) {
        const set = this.listeners.get(type);
        if (set) set.delete(listener);
    }

    emit(type: string, data?: any) {
        const set = this.listeners.get(type);
        if (!set) return;
        for (const listener of set) {
            listener(data ?? {});
        }
    }

    close() {
        this.closed = true;
        this.emit("close", {});
    }
}

describe("sseApi reconnect behavior", () => {
    const originalEventSource = (globalThis as any).EventSource;
    const originalWebSocket = (globalThis as any).WebSocket;

    beforeEach(() => {
        MockEventSource.instances = [];
        MockWebSocket.instances = [];
        document.body.innerHTML = "";
        (globalThis as any).EventSource = MockEventSource as any;
        (globalThis as any).WebSocket = MockWebSocket as any;
        (globalThis as any).__SRAT_SSE_RECONNECT_MS = 0;
        (globalThis as any).__SRAT_WS_RECONNECT_MS = 0;
    });

    afterEach(() => {
        (globalThis as any).EventSource = originalEventSource;
        (globalThis as any).WebSocket = originalWebSocket;
        delete (globalThis as any).__SRAT_SSE_INACTIVITY_MS;
        delete (globalThis as any).__SRAT_WS_INACTIVITY_MS;
        delete (globalThis as any).__SRAT_SSE_RECONNECT_MS;
        delete (globalThis as any).__SRAT_WS_RECONNECT_MS;
    });

    it("reconnects SSE on error", async () => {
        const store = await createTestStore();
        const { sseApi } = await import("../../store/sseApi");

        const subscription = store.dispatch(
            sseApi.endpoints.getServerEvents.initiate(),
        );
        await subscription;

        expect(MockEventSource.instances.length).toBe(1);
        const firstEventSource = MockEventSource.instances[0];
        expect(firstEventSource).toBeTruthy();
        firstEventSource?.emit("open", {});
        firstEventSource?.emit("error", {});

        const reconnected = await waitForCondition(
            () => MockEventSource.instances.length >= 2,
        );
        expect(reconnected).toBe(true);
        expect(MockEventSource.instances.length).toBeGreaterThanOrEqual(2);

        subscription.unsubscribe();
    });

    it("reconnects SSE after inactivity", async () => {
        (globalThis as any).__SRAT_SSE_INACTIVITY_MS = 10;

        const store = await createTestStore();
        const { sseApi } = await import("../../store/sseApi");

        const subscription = store.dispatch(
            sseApi.endpoints.getServerEvents.initiate(),
        );
        await subscription;

        expect(MockEventSource.instances.length).toBe(1);
        const firstEventSource = MockEventSource.instances[0];
        expect(firstEventSource).toBeTruthy();
        firstEventSource?.emit("open", {});

        const reconnected = await waitForCondition(
            () => MockEventSource.instances.length >= 2,
        );
        expect(reconnected).toBe(true);
        expect(MockEventSource.instances.length).toBeGreaterThanOrEqual(2);

        subscription.unsubscribe();
    });

    it("reconnects WebSocket on close", async () => {
        const store = await createTestStore();
        const { wsApi } = await import("../../store/sseApi");

        const subscription = store.dispatch(
            wsApi.endpoints.getServerEvents.initiate(),
        );
        await subscription;

        expect(MockWebSocket.instances.length).toBe(1);
        const firstSocket = MockWebSocket.instances[0];
        expect(firstSocket).toBeTruthy();
        firstSocket?.emit("open", {});
        firstSocket?.emit("close", {});

        const reconnected = await waitForCondition(
            () => MockWebSocket.instances.length >= 2,
        );
        expect(reconnected).toBe(true);
        expect(MockWebSocket.instances.length).toBeGreaterThanOrEqual(2);

        subscription.unsubscribe();
    });

    it("reconnects WebSocket after inactivity", async () => {
        (globalThis as any).__SRAT_WS_INACTIVITY_MS = 10;

        const store = await createTestStore();
        const { wsApi } = await import("../../store/sseApi");

        const subscription = store.dispatch(
            wsApi.endpoints.getServerEvents.initiate(),
        );
        await subscription;

        expect(MockWebSocket.instances.length).toBe(1);
        const firstSocket = MockWebSocket.instances[0];
        expect(firstSocket).toBeTruthy();
        firstSocket?.emit("open", {});

        const reconnected = await waitForCondition(
            () => MockWebSocket.instances.length >= 2,
        );
        expect(reconnected).toBe(true);
        expect(MockWebSocket.instances.length).toBeGreaterThanOrEqual(2);

        subscription.unsubscribe();
    });

    it("reports isLoading when WebSocket is disconnected", async () => {
        const store = await createTestStore();
        const React = await import("react");
        const { render, screen, act } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { useGetServerEventsQuery } = await import("../../store/sseApi");

        const TestComponent = () => {
            const { isLoading } = useGetServerEventsQuery();
            return React.createElement("div", {}, isLoading ? "loading" : "ready");
        };

        render(
            React.createElement(Provider, {
                store,
                children: React.createElement(TestComponent),
            }),
        );

        const loadingInitial = await screen.findByText("loading");
        expect(loadingInitial).toBeTruthy();

        const wsCreated = await waitForCondition(
            () => MockWebSocket.instances.length === 1,
        );
        expect(wsCreated).toBe(true);

        const socket = MockWebSocket.instances[0];
        if (!socket) {
            throw new Error("Expected WebSocket instance to exist");
        }
        await act(async () => {
            socket.emit("open", {});
        });

        const ready = await screen.findByText("ready");
        expect(ready).toBeTruthy();

        await act(async () => {
            socket.emit("close", {});
        });

        const loadingAgain = await screen.findByText("loading");
        expect(loadingAgain).toBeTruthy();
    });
});
*/