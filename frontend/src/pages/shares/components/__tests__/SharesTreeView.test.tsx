import "../../../../../test/setup";
import { describe, it, expect, beforeEach, mock } from "bun:test";

describe("SharesTreeView component", () => {
    beforeEach(() => {
        mock.restore();
    });

    const setupOverrides = (options?: { confirmResult?: unknown }) => {
        const disableCalls: Array<string> = [];
        const enableCalls: Array<string> = [];
        const confirmCalls: Array<any> = [];

        return {
            tracking: {
                disableCalls,
                enableCalls,
                confirmCalls,
            },
            overrides: {
                disableShare: async ({ shareName }: { shareName: string }) => {
                    disableCalls.push(shareName);
                },
                enableShare: async ({ shareName }: { shareName: string }) => {
                    enableCalls.push(shareName);
                },
                confirm: async (confirmOptions: unknown) => {
                    confirmCalls.push(confirmOptions);
                    return options?.confirmResult ?? { confirmed: true };
                },
            },
        } as const;
    };

    it("allows selecting and toggling shares", async () => {
        const { overrides, tracking } = setupOverrides();

        const React = await import("react");
        const { render, screen, fireEvent, waitFor } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../../../test/setup");
        // @ts-expect-error - Query param loads isolated module instance
        const { SharesTreeView } = await import("../SharesTreeView?shares-tree-test");

        const onSelect = mock(() => { });
        const store = await createTestStore();

        render(
            React.createElement(
                Provider as any,
                { store },
                React.createElement(SharesTreeView as any, {
                    shares: {
                        doc: {
                            name: "Documents",
                            usage: "none",
                            mount_point_data: { warnings: undefined },
                            disabled: false,
                        },
                        arc: {
                            name: "Archive",
                            usage: "internal",
                            mount_point_data: {},
                            disabled: true,
                        },
                    },
                    expandedItems: ["group-none", "group-internal"],
                    onExpandedItemsChange: () => { },
                    selectedShareKey: "doc",
                    onShareSelect: onSelect,
                    testOverrides: overrides,
                })
            )
        );

        const documentsNode = await screen.findByText("Documents");
        fireEvent.click(documentsNode);
        expect(onSelect).toHaveBeenCalledWith("doc", expect.objectContaining({ name: "Documents" }));

        const firstToggle = await screen.findByTestId("share-toggle-doc");
        fireEvent.click(firstToggle);

        const archiveToggle = await screen.findByTestId("share-toggle-arc");
        fireEvent.click(archiveToggle);

        await waitFor(() => expect(tracking.disableCalls.length).toBeGreaterThanOrEqual(1));
        await waitFor(() => expect(tracking.enableCalls.length).toBeGreaterThanOrEqual(1));
    });

    it("hides non-internal shares while in protected mode", async () => {
        const { overrides } = setupOverrides();

        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../../../test/setup");
        // @ts-expect-error - Query param loads isolated module instance
        const { SharesTreeView } = await import("../SharesTreeView?shares-tree-protected");
        const store = await createTestStore();

        render(
            React.createElement(
                Provider as any,
                { store },
                React.createElement(SharesTreeView as any, {
                    shares: {
                        doc: {
                            name: "Documents",
                            usage: "none",
                            mount_point_data: {},
                            disabled: false,
                        },
                        sys: {
                            name: "System",
                            usage: "internal",
                            mount_point_data: {},
                            disabled: false,
                        },
                    },
                    expandedItems: ["group-internal"],
                    onExpandedItemsChange: () => { },
                    selectedShareKey: undefined,
                    onShareSelect: () => { },
                    protectedMode: true,
                    testOverrides: overrides,
                })
            )
        );

        expect(await screen.findByRole("tree")).toBeTruthy();
        expect(screen.queryByText("Documents")).toBeNull();
    });

    it("does not disable share when confirmation is declined", async () => {
        const { overrides, tracking } = setupOverrides({ confirmResult: { confirmed: false } });

        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../../../test/setup");
        // @ts-expect-error - Query param loads isolated module instance
        const { SharesTreeView } = await import("../SharesTreeView?shares-tree-cancel-toggle");
        const store = await createTestStore();

        render(
            React.createElement(
                Provider as any,
                { store },
                React.createElement(SharesTreeView as any, {
                    shares: {
                        doc: {
                            name: "Documents",
                            usage: "none",
                            mount_point_data: {},
                            disabled: false,
                        },
                    },
                    expandedItems: ["group-none"],
                    onExpandedItemsChange: () => { },
                    selectedShareKey: "doc",
                    onShareSelect: () => { },
                    testOverrides: overrides,
                })
            )
        );

        const toggle = await screen.findByTestId("share-toggle-doc");
        fireEvent.click(toggle);

        expect(tracking.disableCalls.length).toBe(0);
    });

    it("hides toggle controls when readOnly is enabled", async () => {
        const { overrides } = setupOverrides();

        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../../../test/setup");
        // @ts-expect-error - Query param loads isolated module instance
        const { SharesTreeView } = await import("../SharesTreeView?shares-tree-readonly");
        const store = await createTestStore();

        render(
            React.createElement(
                Provider as any,
                { store },
                React.createElement(SharesTreeView as any, {
                    shares: {
                        doc: {
                            name: "Documents",
                            usage: "none",
                            mount_point_data: { invalid: true, invalid_error: "bad" },
                            disabled: false,
                        },
                    },
                    expandedItems: ["group-none"],
                    onExpandedItemsChange: () => { },
                    selectedShareKey: undefined,
                    onShareSelect: () => { },
                    readOnly: true,
                    testOverrides: overrides,
                })
            )
        );

        expect(await screen.findByText(/Documents/)).toBeTruthy();
        expect(screen.queryByTestId("share-toggle-doc")).toBeNull();
    });
});
