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
  it("prefills Label field with current partition label on open", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const { FilesystemLabelDialog } = await import("../FilesystemLabelDialog");

    const partition = {
      id: "part-label-prefill-1",
      name: "media",
      device_path: "/dev/sde1",
    };

    await renderWithProviders(
      React.createElement(FilesystemLabelDialog as any, {
        open: true,
        partition,
        onClose: () => {},
      }),
    );

    const input = await screen.findByRole("textbox", { name: /label/i });
    expect((input as HTMLInputElement).value).toBe("media");
  });

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

    const hints = await screen.findAllByText(/Label tools are not available/i);
    expect(hints.length).toBeGreaterThan(0);
  });

  it("shows Set Label missing-tools install hint from support preflight", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const { FilesystemLabelDialog } = await import("../FilesystemLabelDialog");

    const partition = {
      id: "part-label-2",
      name: "backup",
      device_path: "/dev/sdd1",
      fs_type: "ext4",
      filesystem_info: {
        support: {
          canSetLabel: true,
        },
      },
    };

    const server = await getMswServer();
    server.use(
      http.get("/api/filesystem/support", () => {
        return HttpResponse.json({
          canMount: true,
          canFormat: true,
          canCheck: true,
          canSetLabel: false,
          canGetState: true,
          alpinePackage: "e2fsprogs",
          missingTools: ["e2label"],
        });
      }),
    );

    await renderWithProviders(
      React.createElement(FilesystemLabelDialog as any, {
        open: true,
        partition,
        onClose: () => {},
      }),
    );

    const button = await screen.findByRole("button", { name: /set label/i });
    expect((button as HTMLButtonElement).disabled).toBe(true);

    const missingTools = await screen.findAllByText(/Missing tools: e2label/i);
    expect(missingTools.length).toBeGreaterThan(0);
    const installHints = await screen.findAllByText(/apk add e2fsprogs/i);
    expect(installHints.length).toBeGreaterThan(0);
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

    const hints = await screen.findAllByText(/Format tools are not available/i);
    expect(hints.length).toBeGreaterThan(0);

    const missingTools = await screen.findAllByText(/Missing tools: mkfs\.ext4/i);
    expect(missingTools.length).toBeGreaterThan(0);
    const installHints = await screen.findAllByText(/apk add e2fsprogs/i);
    expect(installHints.length).toBeGreaterThan(0);
  });

  it("updates format support feedback when filesystem type changes", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { FilesystemFormatDialog } = await import("../FilesystemFormatDialog");

    const partition = {
      id: "part-format-2",
      name: "switch-test",
      device_path: "/dev/sdf1",
      fs_type: "f2fs",
      filesystem_info: {
        support: {
          canFormat: true,
        },
      },
    };

    const server = await getMswServer();
    server.use(
      http.get("/api/filesystem/support", ({ request }) => {
        const fsType = new URL(request.url).searchParams.get("fstype");
        if (fsType === "zfs") {
          return HttpResponse.json({
            canMount: true,
            canFormat: false,
            canCheck: false,
            canSetLabel: false,
            canGetState: true,
            alpinePackage: "zfs",
            missingTools: ["zpool"],
          });
        }

        return HttpResponse.json({
          canMount: true,
          canFormat: true,
          canCheck: true,
          canSetLabel: false,
          canGetState: true,
          alpinePackage: "f2fs-tools",
          missingTools: [],
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

    const formatButton = await screen.findByRole("button", { name: /format/i });
    expect((formatButton as HTMLButtonElement).disabled).toBe(false);

    const fsTypeInput = await screen.findByRole("textbox", {
      name: /filesystem type/i,
    });
    const user = userEvent.setup();
    await user.clear(fsTypeInput as HTMLInputElement);
    await user.type(fsTypeInput as HTMLInputElement, "zfs");

    const hints = await screen.findAllByText(/Format tools are not available/i);
    expect(hints.length).toBeGreaterThan(0);
    const installHints = await screen.findAllByText(/apk add zfs/i);
    expect(installHints.length).toBeGreaterThan(0);

    const disabledFormatButton = await screen.findByRole("button", {
      name: /format/i,
    });
    expect((disabledFormatButton as HTMLButtonElement).disabled).toBe(true);
  });
});
