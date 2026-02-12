// Shared test setup (DOM globals, APIURL, and store helper)
import { afterEach, beforeEach, describe, expect, it } from "bun:test";
import "../../../../test/setup";
import { createTestStore } from "../../../../test/setup";

// Minimal localStorage shim for bun:test
if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
        setItem: (k: string, v: string) => {
            _store[k] = String(v);
        },
        removeItem: (k: string) => {
            delete _store[k];
        },
        clear: () => {
            for (const k of Object.keys(_store)) delete _store[k];
        },
    };
}

describe("Volumes restored selection", () => {
    beforeEach(() => {
        localStorage.clear();
    });

    afterEach(async () => {
        const { cleanup } = await import("@testing-library/react");
        cleanup();
    });

    it("restores selected partition and shows details", async () => {
        // pre-populate saved selected partition id
        localStorage.setItem("volumes.selectedPartitionId", "part-42");

        // Dynamically import React/testing utilities and the component after globals are set
        const React = await import("react");
        const { render, within } = await import("@testing-library/react");
        const { MemoryRouter } = await import("react-router");
        const { Volumes } = await import("../Volumes");
        const { Provider } = await import("react-redux");
        const store = await createTestStore();

        const initialDisks = [
            {
                id: "disk-1",
                model: "TestDisk",
                partitions: [
                    {
                        id: "part-42",
                        name: "RestoredPartition",
                        size: 1024,
                        mount_point_data: [],
                    },
                ],
            },
        ];

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        MemoryRouter,
                        null,
                        React.createElement(Volumes as any, { initialDisks }),
                    ),
                },
            ),
        );

        // The VolumeDetailsPanel displays the partition name; wait for it to appear
        const el = await within(container).findByText("RestoredPartition");
        expect(el).toBeTruthy();
    });
});
