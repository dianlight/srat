/* eslint-disable */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

describe("ShareEditForm component", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  const setupCommonOverrides = () => {
    const volumeResult = {
      disks: [
        {
          partitions: [
            {
              mount_point_data: [
                {
                  path: "/mnt/free",
                  path_hash: "free-hash",
                  disk_label: "FreeDisk",
                  is_mounted: true,
                  is_write_supported: true,
                  time_machine_support: "Supported",
                },
              ],
            },
          ],
        },
      ],
      isLoading: false,
      error: null,
    };

    const usersResult = {
      data: [
        { username: "admin", is_admin: true },
        { username: "guest", is_admin: false },
      ],
      isLoading: false,
      error: null,
      refetch: () => Promise.resolve(),
    };

    const overrides = {
      useVolume: () => volumeResult,
      useGetApiUsersQuery: () => usersResult,
      useLabFeatures: () => ({ features: [], isAvailable: () => false, isLoading: false }),
      useGetApiSharesFoldersQuery: () => ({ data: undefined, isFetching: false, error: undefined }),
    } as const;

    return { overrides };
  };

  it("cycles share name casing and submits form", async () => {
    const { overrides } = setupCommonOverrides();

    const React = await import("react");
    const { render, screen, waitFor } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { Usage } = await import("../../../../store/sratApi");
    const { ShareEditForm } =
      // @ts-expect-error - Query param fetches isolated module instance
      await import("../ShareEditForm?share-edit-form-test");

    const handleSubmit = vi.fn(() => {});

    render(
      React.createElement(ShareEditForm as any, {
        shareData: {
          name: "TestShare",
          usage: Usage.None,
          mount_point_data: {
            path: "/mnt/free",
            path_hash: "free-hash",
            is_mounted: true,
            is_write_supported: true,
          },
        },
        shares: {
          docKey: {
            name: "Documents",
            usage: "general",
            mount_point_data: {
              path_hash: "doc-hash",
            },
          },
        },
        onSubmit: handleSubmit,
        testOverrides: overrides,
      }),
    );

    const [nameInput] = await screen.findAllByLabelText(/Share Name/i);
    expect((nameInput as HTMLInputElement).value).toBe("TestShare");

    const cycleButton = await screen.findByRole("button", {
      name: /cycle share name casing/i,
    });
    const user = userEvent.setup();
    await user.click(cycleButton);
    await user.click(cycleButton);

    const addDefaults = (
      await screen.findAllByLabelText(/add suggested default veto files/i)
    )[0];
    if (addDefaults) {
      await user.click(addDefaults);
    }

    const submitButton = await screen.findByRole("button", { name: /create/i });
    await user.click(submitButton);

    await waitFor(() => expect(handleSubmit).toHaveBeenCalled());
    const submissionCalls = handleSubmit.mock.calls as any[];
    const submission = submissionCalls[0]?.[0];
    expect(submission?.name).toBeTruthy();
    expect(submission?.veto_files?.length).toBeGreaterThanOrEqual(0);
  }, 60000);

  it("renders delete action for existing share", async () => {
    const { overrides } = setupCommonOverrides();

    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { Usage } = await import("../../../../store/sratApi");
    const { ShareEditForm } =
      // @ts-expect-error - Query param fetches isolated module instance
      await import("../ShareEditForm?share-edit-form-existing");

    const handleDelete = vi.fn(() => {});

    render(
      React.createElement(ShareEditForm as any, {
        shareData: {
          org_name: "Existing",
          name: "Existing",
          usage: Usage.Share,
          mount_point_data: {
            path: "/mnt/existing",
            path_hash: "existing-hash",
            is_mounted: true,
            is_write_supported: true,
          },
        },
        shares: {},
        onSubmit: () => {},
        onDelete: handleDelete,
        testOverrides: overrides,
      }),
    );

    const deleteButton = await screen.findByRole("button", { name: /delete/i });
    const user = userEvent.setup();
    await user.click(deleteButton);

    expect(handleDelete).toHaveBeenCalledWith(
      "Existing",
      expect.objectContaining({ name: "Existing" }),
    );
  });

  it("hides Volume field and disables edit name button for internal shares", async () => {
    const { overrides } = setupCommonOverrides();

    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { Usage } = await import("../../../../store/sratApi");
    const { ShareEditForm } =
      // @ts-expect-error - Query param fetches isolated module instance
      await import("../ShareEditForm?share-edit-form-internal");

    render(
      React.createElement(ShareEditForm as any, {
        shareData: {
          org_name: "InternalShare",
          name: "InternalShare",
          usage: Usage.Internal,
          mount_point_data: {
            path: "/internal/path",
            path_hash: "internal-hash",
            is_mounted: true,
            is_write_supported: true,
          },
        },
        shares: {},
        onSubmit: () => {},
        testOverrides: overrides,
      }),
    );

    // Volume field should not be present for internal shares
    const volumeLabels = screen.queryAllByText("Volume");
    expect(volumeLabels.length).toBe(0);
  });

  it("does not submit the form when Enter is pressed in a text field", async () => {
    const { overrides } = setupCommonOverrides();

    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { Usage } = await import("../../../../store/sratApi");
    const { ShareEditForm } =
      // @ts-expect-error - Query param fetches isolated module instance
      await import("../ShareEditForm?share-edit-form-enter-guard");

    const handleSubmit = vi.fn();

    render(
      React.createElement(ShareEditForm as any, {
        shareData: {
          name: "TestShare",
          usage: Usage.None,
          mount_point_data: {
            path: "/mnt/free",
            path_hash: "free-hash",
            is_mounted: true,
            is_write_supported: true,
          },
        },
        shares: {},
        onSubmit: handleSubmit,
        testOverrides: overrides,
      }),
    );

    const [nameInput] = await screen.findAllByLabelText(/Share Name/i);
    const user = userEvent.setup();
    await user.click(nameInput);
    await user.keyboard("{Enter}");

    expect(handleSubmit).not.toHaveBeenCalled();
  });

  it("blocks submit when the selected volume is not mounted (issue #1162)", async () => {
    const volumeResult = {
      disks: [
        {
          partitions: [
            {
              mount_point_data: [
                {
                  path: "/mnt/unmounted",
                  path_hash: "unmounted-hash",
                  disk_label: "UnmountedDisk",
                  is_mounted: false,
                  is_write_supported: false,
                  time_machine_support: "Unsupported",
                },
              ],
            },
          ],
        },
      ],
      isLoading: false,
      error: null,
    };

    const usersResult = {
      data: [{ username: "admin", is_admin: true }],
      isLoading: false,
      error: null,
      refetch: () => Promise.resolve(),
    };

    const overrides = {
      useVolume: () => volumeResult,
      useGetApiUsersQuery: () => usersResult,
      useLabFeatures: () => ({ features: [], isAvailable: () => false, isLoading: false }),
      useGetApiSharesFoldersQuery: () => ({ data: undefined, isFetching: false, error: undefined }),
    } as const;

    const React = await import("react");
    const { render, screen, waitFor } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { Usage } = await import("../../../../store/sratApi");
    const { ShareEditForm } =
      // @ts-expect-error - Query param fetches isolated module instance
      await import("../ShareEditForm?share-edit-form-unmounted-guard");

    const handleSubmit = vi.fn(() => {});

    render(
      React.createElement(ShareEditForm as any, {
        shareData: {
          name: "GuardShare",
          usage: Usage.None,
          mount_point_data: {
            path: "/mnt/unmounted",
            path_hash: "unmounted-hash",
            is_mounted: false,
            is_write_supported: false,
          },
        },
        shares: {},
        onSubmit: handleSubmit,
        testOverrides: overrides,
      }),
    );

    const submitButton = await screen.findByRole("button", { name: /create/i });
    const user = userEvent.setup();
    await user.click(submitButton);

    await waitFor(() => expect(screen.getByText(/not mounted/i)).toBeTruthy());
    expect(handleSubmit).not.toHaveBeenCalled();
  }, 60000);

  it("allows hyphens in share names (issue #1400)", async () => {
    const { overrides } = setupCommonOverrides();

    const React = await import("react");
    const { render, screen, waitFor } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { Usage } = await import("../../../../store/sratApi");
    const { ShareEditForm } =
      // @ts-expect-error - Query param fetches isolated module instance
      await import("../ShareEditForm?share-edit-form-hyphen");

    const handleSubmit = vi.fn(() => {});

    render(
      React.createElement(ShareEditForm as any, {
        shareData: {
          name: "paritycheck1",
          usage: Usage.None,
          mount_point_data: {
            path: "/mnt/free",
            path_hash: "free-hash",
            is_mounted: true,
            is_write_supported: true,
          },
        },
        shares: {},
        onSubmit: handleSubmit,
        testOverrides: overrides,
      }),
    );

    const user = userEvent.setup();
    const [nameInput] = await screen.findAllByLabelText(/Share Name/i);
    expect((nameInput as HTMLInputElement).value).toBe("paritycheck1");
    // Type hyphens through the real input so the charset validation runs on them
    await user.click(nameInput);
    await user.keyboard("{End}-2");
    expect((nameInput as HTMLInputElement).value).toBe("paritycheck1-2");

    const submitButton = await screen.findByRole("button", { name: /create/i });
    await user.click(submitButton);

    // No charset validation error for the hyphenated name
    expect(screen.queryByText(/can only contain/i)).toBeNull();

    await waitFor(() => expect(handleSubmit).toHaveBeenCalled());
    const submission = (handleSubmit.mock.calls as any[])[0]?.[0];
    expect(submission?.name).toBe("paritycheck1-2");
  }, 60000);

  it("keeps Usage read-only for shares on an HA-mounted volume (issue #1646)", async () => {
    const { overrides } = setupCommonOverrides();

    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { Usage } = await import("../../../../store/sratApi");
    const { ShareEditForm } =
      // @ts-expect-error - Query param fetches isolated module instance
      await import("../ShareEditForm?share-edit-form-ha-usage");

    render(
      React.createElement(ShareEditForm as any, {
        shareData: {
          org_name: "HA_USBTEST",
          name: "HA_USBTEST",
          usage: Usage.Media,
          mount_point_data: {
            path: "/mnt/usbtest",
            is_mounted: true,
            is_ha_mounted: true,
            is_write_supported: true,
          },
        },
        shares: {},
        onSubmit: () => {},
        testOverrides: overrides,
      }),
    );

    const usageField = await screen.findByLabelText(/Usage/i);
    const usageControl = usageField.closest("[role='combobox']") ?? usageField;
    expect(
      usageControl.getAttribute("aria-disabled") === "true" ||
        (usageControl as HTMLInputElement).disabled === true,
    ).toBe(true);
    expect(
      screen.getByText(/Usage is managed by Home Assistant/i),
    ).toBeTruthy();
  }, 60000);

  // Share subfolders (issue #184) are a Lab feature.
  const withSubfolders = (available: boolean) => {
    const { overrides } = setupCommonOverrides();
    return {
      ...overrides,
      useLabFeatures: () => ({
        features: [],
        isAvailable: (key: string) => available && key === "share_subfolders",
        isLoading: false,
      }),
    };
  };

  it("hides the Subfolder field without the share_subfolders lab feature", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { Usage } = await import("../../../../store/sratApi");
    const { ShareEditForm } =
      // @ts-expect-error - Query param fetches isolated module instance
      await import("../ShareEditForm?share-edit-form-subfolder-hidden");

    render(
      React.createElement(ShareEditForm as any, {
        shareData: {
          name: "NoLab",
          usage: Usage.None,
          mount_point_data: { path: "/mnt/free", is_mounted: true, is_write_supported: true },
        },
        shares: {},
        onSubmit: () => {},
        testOverrides: withSubfolders(false),
      }),
    );

    await screen.findAllByLabelText(/Share Name/i);
    expect(screen.queryByLabelText(/Subfolder/i)).toBeNull();
  }, 60000);

  it("shares a folder of an already shared volume and refuses a folder in use", async () => {
    const React = await import("react");
    const { render, screen, waitFor } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { Usage } = await import("../../../../store/sratApi");
    const { ShareEditForm } =
      // @ts-expect-error - Query param fetches isolated module instance
      await import("../ShareEditForm?share-edit-form-subfolder-create");

    const handleSubmit = vi.fn(() => {});
    render(
      React.createElement(ShareEditForm as any, {
        shareData: {
          name: "EMPEROR",
          usage: Usage.None,
          mount_point_data: { path: "/mnt/free", is_mounted: true, is_write_supported: true },
        },
        shares: {
          FREE: { name: "FREE", mount_point_data: { path: "/mnt/free" } },
          PAVEL: { name: "PAVEL", subfolder: "pavel", mount_point_data: { path: "/mnt/free" } },
        },
        onSubmit: handleSubmit,
        testOverrides: withSubfolders(true),
      }),
    );

    const user = userEvent.setup();
    const submitButton = await screen.findByRole("button", { name: /create/i });

    // The volume root is taken by FREE: a subfolder is required.
    await user.click(submitButton);
    await waitFor(() =>
      expect(screen.getByText(/already exposes this whole volume/i)).toBeTruthy(),
    );

    const subfolder = await screen.findByLabelText(/Subfolder/i);
    await user.type(subfolder, "pavel");
    await user.click(submitButton);
    await waitFor(() =>
      expect(screen.getByText(/already exposes this folder/i)).toBeTruthy(),
    );

    await user.clear(subfolder);
    await user.type(subfolder, "../etc");
    await user.click(submitButton);
    await waitFor(() => expect(screen.getByText(/are not allowed/i)).toBeTruthy());
    expect(handleSubmit).not.toHaveBeenCalled();

    await user.clear(subfolder);
    await user.type(subfolder, "emperor");
    await user.click(submitButton);
    await waitFor(() => expect(handleSubmit).toHaveBeenCalled());
    const submission = (handleSubmit.mock.calls as any[])[0]?.[0];
    expect(submission?.subfolder).toBe("emperor");
    expect(submission?.mount_point_data?.path).toBe("/mnt/free");
  }, 60000);

  it("shows a stored subfolder read-only when Lab Mode is off", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { Usage } = await import("../../../../store/sratApi");
    const { ShareEditForm } =
      // @ts-expect-error - Query param fetches isolated module instance
      await import("../ShareEditForm?share-edit-form-subfolder-readonly");

    render(
      React.createElement(ShareEditForm as any, {
        shareData: {
          org_name: "EMPEROR",
          name: "EMPEROR",
          usage: Usage.None,
          subfolder: "emperor",
          mount_point_data: { path: "/mnt/free", is_mounted: true, is_write_supported: true },
        },
        shares: {},
        onSubmit: () => {},
        testOverrides: withSubfolders(false),
      }),
    );

    const subfolder = (await screen.findByLabelText(/Subfolder/i)) as HTMLInputElement;
    expect(subfolder.value).toBe("emperor");
    expect(subfolder.disabled).toBe(true);
    expect(screen.getByText(/Turn on Lab Mode to change the subfolder/i)).toBeTruthy();
  }, 60000);

  it("fills the Subfolder field from the folder browser", async () => {
    const React = await import("react");
    const { render, screen, within } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { Usage } = await import("../../../../store/sratApi");
    const { ShareEditForm } =
      // @ts-expect-error - Query param fetches isolated module instance
      await import("../ShareEditForm?share-edit-form-subfolder-browse");

    const listings: Record<string, string[]> = { "": ["emperor", "Rodina"], Rodina: ["photos"] };
    const useGetApiSharesFoldersQuery = (arg: { mountPath: string; path?: string }, options?: { skip?: boolean }) =>
      options?.skip
        ? { data: undefined, isFetching: false, error: undefined }
        : {
            data: { mount_path: arg.mountPath, path: arg.path ?? "", folders: listings[arg.path ?? ""] ?? [], truncated: false },
            isFetching: false,
            error: undefined,
          };

    render(
      React.createElement(ShareEditForm as any, {
        shareData: {
          name: "PHOTOS",
          usage: Usage.None,
          mount_point_data: { path: "/mnt/free", is_mounted: true, is_write_supported: true },
        },
        shares: {},
        onSubmit: () => {},
        testOverrides: { ...withSubfolders(true), useGetApiSharesFoldersQuery },
      }),
    );

    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: /browse folders/i }));
    const dialog = await screen.findByRole("dialog", { name: /choose a folder/i });
    await user.click(within(dialog).getByRole("button", { name: "Rodina" }));
    await user.click(within(dialog).getByRole("button", { name: "photos" }));
    await user.click(within(dialog).getByRole("button", { name: 'Use "photos"' }));

    const subfolder = (await screen.findByLabelText(/Subfolder/i)) as HTMLInputElement;
    expect(subfolder.value).toBe("Rodina/photos");
  }, 60000);
});
