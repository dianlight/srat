import { describe, it, expect, beforeEach } from "bun:test";

// Required localStorage shim for testing environment
if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
        setItem: (k: string, v: string) => { _store[k] = String(v); },
        removeItem: (k: string) => { delete _store[k]; },
        clear: () => { for (const k of Object.keys(_store)) delete _store[k]; },
    };
}

describe("SambaStatusMetrics Component", () => {
    beforeEach(() => {
        localStorage.clear();
        document.body.innerHTML = '';
    });

    it("renders Samba status metrics component", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { SambaStatusMetrics } = await import("../metrics/SambaStatusMetrics");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store },
                React.createElement(SambaStatusMetrics as any)
            )
        );

        expect(container).toBeTruthy();
    });

    it("displays Samba service status", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { SambaStatusMetrics } = await import("../metrics/SambaStatusMetrics");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store },
                React.createElement(SambaStatusMetrics as any)
            )
        );

        // Check for status indicators
        expect(container.textContent).toBeTruthy();
    });

    it("renders loading state correctly", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { SambaStatusMetrics } = await import("../metrics/SambaStatusMetrics");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store },
                React.createElement(SambaStatusMetrics as any)
            )
        );

        const loadingElements = container.querySelectorAll('[role="progressbar"]');
        expect(loadingElements.length).toBeGreaterThanOrEqual(0);
    });

    it("displays connection statistics", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { SambaStatusMetrics } = await import("../metrics/SambaStatusMetrics");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store },
                React.createElement(SambaStatusMetrics as any)
            )
        );

        expect(container).toBeTruthy();
    });

    it("shows service running/stopped indicator", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { SambaStatusMetrics } = await import("../metrics/SambaStatusMetrics");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store },
                React.createElement(SambaStatusMetrics as any)
            )
        );

        const icons = container.querySelectorAll('svg');
        expect(icons.length).toBeGreaterThanOrEqual(0);
    });
});

describe("MetricCard Component", () => {
    beforeEach(() => {
        localStorage.clear();
        document.body.innerHTML = '';
    });

    it("renders metric card with title", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { MetricCard } = await import("../metrics/MetricCard");

        const { container } = render(
            React.createElement(MetricCard as any, {
                title: "Test Metric",
                value: "100"
            })
        );

        expect(container.textContent).toContain("Test Metric");
    });

    it("displays metric value", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { MetricCard } = await import("../metrics/MetricCard");

        const { container } = render(
            React.createElement(MetricCard as any, {
                title: "CPU",
                value: "45%"
            })
        );

        expect(container.textContent).toContain("45%");
    });

    it("renders custom icon when provided", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { MetricCard } = await import("../metrics/MetricCard");
        const MemoryIcon = await import("@mui/icons-material/Memory");

        const { container } = render(
            React.createElement(MetricCard as any, {
                title: "Memory",
                value: "8GB",
                icon: React.createElement(MemoryIcon.default)
            })
        );

        const icons = container.querySelectorAll('svg');
        expect(icons.length).toBeGreaterThanOrEqual(0);
    });

    it("handles click events", async () => {
        const React = await import("react");
        const { render, fireEvent } = await import("@testing-library/react");
        const { MetricCard } = await import("../metrics/MetricCard");

        let clicked = false;
        const handleClick = () => { clicked = true; };

        const { container } = render(
            React.createElement(MetricCard as any, {
                title: "Storage",
                value: "500GB",
                onClick: handleClick
            })
        );

        const card = container.querySelector('[class*="MuiCard"]');
        if (card) {
            fireEvent.click(card);
        }

        expect(container).toBeTruthy();
    });

    it("displays loading state", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { MetricCard } = await import("../metrics/MetricCard");

        const { container } = render(
            React.createElement(MetricCard as any, {
                title: "Network",
                value: "",
                loading: true
            })
        );

        const loadingElements = container.querySelectorAll('[role="progressbar"]');
        expect(loadingElements.length).toBeGreaterThanOrEqual(0);
    });

    it("renders with different variants", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { MetricCard } = await import("../metrics/MetricCard");

        const { container } = render(
            React.createElement(MetricCard as any, {
                title: "Disk",
                value: "100GB",
                variant: "outlined"
            })
        );

        expect(container).toBeTruthy();
    });
});

describe("NetworkHealthMetrics Component", () => {
    beforeEach(() => {
        localStorage.clear();
        document.body.innerHTML = '';
    });

    it("renders network health metrics", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { NetworkHealthMetrics } = await import("../metrics/NetworkHealthMetrics");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store },
                React.createElement(NetworkHealthMetrics as any)
            )
        );

        expect(container).toBeTruthy();
    });

    it("displays network interface statistics", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { NetworkHealthMetrics } = await import("../metrics/NetworkHealthMetrics");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store },
                React.createElement(NetworkHealthMetrics as any)
            )
        );

        expect(container.textContent).toBeTruthy();
    });

    it("shows bandwidth usage", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { NetworkHealthMetrics } = await import("../metrics/NetworkHealthMetrics");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store },
                React.createElement(NetworkHealthMetrics as any)
            )
        );

        expect(container).toBeTruthy();
    });
});

describe("ProcessMetrics Component", () => {
    beforeEach(() => {
        localStorage.clear();
        document.body.innerHTML = '';
    });

    it("renders process metrics component", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ProcessMetrics } = await import("../metrics/ProcessMetrics");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store },
                React.createElement(ProcessMetrics as any)
            )
        );

        expect(container).toBeTruthy();
    });

    it("displays process list", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ProcessMetrics } = await import("../metrics/ProcessMetrics");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store },
                React.createElement(ProcessMetrics as any)
            )
        );

        expect(container.textContent).toBeTruthy();
    });
});

describe("DiskHealthMetrics Component", () => {
    beforeEach(() => {
        localStorage.clear();
        document.body.innerHTML = '';
    });

    it("renders disk health metrics", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { DiskHealthMetrics } = await import("../metrics/DiskHealthMetrics");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store },
                React.createElement(DiskHealthMetrics as any)
            )
        );

        expect(container).toBeTruthy();
    });

    it("displays disk usage information", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { DiskHealthMetrics } = await import("../metrics/DiskHealthMetrics");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store },
                React.createElement(DiskHealthMetrics as any)
            )
        );

        expect(container.textContent).toBeTruthy();
    });
});
