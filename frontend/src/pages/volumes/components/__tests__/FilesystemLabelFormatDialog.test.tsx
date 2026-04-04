import { cleanup } from "@testing-library/react";
import { afterEach, describe, expect, it } from "bun:test";
import { http, HttpResponse } from "msw";
import { withTestHandlers } from "../../../../../test/bun-setup";
import "../../../../../test/setup";

const filesystemsUrl = /.*\/api\/filesystems(?:\?.*)?$/;

afterEach(() => {
  cleanup();
});

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
    const { sratApi } = await import("../../../../store/sratApi");
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

    await renderWithProviders(
      React.createElement(FilesystemLabelDialog as any, {
        open: true,
        partition,
        onClose: () => {},
      }),
      {
        seedStore: (store) => {
          store.dispatch(
            sratApi.util.upsertQueryData(
              "getApiFilesystemSupport",
              { fstype: "ext4" },
              {
                canMount: true,
                canFormat: true,
                canCheck: true,
                canSetLabel: false,
                canGetState: true,
                alpinePackage: "e2fsprogs",
                missingTools: ["e2label"],
                isExportable: false,
                isCheckReportProgress: false,
                isFormatReportProgress: false,
                labelRule: "",
              },
            ),
          );
        },
      },
    );

    const missingTools = await screen.findAllByText(/Missing tools: e2label/i);
    const button = await screen.findByRole("button", { name: /set label/i });
    expect((button as HTMLButtonElement).disabled).toBe(true);

    expect(missingTools.length).toBeGreaterThan(0);
    const installHints = await screen.findAllByText(/apk add e2fsprogs/i);
    expect(installHints.length).toBeGreaterThan(0);
  });

  it("validates the label dialog input against LabelRule and shows the accepted format hint", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { sratApi } = await import("../../../../store/sratApi");
    const { FilesystemLabelDialog } = await import("../FilesystemLabelDialog");

    const partition = {
      id: "part-label-rule-1",
      name: "",
      device_path: "/dev/sdz1",
      fs_type: "vfat",
    };

    await renderWithProviders(
      React.createElement(FilesystemLabelDialog as any, {
        open: true,
        partition,
        onClose: () => {},
      }),
      {
        seedStore: (store) => {
          store.dispatch(
            sratApi.util.upsertQueryData(
              "getApiFilesystemSupport",
              { fstype: "vfat" },
              {
                canMount: true,
                canFormat: true,
                canCheck: true,
                canSetLabel: true,
                canGetState: true,
                labelRule: "^[A-Z0-9]{1,5}$",
                alpinePackage: "dosfstools",
                missingTools: [],
                isExportable: false,
                isCheckReportProgress: false,
                isFormatReportProgress: false,
              },
            ),
          );
        },
      },
    );

    const input = await screen.findByRole("textbox", { name: /label/i });
    const button = await screen.findByRole("button", { name: /set label/i });
    const user = userEvent.setup();

    expect(await screen.findByText(/Accepted format: \^\[A-Z0-9\]\{1,5\}\$/i)).toBeTruthy();
    expect((button as HTMLButtonElement).disabled).toBe(true);

    await user.type(input, "bad-label");
    expect(await screen.findByText(/Invalid label\./i)).toBeTruthy();
    expect((button as HTMLButtonElement).disabled).toBe(true);

    await user.clear(input);
    await user.type(input, "DATA");
    expect((button as HTMLButtonElement).disabled).toBe(false);
  });

  it("shows the accepted format hint for the optional format label and keeps empty values allowed", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const { sratApi } = await import("../../../../store/sratApi");
    const { FilesystemFormatDialog } = await import("../FilesystemFormatDialog");

    const partition = {
      id: "part-format-rule-1",
      name: "format-rule",
      device_path: "/dev/sdy1",
      fs_type: "ext4",
      filesystem_info: {
        support: {
          canFormat: true,
        },
      },
    };

    await withTestHandlers(
      [
        http.get(filesystemsUrl, () =>
          HttpResponse.json({
            filesystems: [
              {
                name: "Extended Filesystem",
                type: "ext4",
                description: "EXT4 Filesystem",
                support: {
                  canMount: true,
                  canFormat: true,
                  canCheck: true,
                  canSetLabel: true,
                  canGetState: true,
                  isExportable: false,
                },
              },
            ],
            mount_flags: [],
          }),
        ),
      ],
      async () => {
        await renderWithProviders(
          React.createElement(FilesystemFormatDialog as any, {
            open: true,
            partition,
            onClose: () => {},
          }),
          {
            seedStore: (store) => {
              store.dispatch(
                sratApi.util.upsertQueryData(
                  "getApiFilesystemSupport",
                  { fstype: "ext4" },
                  {
                    canMount: true,
                    canFormat: true,
                    canCheck: true,
                    canSetLabel: true,
                    canGetState: true,
                    labelRule: "^[A-Z0-9]{1,5}$",
                    alpinePackage: "e2fsprogs",
                    missingTools: [],
                    isExportable: false,
                    isCheckReportProgress: false,
                    isFormatReportProgress: false,
                  },
                ),
              );
            },
          },
        );

        const input = await screen.findByRole("textbox", {
          name: /label \(optional\)/i,
        });

        expect((input as HTMLInputElement).value).toBe("");
        expect(
          await screen.findByText(/Accepted format: \^\[A-Z0-9\]\{1,5\}\$/i),
        ).toBeTruthy();
        expect(screen.queryByText(/Invalid label\./i)).toBeNull();
      },
    );
  });

  it("disables Format when format tools are unavailable", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const { sratApi } = await import("../../../../store/sratApi");
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

    await renderWithProviders(
      React.createElement(FilesystemFormatDialog as any, {
        open: true,
        partition,
        onClose: () => {},
      }),
      {
        seedStore: (store) => {
          store.dispatch(
            sratApi.util.upsertQueryData(
              "getApiFilesystemSupport",
              { fstype: "ext4" },
              {
                canMount: true,
                canFormat: false,
                canCheck: true,
                canSetLabel: true,
                canGetState: true,
                alpinePackage: "e2fsprogs",
                missingTools: ["mkfs.ext4"],
                isExportable: false,
                isCheckReportProgress: false,
                isFormatReportProgress: false,
                labelRule: "",
              },
            ),
          );
        },
      },
    );

    const hints = await screen.findAllByText(/Format tools are not available/i);
    const button = await screen.findByRole("button", { name: /format/i });
    expect((button as HTMLButtonElement).disabled).toBe(true);

    expect(hints.length).toBeGreaterThan(0);

    const missingTools = await screen.findAllByText(/Missing tools: mkfs\.ext4/i);
    expect(missingTools.length).toBeGreaterThan(0);
    const installHints = await screen.findAllByText(/apk add e2fsprogs/i);
    expect(installHints.length).toBeGreaterThan(0);
  });

  it("shows only format-capable filesystems in format type dropdown", async () => {
    const React = await import("react");
    const { fireEvent, screen } = await import("@testing-library/react");
    const { sratApi } = await import("../../../../store/sratApi");
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

    await renderWithProviders(
      React.createElement(FilesystemFormatDialog as any, {
        open: true,
        partition,
        onClose: () => {},
      }),
      {
        seedStore: (store) => {
          store.dispatch(
            sratApi.util.upsertQueryData("getApiFilesystems", undefined, {
              filesystems: [
                {
                  name: "Extended Filesystem",
                  type: "ext4",
                  description: "EXT4 Filesystem",
                  support: {
                    canMount: true,
                    canFormat: true,
                    canCheck: true,
                    canSetLabel: true,
                    canGetState: true,
                    isExportable: false,
                    isCheckReportProgress: false,
                    isFormatReportProgress: false,
                    labelRule: "",
                  },
                },
                {
                  name: "Flash-Friendly FS",
                  type: "f2fs",
                  description: "F2FS Filesystem",
                  support: {
                    canMount: true,
                    canFormat: true,
                    canCheck: true,
                    canSetLabel: false,
                    canGetState: true,
                    isExportable: false,
                    isCheckReportProgress: false,
                    isFormatReportProgress: false,
                    labelRule: "",
                  },
                },
                {
                  name: "ZFS",
                  type: "zfs",
                  description: "ZFS",
                  support: {
                    canMount: true,
                    canFormat: false,
                    canCheck: false,
                    canSetLabel: false,
                    canGetState: true,
                    isExportable: false,
                    isCheckReportProgress: false,
                    isFormatReportProgress: false,
                    labelRule: "",
                  },
                },
              ],
              mount_flags: [],
            }),
          );
          store.dispatch(
            sratApi.util.upsertQueryData(
              "getApiFilesystemSupport",
              { fstype: "f2fs" },
              {
                canMount: true,
                canFormat: true,
                canCheck: true,
                canSetLabel: false,
                canGetState: true,
                isExportable: false,
                isCheckReportProgress: false,
                isFormatReportProgress: false,
                labelRule: "",
                alpinePackage: "f2fs-tools",
                missingTools: [],
              },
            ),
          );
          store.dispatch(
            sratApi.util.upsertQueryData(
              "getApiFilesystemSupport",
              { fstype: "ext4" },
              {
                canMount: true,
                canFormat: true,
                canCheck: true,
                canSetLabel: true,
                canGetState: true,
                isExportable: false,
                isCheckReportProgress: false,
                isFormatReportProgress: false,
                labelRule: "",
                alpinePackage: "e2fsprogs",
                missingTools: [],
              },
            ),
          );
        },
      },
    );

    const fsTypeDropdown = await screen.findByRole("combobox", {
      name: /filesystem type/i,
    });
    fireEvent.mouseDown(fsTypeDropdown);

    const ext4Option = await screen.findByRole("option", {
      name: /EXT4 Filesystem \(ext4\)/i,
    });
    expect(ext4Option).toBeTruthy();

    const f2fsOption = await screen.findByRole("option", {
      name: /F2FS Filesystem \(f2fs\)/i,
    });
    expect(f2fsOption).toBeTruthy();

    const zfsOption = screen.queryByRole("option", {
      name: /ZFS \(zfs\)/i,
    });
    expect(zfsOption).toBeNull();
  });
});
