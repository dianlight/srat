import React from "react";
import { cleanup, render, within } from "@testing-library/react";
import { Provider } from "react-redux";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { createTestStore } from "/test/testing";
import { getDiskIdentifier, getPartitionIdentifier } from "../utils";
import { Volumes } from "../Volumes";

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

    afterEach(() => {
        cleanup();
    });

    it("restores selected partition and shows details", async () => {
        const store = await createTestStore();

        const initialDisks = [
            {
                id: "disk-1",
                model: "TestDisk",
                partitions: {
                    "part-42": {
                        id: "part-42",
                        name: "RestoredPartition",
                        size: 1024,
                        mount_point_data: {},
                    },
                },
            },
        ];

        const diskIdentifier = getDiskIdentifier(initialDisks[0] as any, 0);
        const partitionKey = "part-42";
        const partitionIdentifier = getPartitionIdentifier(
            diskIdentifier,
            (initialDisks[0].partitions as any)[partitionKey],
            partitionKey,
            0,
        );

        // pre-populate saved selected partition id
        localStorage.setItem("volumes.selectedPartitionId", partitionIdentifier);

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
