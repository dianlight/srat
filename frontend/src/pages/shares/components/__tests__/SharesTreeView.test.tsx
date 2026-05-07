import { render, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Provider } from "react-redux";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createTestStore } from "/test/testing";
import { Type, Usage } from "../../../../store/sratApi";
import { SharesTreeView } from "../SharesTreeView";

describe("SharesTreeView component", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  const setupOverrides = (options?: { confirmResult?: unknown }) => {
    const disableCalls: Array<string> = [];
    const enableCalls: Array<string> = [];
    const confirmCalls: Array<unknown> = [];

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
          if ((result as { confirmed?: boolean })?.confirmed === false) {
            throw result;
          }
          return result;
        },
      },
    } as const;
  };

  const mountPointData = (overrides: Record<string, unknown> = {}) => ({
    path: "/mnt/test",
    type: Type.Host,
    ...overrides,
  });

  it("allows selecting and toggling shares", async () => {
    const { overrides, tracking } = setupOverrides();
    const onSelect = vi.fn(() => {});
    const store = await createTestStore();

    const { getByLabelText } = render(
      <Provider store={store}>
        <SharesTreeView
          shares={{
            doc: {
              name: "Documents",
              usage: Usage.None,
              mount_point_data: mountPointData({ warnings: undefined }),
              disabled: false,
            },
            arc: {
              name: "Archive",
              usage: Usage.Internal,
              mount_point_data: mountPointData(),
              disabled: true,
            },
          }}
          expandedItems={["group-none", "group-internal"]}
          onExpandedItemsChange={() => {}}
          selectedShareKey="doc"
          onShareSelect={onSelect}
          testOverrides={overrides}
        />
      </Provider>,
    );

    const user = userEvent.setup();
    await waitFor(() => {
      expect(getByLabelText("Documents")).toBeTruthy();
    });

    const documentsNode = getByLabelText("Documents");
    await user.click(documentsNode);
    expect(onSelect).toHaveBeenCalledWith(
      "doc",
      expect.objectContaining({ name: "Documents" }),
    );

    const clickShareAction = async (actionName: RegExp, shareText: string) => {
      const shareNode = getByLabelText(shareText);
      const shareContainer = shareNode.closest('[role="treeitem"]');
      if (!shareContainer) return;

      const shareScope = within(shareContainer as HTMLElement);
      const directButtons = shareScope.queryAllByRole("button", {
        name: actionName,
      });
      if (directButtons.length > 0) {
        await user.click(directButtons[0]!);
        return;
      }

      const menuButtons = shareScope.queryAllByRole("button", {
        name: /more actions/i,
      });
      if (menuButtons.length > 0) {
        await user.click(menuButtons[0]!);
        const portalQueries = within(document.body);
        const menuItem = await waitFor(() =>
          portalQueries.getByRole("menuitem", { name: actionName }),
        );
        await user.click(menuItem);
      }
    };

    await clickShareAction(/disable share/i, "Documents");
    await clickShareAction(/enable share/i, "Archive");

    await waitFor(() => {
      expect(tracking.disableCalls.length).toBeGreaterThanOrEqual(1);
    });
    await waitFor(() => {
      expect(tracking.enableCalls.length).toBeGreaterThanOrEqual(1);
    });
  });

  it("hides non-internal shares while in protected mode", async () => {
    const { overrides } = setupOverrides();
    const store = await createTestStore();

    const { container } = render(
      <Provider store={store}>
        <SharesTreeView
          shares={{
            doc: {
              name: "Documents",
              usage: Usage.None,
              mount_point_data: mountPointData(),
              disabled: false,
            },
            sys: {
              name: "System",
              usage: Usage.Internal,
              mount_point_data: mountPointData({ path: "/mnt/system" }),
              disabled: false,
            },
          }}
          expandedItems={["group-internal"]}
          onExpandedItemsChange={() => {}}
          selectedShareKey={undefined}
          onShareSelect={() => {}}
          protectedMode
          testOverrides={overrides}
        />
      </Provider>,
    );

    const trees = within(container).queryAllByRole("tree");
    expect(trees).toHaveLength(1);
    expect(within(container).queryByText("Documents")).toBeNull();
  });

  it("does not disable share when confirmation is declined", async () => {
    const { overrides, tracking } = setupOverrides({
      confirmResult: { confirmed: false },
    });
    const store = await createTestStore();

    const { container } = render(
      <Provider store={store}>
        <SharesTreeView
          shares={{
            doc: {
              name: "Documents",
              usage: Usage.None,
              mount_point_data: mountPointData(),
              disabled: false,
            },
          }}
          expandedItems={["group-none"]}
          onExpandedItemsChange={() => {}}
          selectedShareKey="doc"
          onShareSelect={() => {}}
          testOverrides={overrides}
        />
      </Provider>,
    );

    const user = userEvent.setup();
    const directButtons = within(container).queryAllByRole("button", {
      name: /disable share/i,
    });

    if (directButtons.length > 0) {
      await user.click(directButtons[0]!);
    } else {
      const menuButton = within(container).getByRole("button", {
        name: /more actions/i,
      });
      await user.click(menuButton);
      const menuItem = await within(document.body).findByRole("menuitem", {
        name: /disable share/i,
      });
      await user.click(menuItem);
    }

    expect(tracking.disableCalls.length).toBe(0);
  });

  it("hides toggle controls when readOnly is enabled", async () => {
    const { overrides } = setupOverrides();
    const store = await createTestStore();

    const { container } = render(
      <Provider store={store}>
        <SharesTreeView
          shares={{
            doc: {
              name: "Documents",
              usage: Usage.None,
              mount_point_data: mountPointData({
                invalid: true,
                invalid_error: "bad",
              }),
              disabled: false,
            },
          }}
          expandedItems={["group-none"]}
          onExpandedItemsChange={() => {}}
          selectedShareKey={undefined}
          onShareSelect={() => {}}
          readOnly
          testOverrides={overrides}
        />
      </Provider>,
    );

    expect(await within(container).findAllByLabelText("Documents")).toHaveLength(
      1,
    );
    expect(
      within(container).queryByRole("button", { name: /disable share/i }),
    ).toBeNull();
    expect(
      within(container).queryByRole("button", { name: /enable share/i }),
    ).toBeNull();
  });

  it("shows full share name in tooltip on hover", async () => {
    const { overrides } = setupOverrides();
    const store = await createTestStore();

    const longShareName =
      "This is a very long share name to verify tooltip shows the complete value";

    render(
      <Provider store={store}>
        <SharesTreeView
          shares={{
            long: {
              name: longShareName,
              usage: Usage.None,
              mount_point_data: mountPointData({ path: "/mnt/long" }),
              disabled: false,
            },
          }}
          expandedItems={["group-none"]}
          onExpandedItemsChange={() => {}}
          selectedShareKey={undefined}
          onShareSelect={() => {}}
          testOverrides={overrides}
        />
      </Provider>,
    );

    const user = userEvent.setup();
    await user.hover(within(document.body).getByText(longShareName));

    const tooltip = await within(document.body).findByRole("tooltip");
    expect(tooltip.textContent).toBe(longShareName);
  });
});
