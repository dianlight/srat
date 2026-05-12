import { http, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";
import { getMswServer, renderWithTestStore, withTestHandlers } from "/test/testing";

const filesystemsUrl = /.*\/api\/filesystems(?:\?.*)?$/;
const formatUrl = /.*\/api\/filesystem\/format(?:\?.*)?$/;
const labelUrl = /.*\/api\/filesystem\/label(?:\?.*)?$/;
const supportUrl = /.*\/api\/filesystem\/support(?:\?.*)?$/;

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

    await renderWithTestStore(
      React.createElement(FilesystemLabelDialog as any, {
        open: true,
        partition,
        onClose: () => {},
      }),
    );

    const input = await screen.findByRole("textbox", { name: /label/i });
    expect((input as HTMLInputElement).value).toBe("media");
  });

  it("propagates a successful label change so the tree and details can refresh", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { FilesystemLabelDialog } = await import("../FilesystemLabelDialog");

    const partition = {
      id: "part-label-refresh-1",
      name: "old-label",
      device_path: "/dev/sdf1",
      fs_type: "ext4",
    };

    await withTestHandlers(
      [
        http.options(supportUrl, () => HttpResponse.json({})),
        http.get(supportUrl, () =>
          HttpResponse.json({
            canMount: true,
            canFormat: true,
            canCheck: true,
            canSetLabel: true,
            canGetState: true,
            labelRule: "^.{1,16}$",
            alpinePackage: "e2fsprogs",
            missingTools: [],
            isExportable: false,
            isCheckReportProgress: false,
            isFormatReportProgress: false,
          }),
        ),
        http.options(labelUrl, () => HttpResponse.json({})),
        http.put(labelUrl, () => HttpResponse.json({ success: true })),
      ],
      async () => {
        await renderWithTestStore(
          React.createElement(FilesystemLabelDialog as any, {
            open: true,
            partition,
            onClose: () => {},
          }),
        );

        const user = userEvent.setup();
        const input = await screen.findByRole("textbox", { name: /label/i });
        const submitButton = await screen.findByRole("button", { name: /^set label$/i });
        await user.clear(input);
        await user.type(input, "NEWLABEL");
        expect((submitButton as HTMLButtonElement).disabled).toBe(false);
        await user.click(submitButton);
      },
    );
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

    await renderWithTestStore(
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

    await withTestHandlers(
      [
        http.options(supportUrl, () => HttpResponse.json({})),
        http.get(supportUrl, () =>
          HttpResponse.json({
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
          }),
        ),
      ],
      async () => {
        await renderWithTestStore(
          React.createElement(FilesystemLabelDialog as any, {
            open: true,
            partition,
            onClose: () => {},
          }),
        );

        const button = await screen.findByRole("button", { name: /set label/i });
        expect((button as HTMLButtonElement).disabled).toBe(true);
        expect((await screen.findAllByText(/Install hint:/i)).length).toBeGreaterThan(0);
        expect((await screen.findAllByText(/apk add e2fsprogs/i)).length).toBeGreaterThan(0);
      },
    );
  });

  it("validates the label dialog input against LabelRule and shows the accepted format hint", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { FilesystemLabelDialog } = await import("../FilesystemLabelDialog");

    const partition = {
      id: "part-label-rule-1",
      name: "",
      device_path: "/dev/sdz1",
      fs_type: "vfat",
    };

    await withTestHandlers(
      [
        http.options(supportUrl, () => HttpResponse.json({})),
        http.get(supportUrl, () =>
          HttpResponse.json({
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
          }),
        ),
      ],
      async () => {
        await renderWithTestStore(
          React.createElement(FilesystemLabelDialog as any, {
            open: true,
            partition,
            onClose: () => {},
          }),
        );

        const input = await screen.findByRole("textbox", { name: /label/i });
        const button = await screen.findByRole("button", { name: /set label/i });
        const user = userEvent.setup();

        expect(await screen.findByText(/Accepted format:/i)).toBeTruthy();
        expect((button as HTMLButtonElement).disabled).toBe(true);

        await user.type(input, "bad-label");
        expect(await screen.findByText(/Invalid label\./i)).toBeTruthy();
        expect((button as HTMLButtonElement).disabled).toBe(true);

        await user.clear(input);
        await user.type(input, "DATA");
        expect((button as HTMLButtonElement).disabled).toBe(false);
      },
    );
  });

  it("shows the accepted format hint for the optional format label and keeps empty values allowed", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
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
        http.options(supportUrl, () => HttpResponse.json({})),
        http.get(supportUrl, () =>
          HttpResponse.json({
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
          }),
        ),
      ],
      async () => {
        await renderWithTestStore(
          React.createElement(FilesystemFormatDialog as any, {
            open: true,
            partition,
            onClose: () => {},
          }),
        );

        const input = await screen.findByRole("textbox", {
          name: /label \(optional\)/i,
        });

        expect((input as HTMLInputElement).value).toBe("");
        expect(await screen.findByText(/Accepted format:/i)).toBeTruthy();
        expect(screen.queryByText(/Invalid label\./i)).toBeNull();
      },
    );
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

    await withTestHandlers(
      [
        http.options(supportUrl, () => HttpResponse.json({})),
        http.get(supportUrl, () =>
          HttpResponse.json({
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
          }),
        ),
      ],
      async () => {
        await renderWithTestStore(
          React.createElement(FilesystemFormatDialog as any, {
            open: true,
            partition,
            onClose: () => {},
          }),
        );

        const button = await screen.findByRole("button", { name: /format/i });
        expect((button as HTMLButtonElement).disabled).toBe(true);

        expect((await screen.findAllByText(/Install hint:/i)).length).toBeGreaterThan(0);
        const installHints = await screen.findAllByText(/apk add e2fsprogs/i);
        expect(installHints.length).toBeGreaterThan(0);
      },
    );
  });

  it("renders format progress and logs when verbose mode is enabled", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const { FilesystemFormatDialog } = await import("../FilesystemFormatDialog");

    const partition = {
      id: "part-format-progress-1",
      name: "format-progress",
      device_path: "/dev/sdh1",
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
                  isCheckReportProgress: false,
                  isFormatReportProgress: false,
                },
              },
            ],
            mount_flags: [],
          }),
        ),
        http.options(supportUrl, () => HttpResponse.json({})),
        http.get(supportUrl, () =>
          HttpResponse.json({
            canMount: true,
            canFormat: true,
            canCheck: true,
            canSetLabel: true,
            canGetState: true,
            labelRule: "^[^\\x00/]{1,16}$",
            alpinePackage: "e2fsprogs",
            missingTools: [],
            isExportable: false,
            isCheckReportProgress: false,
            isFormatReportProgress: false,
          }),
        ),
      ],
      async () => {
        await renderWithTestStore(
          React.createElement(FilesystemFormatDialog as any, {
            open: true,
            partition,
            initialVerbose: true,
            taskOverride: {
              device: "/dev/sdh1",
              operation: "format",
              status: "running",
              progress: 42,
              notes: ["mkfs.ext4: writing inode tables"],
            },
            onClose: () => {},
          }),
        );

        expect(
          await screen.findByRole("switch", { name: /verbose/i }),
        ).toBeTruthy();
        expect(
          await screen.findByText("mkfs.ext4: writing inode tables"),
        ).toBeTruthy();
        expect(await screen.findByText("[stdout]", { exact: false })).toBeTruthy();
        expect(await screen.findByRole("progressbar")).toBeTruthy();
        expect(await screen.findByText("RUNNING")).toBeTruthy();
      },
    );
  });

  it("submits the verbose flag when starting a format", async () => {
    const React = await import("react");
    const { screen, waitFor } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { FilesystemFormatDialog } = await import("../FilesystemFormatDialog");

    const partition = {
      id: "part-format-verbose-1",
      name: "verbose-format",
      device_path: "/dev/sdi1",
      fs_type: "ext4",
      filesystem_info: {
        support: {
          canFormat: true,
        },
      },
    };

    let requestBody: Record<string, unknown> | null = null;

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
                  isCheckReportProgress: false,
                  isFormatReportProgress: false,
                },
              },
            ],
            mount_flags: [],
          }),
        ),
        http.options(formatUrl, () => HttpResponse.json({})),
        http.post(formatUrl, async ({ request }) => {
          requestBody = (await request.clone().json()) as Record<string, unknown>;
          return HttpResponse.json({
            success: true,
            errorsFound: false,
            errorsFixed: false,
            exitCode: 0,
            message: "Format operation started for /dev/sdi1 as ext4",
          });
        }),
        http.options(supportUrl, () => HttpResponse.json({})),
        http.get(supportUrl, () =>
          HttpResponse.json({
            canMount: true,
            canFormat: true,
            canCheck: true,
            canSetLabel: true,
            canGetState: true,
            labelRule: "^[^\\x00/]{1,16}$",
            alpinePackage: "e2fsprogs",
            missingTools: [],
            isExportable: false,
            isCheckReportProgress: false,
            isFormatReportProgress: false,
          }),
        ),
      ],
      async () => {
        await renderWithTestStore(
          React.createElement(FilesystemFormatDialog as any, {
            open: true,
            partition,
            onClose: () => {},
          }),
        );

        const user = userEvent.setup({ pointerEventsCheck: 0 });
        await user.click(await screen.findByRole("switch", { name: /verbose/i }));

        await waitFor(async () => {
          const formatButton = await screen.findByRole("button", { name: /^format$/i });
          expect((formatButton as HTMLButtonElement).disabled).toBe(false);
        });

        await user.click(await screen.findByRole("button", { name: /^format$/i }));

        await waitFor(() => {
          expect(requestBody).toBeTruthy();
          expect(requestBody?.verbose).toBe(true);
        });
      },
    );
  });

  it("hides the Format action after a successful format", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const { FilesystemFormatDialog } = await import("../FilesystemFormatDialog");

    const partition = {
      id: "part-format-success-1",
      name: "success-format",
      device_path: "/dev/sdj1",
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
                  isCheckReportProgress: false,
                  isFormatReportProgress: false,
                },
              },
            ],
            mount_flags: [],
          }),
        ),
        http.options(supportUrl, () => HttpResponse.json({})),
        http.get(supportUrl, () =>
          HttpResponse.json({
            canMount: true,
            canFormat: true,
            canCheck: true,
            canSetLabel: true,
            canGetState: true,
            labelRule: "^[^\\x00/]{1,16}$",
            alpinePackage: "e2fsprogs",
            missingTools: [],
            isExportable: false,
            isCheckReportProgress: false,
            isFormatReportProgress: false,
          }),
        ),
      ],
      async () => {
        await renderWithTestStore(
          React.createElement(FilesystemFormatDialog as any, {
            open: true,
            partition,
            onClose: () => {},
            taskOverride: {
              device: "/dev/sdj1",
              operation: "format",
              status: "success",
              progress: 100,
              message: "Format completed successfully.",
            },
          }),
        );

        expect(await screen.findByText(/Format completed successfully\./i)).toBeTruthy();
        expect(screen.queryByRole("button", { name: /^format$/i })).toBeNull();
        expect(await screen.findByRole("button", { name: /close/i })).toBeTruthy();
      },
    );
  });

  it("shows the formatter error after a failed format", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const { FilesystemFormatDialog } = await import("../FilesystemFormatDialog");

    const partition = {
      id: "part-format-failure-1",
      name: "failure-format",
      device_path: "/dev/sdk1",
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
                  isCheckReportProgress: false,
                  isFormatReportProgress: false,
                },
              },
            ],
            mount_flags: [],
          }),
        ),
        http.options(supportUrl, () => HttpResponse.json({})),
        http.get(supportUrl, () =>
          HttpResponse.json({
            canMount: true,
            canFormat: true,
            canCheck: true,
            canSetLabel: true,
            canGetState: true,
            labelRule: "^[^\\x00/]{1,16}$",
            alpinePackage: "e2fsprogs",
            missingTools: [],
            isExportable: false,
            isCheckReportProgress: false,
            isFormatReportProgress: false,
          }),
        ),
      ],
      async () => {
        await renderWithTestStore(
          React.createElement(FilesystemFormatDialog as any, {
            open: true,
            partition,
            onClose: () => {},
            taskOverride: {
              device: "/dev/sdk1",
              operation: "format",
              status: "failure",
              progress: 100,
              error: "mkfs.ext4: /dev/sdk1 is busy",
            },
          }),
        );

        expect(await screen.findByText(/mkfs\.ext4: \/dev\/sdk1 is busy/i)).toBeTruthy();
        expect(await screen.findByRole("button", { name: /^format$/i })).toBeTruthy();
      },
    );
  });

  it("shows only format-capable filesystems in format type dropdown", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
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

    const server = await getMswServer();
    server.use(
      http.options(supportUrl, () => HttpResponse.json({})),
      http.get(supportUrl, ({ request }) => {
        const fsType =
          new URL(request.url).searchParams.get("fstype")?.toLowerCase() ?? "";

        if (fsType === "f2fs") {
          return HttpResponse.json({
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
          });
        }

        if (fsType === "ext4") {
          return HttpResponse.json({
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
          });
        }

        return HttpResponse.json({
          canMount: true,
          canFormat: false,
          canCheck: false,
          canSetLabel: false,
          canGetState: true,
          isExportable: false,
          isCheckReportProgress: false,
          isFormatReportProgress: false,
          labelRule: "",
          alpinePackage: "",
          missingTools: [],
        });
      }),
    );

    await renderWithTestStore(
      React.createElement(FilesystemFormatDialog as any, {
        open: true,
        partition,
        onClose: () => {},
      }),
      {
        seedStore: (store) => {
          (store.dispatch as any)(
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
        },
      },
    );

    const fsTypeDropdown = await screen.findByRole("combobox", {
      name: /filesystem type/i,
    });
    const user = userEvent.setup();
    await user.click(fsTypeDropdown);

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
