import { afterEach, beforeEach, describe, expect, it, mock } from "bun:test";
import "../../../../../test/setup";

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
        const { render, waitFor, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../../../test/setup");
        // @ts-expect-error - Query param loads isolated module instance
        const { SharesTreeView } = await import("../SharesTreeView?shares-tree-test");

        const onSelect = mock(() => { });
        const store = await createTestStore();

        const { getByText } = render(
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
        await waitFor(() => {
            expect(getByText("Documents")).toBeTruthy();
        });
        const documentsNode = getByText("Documents");
        await user.click(documentsNode);
        expect(onSelect).toHaveBeenCalledWith("doc", expect.objectContaining({ name: "Documents" }));

        // Helper function to click an action (handles both desktop buttons and mobile menu)
        const clickShareAction = async (actionName: RegExp, shareText: string) => {
            // Find the share node first to ensure we're targeting the right share
            await waitFor(() => {
                expect(getByText(shareText)).toBeTruthy();
            });
            const shareNode = getByText(shareText);
            const shareContainer = shareNode.closest('[role="treeitem"]');
            if (!shareContainer) return;

            const shareScope = within(shareContainer as HTMLElement);

            // First try to find direct action buttons (desktop view) within this share
            const directButtons = shareScope.queryAllByRole("button", { name: actionName });
            if (directButtons.length > 0) {
                await user.click(directButtons[0]!);
                return;
            }
            // If no direct buttons, try to find the menu button within this share
            const menuButtons = shareScope.queryAllByRole("button", { name: /more actions/i });
            if (menuButtons.length > 0) {
                await user.click(menuButtons[0]!);
                // Wait for menu to open - search in document.body since Menu uses Portal
                const portalQueries = within(document.body);
                await waitFor(() => {
                    expect(
                        portalQueries.getByRole("menuitem", { name: actionName }),
                    ).toBeTruthy();
                });
                const menuItem = portalQueries.getByRole("menuitem", {
                    name: actionName,
                });
                await user.click(menuItem);
            }
        };

        // Disable the Documents share (which is enabled)
        await clickShareAction(/disable share/i, "Documents");

        // Enable the Archive share (which is disabled)
        await clickShareAction(/enable share/i, "Archive");

        await waitFor(() => expect(tracking.disableCalls.length).toBeGreaterThanOrEqual(1));
        await waitFor(() => expect(tracking.enableCalls.length).toBeGreaterThanOrEqual(1));
    });

    it("hides non-internal shares while in protected mode", async () => {
        const { overrides } = setupOverrides();

        const React = await import("react");
        const { render, within } = await import("@testing-library/react");
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
        const { render, waitFor, within } = await import(
            "@testing-library/react",
        );
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

        // Helper function to click an action (handles both desktop buttons and mobile menu)
        const clickShareAction = async (actionName: RegExp) => {
            // First try to find direct action buttons (desktop view)
            const directButtons = within(container).queryAllByRole("button", { name: actionName });
            if (directButtons.length > 0) {
                await user.click(directButtons[0]!);
                return;
            }
            // If no direct buttons, try to find the menu button and click the menu item
            const menuButtons = within(container).queryAllByRole("button", { name: /more actions/i });
            if (menuButtons.length > 0) {
                await user.click(menuButtons[0]!);
                // Wait for menu to open - search in document.body since Menu uses Portal
                const portalQueries = within(document.body);
                await waitFor(() => {
                    expect(
                        portalQueries.getByRole("menuitem", { name: actionName }),
                    ).toBeTruthy();
                });
                const menuItem = portalQueries.getByRole("menuitem", {
                    name: actionName,
                });
                await user.click(menuItem);
            }
        };

        await clickShareAction(/disable share/i);

        expect(tracking.disableCalls.length).toBe(0);
    });

    it("hides toggle controls when readOnly is enabled", async () => {
        const { overrides } = setupOverrides();

        const React = await import("react");
        const { render, within } = await import("@testing-library/react");
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
