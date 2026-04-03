import { describe, expect, it } from "bun:test";
import { http, HttpResponse } from "msw";
import { getMswServer } from "../../../../../test/bun-setup";
import "../../../../../test/setup";

async function renderWithProviders(
  element: any,
  options?: { seedStore?: (store: any) => void },
) {
  const React = await import("react");
  const { render } = await import("@testing-library/react");
  const { Provider } = await import("react-redux");
  const { createTestStore } = await import("../../../../../test/setup");

  const store = await createTestStore();
  if (options?.seedStore) {
    options.seedStore(store);
  }

  const result = render(
    React.createElement(Provider, { store, children: element }),
  );
  return { ...result, store };
}

describe("Filesystem label/format dialogs", () => {
  it("disables Set Label when label tools are unavailable", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const { FilesystemLabelDialog } = await import("../FilesystemLabelDialog");

    const partition = {
      id: "part-label-1",
      name: "data",
      device_path: "/dev/sdb1",
      filesystem_info: {
        support: {
          canSetLabel: false,
        },
      },
    };

    await renderWithProviders(
      React.createElement(FilesystemLabelDialog as any, {
        open: true,
        partition,
        onClose: () => {},
      }),
    );

    const button = await screen.findByRole("button", { name: /set label/i });
    expect((button as HTMLButtonElement).disabled).toBe(true);

    const hint = await screen.findByText(/Label tools are not available/i);
    expect(hint).toBeTruthy();
  });

  it("disables Format when format tools are unavailable", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const { FilesystemFormatDialog } = await import("../FilesystemFormatDialog");

    const partition = {
      id: "part-format-1",
      name: "archive",
      device_path: "/dev/sdc1",
      fs_type: "ext4",
      filesystem_info: {
        support: {
          canFormat: false,
        },
      },
    };

    const server = await getMswServer();
    server.use(
      http.get("/api/filesystem/support", () => {
        return HttpResponse.json({
          canMount: true,
          canFormat: false,
          canCheck: true,
          canSetLabel: true,
          canGetState: true,
          alpinePackage: "e2fsprogs",
          missingTools: ["mkfs.ext4"],
        });
      }),
    );

    await renderWithProviders(
      React.createElement(FilesystemFormatDialog as any, {
        open: true,
        partition,
        onClose: () => {},
      }),
    );

    const button = await screen.findByRole("button", { name: /format/i });
    expect((button as HTMLButtonElement).disabled).toBe(true);

    const hint = await screen.findByText(/Format tools are not available/i);
    expect(hint).toBeTruthy();
  });
});
