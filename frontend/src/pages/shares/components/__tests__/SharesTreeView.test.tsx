import "../../../../../test/setup";
import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";

describe("SharesTreeView component", () => {
    beforeEach(() => {
        mock.restore();
    });

    afterEach(async () => {
        mock.restore();
        const { cleanup } = await import("@testing-library/react");
        cleanup();
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
                    const result = options?.confirmResult ?? { confirmed: true };
                    // Simulate material-ui-confirm behavior: reject when user cancels
                    if ((result as any)?.confirmed === false) {
                        throw result;
                    }
                    return result;
                },
            },
        } as const;
    };

    it("allows selecting and toggling shares", async () => {
        const { overrides, tracking } = setupOverrides();

        const React = await import("react");
        const { render, screen, waitFor } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
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

        const user = userEvent.setup();
        const documentsNode = await screen.findByText("Documents");
        await user.click(documentsNode);
        expect(onSelect).toHaveBeenCalledWith("doc", expect.objectContaining({ name: "Documents" }));

        // Find disable button for Documents share (which is enabled)
        const disableButtons = await screen.findAllByRole("button", { name: /disable share/i });
        if (disableButtons.length > 0) {
            await user.click(disableButtons[0]!);
        }

        // Find enable button for Archive share (which is disabled)
        const enableButtons = await screen.findAllByRole("button", { name: /enable share/i });
        if (enableButtons.length > 0) {
            await user.click(enableButtons[0]!);
        }

        await waitFor(() => expect(tracking.disableCalls.length).toBeGreaterThanOrEqual(1));
        await waitFor(() => expect(tracking.enableCalls.length).toBeGreaterThanOrEqual(1));
    });

    it("hides non-internal shares while in protected mode", async () => {
        const { overrides } = setupOverrides();

        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../../../test/setup");
        // @ts-expect-error - Query param loads isolated module instance
        const { SharesTreeView } = await import("../SharesTreeView?shares-tree-protected");
        const store = await createTestStore();

        const { container } = render(
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

        const trees = within(container).queryAllByRole("tree");
        expect(trees).toHaveLength(1);
        expect(trees[0]).toBeTruthy();
        expect(within(container).queryByText("Documents")).toBeNull();
    });

    it("does not disable share when confirmation is declined", async () => {
        const { overrides, tracking } = setupOverrides({ confirmResult: { confirmed: false } });

        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../../../test/setup");
        // @ts-expect-error - Query param loads isolated module instance
        const { SharesTreeView } = await import("../SharesTreeView?shares-tree-cancel-toggle");
        const store = await createTestStore();

        const { container } = render(
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

        const user = userEvent.setup();
        const disableButtons = await within(container).findAllByRole("button", { name: /disable share/i });
        expect(disableButtons.length).toBeGreaterThan(0);
        await user.click(disableButtons[0]!);

        expect(tracking.disableCalls.length).toBe(0);
    });

    it("hides toggle controls when readOnly is enabled", async () => {
        const { overrides } = setupOverrides();

        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../../../test/setup");
        // @ts-expect-error - Query param loads isolated module instance
        const { SharesTreeView } = await import("../SharesTreeView?shares-tree-readonly");
        const store = await createTestStore();

        const { container } = render(
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

        const documents = await within(container).findAllByText(/Documents/);
        expect(documents).toHaveLength(1);
        expect(documents[0]).toBeTruthy();
        // In readOnly mode, ShareActions component should not be rendered
        expect(within(container).queryByRole("button", { name: /disable share/i })).toBeNull();
        expect(within(container).queryByRole("button", { name: /enable share/i })).toBeNull();
    });
});
