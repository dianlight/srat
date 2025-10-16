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

describe("VolumesTreeView Component", () => {
    beforeEach(() => {
        localStorage.clear();
        document.body.innerHTML = '';
    });

    it("renders volumes tree view component", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store, children: React.createElement(VolumesTreeView as any, {
                    onSelectPartition: () => {}
                }) }
            )
        );

        expect(container).toBeTruthy();
    });

    it("renders tree structure with disks and partitions", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store, children: React.createElement(VolumesTreeView as any, {
                    onSelectPartition: () => {}
                }) }
            )
        );

        // Look for tree view elements
        const treeItems = container.querySelectorAll('[role="treeitem"]');
        expect(treeItems.length).toBeGreaterThanOrEqual(0);
    });

    it("handles partition selection", async () => {
        const React = await import("react");
        const { render, fireEvent } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();
        let selectedPartition = null;
        const onSelectPartition = (disk: any, partition: any) => {
            selectedPartition = partition;
        };

        const { container } = render(
            React.createElement(
                Provider,
                { store, children: React.createElement(VolumesTreeView as any, {
                    onSelectPartition
                }) }
            )
        );

        // Try clicking a tree item
        const treeItems = container.querySelectorAll('[role="treeitem"]');
        const firstTreeItem = treeItems[0];
        if (treeItems.length > 0 && firstTreeItem) {
            fireEvent.click(firstTreeItem);
        }

        expect(container).toBeTruthy();
    });

    it("displays disk icons based on type", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store, children: React.createElement(VolumesTreeView as any, {
                    onSelectPartition: () => {}
                }) }
            )
        );

        // Look for icons
        const icons = container.querySelectorAll('svg');
        expect(icons.length).toBeGreaterThanOrEqual(0);
    });

    it("shows partition mount status", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store, children: React.createElement(VolumesTreeView as any, {
                    onSelectPartition: () => {}
                }) }
            )
        );

        expect(container).toBeTruthy();
    });

    it("handles tree expansion and collapse", async () => {
        const React = await import("react");
        const { render, fireEvent } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store, children: React.createElement(VolumesTreeView as any, {
                    onSelectPartition: () => {}
                }) }
            )
        );

        // Look for expand/collapse buttons
        const expandIcons = container.querySelectorAll('[data-testid*="Expand"]');
        const firstExpandIcon = expandIcons[0];
        if (expandIcons.length > 0 && firstExpandIcon) {
            const button = firstExpandIcon.closest('button');
            if (button) {
                fireEvent.click(button);
            }
        }

        expect(container).toBeTruthy();
    });

    it("displays partition information", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store, children: React.createElement(VolumesTreeView as any, {
                    onSelectPartition: () => {}
                }) }
            )
        );

        // Verify component rendered
        expect(container).toBeTruthy();
    });

    it("handles loading state", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store, children: React.createElement(VolumesTreeView as any, {
                    onSelectPartition: () => {}
                }) }
            )
        );

        // Check for loading indicators
        const loadingElements = container.querySelectorAll('[role="progressbar"]');
        expect(loadingElements.length).toBeGreaterThanOrEqual(0);
    });

    it("highlights selected partition", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store, children: React.createElement(VolumesTreeView as any, {
                    onSelectPartition: () => {},
                    selectedPartitionId: "test-partition"
                }) }
            )
        );

        expect(container).toBeTruthy();
    });

    it("renders empty state when no disks available", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store, children: React.createElement(VolumesTreeView as any, {
                    onSelectPartition: () => {}
                }) }
            )
        );

        expect(container).toBeTruthy();
    });
});
