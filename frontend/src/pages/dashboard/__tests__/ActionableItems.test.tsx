import { render, screen, within } from "@testing-library/react";
import React from "react";
import { Provider } from "react-redux";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it } from "vitest";
import { createTestStore } from "/test/testing";
import { ActionableItemsList } from "../components/ActionableItemsList";

// Minimal localStorage shim for bun:test
if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
        setItem: (k: string, v: string) => { _store[k] = String(v); },
        removeItem: (k: string) => { delete _store[k]; },
        clear: () => { for (const k of Object.keys(_store)) delete _store[k]; },
    };
}

const renderActionableItems = async (props: Record<string, unknown>) => {
    const store = await createTestStore();

    return render(
        React.createElement(
            Provider,
            {
                store,
                children: React.createElement(
                    MemoryRouter,
                    null,
                    React.createElement(ActionableItemsList as any, props)
                ),
            },
        )
    );
};

describe("Dashboard Actionable Items", () => {
    beforeEach(() => {
        localStorage.clear();
    });

    it("renders actionable items list with mount action", async () => {
        const mockPartitions = [
            {
                partition: {
                    id: "part-1",
                    name: "by-uuid-0560-3E7B",
                    size: 125830000000,
                    mount_status: false,
                    system: false,
                },
                action: "mount" as const,
            },
        ];

        const { container } = await renderActionableItems({
            actionablePartitions: mockPartitions,
            isLoading: false,
            error: null,
            showIgnored: false,
        });

        // Check that partition name appears
        const partitionName = await within(container).findByText("by-uuid-0560-3E7B");
        expect(partitionName).toBeTruthy();

        // Check that Mount button appears
        const mountButton = await within(container).findByText("Mount");
        expect(mountButton).toBeTruthy();
    });

    it("renders actionable items list with share action", async () => {
        const mockPartitions = [
            {
                partition: {
                    id: "part-2",
                    name: "EFI",
                    size: 210000000,
                    mount_status: true,
                    system: false,
                },
                action: "share" as const,
            },
        ];

        const { container } = await renderActionableItems({
            actionablePartitions: mockPartitions,
            isLoading: false,
            error: null,
            showIgnored: false,
        });

        // Check that partition name appears
        const partitionName = await within(container).findByText("EFI");
        expect(partitionName).toBeTruthy();

        // Check that Create Share button appears
        const shareButton = await within(container).findByText("Create Share");
        expect(shareButton).toBeTruthy();
    });

    it("shows hide buttons for actionable items", async () => {
        const mockPartitions = [
            {
                partition: {
                    id: "part-1",
                    name: "test-partition",
                    size: 1000000000,
                    mount_status: false,
                    system: false,
                },
                action: "mount" as const,
            },
        ];

        const { container } = await renderActionableItems({
            actionablePartitions: mockPartitions,
            isLoading: false,
            error: null,
            showIgnored: false,
        });

        // Check that Hide button appears
        const hideButtons = within(container).getAllByText("Hide");
        expect(hideButtons.length).toBeGreaterThan(0);
    });

    it("shows loading state", async () => {
        await renderActionableItems({
            actionablePartitions: [],
            isLoading: true,
            error: null,
            showIgnored: false,
        });

        // Check that loading indicator appears using semantic query
        const loadingElement = screen.getByRole("progressbar");
        expect(loadingElement).toBeTruthy();
    });

    it("shows error state", async () => {
        const mockError = new Error("Failed to load volumes");

        const { container } = await renderActionableItems({
            actionablePartitions: [],
            isLoading: false,
            error: mockError,
            showIgnored: false,
        });

        // Check that error message appears - use getAllByText since there may be multiple instances
        const errorMessages = within(container).getAllByText("Could not load volume information.");
        expect(errorMessages.length).toBeGreaterThan(0);
    });

    it("shows empty state when no actionable items", async () => {
        const { container } = await renderActionableItems({
            actionablePartitions: [],
            isLoading: false,
            error: null,
            showIgnored: false,
        });

        // Check that empty message appears
        const emptyMessage = await within(container).findByText("No actionable items at the moment.");
        expect(emptyMessage).toBeTruthy();
    });
});