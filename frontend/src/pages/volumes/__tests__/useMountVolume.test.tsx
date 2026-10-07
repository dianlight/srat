/* eslint-disable */
import { http, HttpResponse } from "msw";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { withTestHandlers } from "/test/testing";

const { toastInfoMock, toastErrorMock, confirmMock } = vi.hoisted(() => {
  const toastInfoMock = vi.fn((..._args: unknown[]) => undefined);
  const toastErrorMock = vi.fn((..._args: unknown[]) => undefined);
  const confirmMock = vi.fn(
    (): Promise<{ reason: "confirm" | "cancel" }> =>
      Promise.resolve({ reason: "cancel" as const }),
  );
  return { toastInfoMock, toastErrorMock, confirmMock };
});

const { sentryBreadcrumbMock, sentryCaptureMock } = vi.hoisted(() => {
  const sentryBreadcrumbMock = vi.fn((..._args: unknown[]) => undefined);
  const sentryCaptureMock = vi.fn((..._args: unknown[]) => undefined);
  return { sentryBreadcrumbMock, sentryCaptureMock };
});

vi.mock("@sentry/react", () => ({
  addBreadcrumb: (...args: unknown[]) => sentryBreadcrumbMock(...args),
  captureException: (...args: unknown[]) => sentryCaptureMock(...args),
}));

vi.mock("react-toastify", () => ({
  ToastContainer: () => null,
  Slide: () => null,
  toast: {
    info: (...args: unknown[]) => toastInfoMock(...args),
    error: (...args: unknown[]) => toastErrorMock(...args),
    warn: (..._args: unknown[]) => undefined,
  },
}));

vi.mock("material-ui-confirm", () => ({
  useConfirm: () => confirmMock,
}));

const mountUrl = /.*\/api\/volume\/mount(?:\?.*)?$/;

const partition = {
  id: "part-mount-1",
  name: "data-vol",
  device_path: "/dev/sdb1",
};

async function renderMountHarness(options?: {
  onCleared?: (...args: unknown[]) => void;
  mountData?: Record<string, unknown>;
}) {
  const React = await import("react");
  const { renderWithTestStore } = await import("/test/testing");
  const { Type } = await import("../../../store/sratApi");
  const { useMountVolume } = await import("../hooks/useMountVolume");

  const onCleared = options?.onCleared ?? (() => undefined);
  const mountData = (options?.mountData ?? {
    path: "/mnt/data",
    root: "/",
    type: Type.Addon,
    is_mounted: false,
  }) as Parameters<
    ReturnType<typeof useMountVolume>["onSubmitMountVolume"]
  >[0];

  function MountHarness() {
    const { onSubmitMountVolume } = useMountVolume({
      selectedPartition: partition as any,
      onCleared: onCleared as () => void,
    });
    const [rootError, setRootError] = React.useState<string | null>(null);
    const setError = (_name: "root", error: { message: string }) => {
      setRootError(error.message);
    };
    return React.createElement(
      "div",
      { "data-testid": "volumes-mount-hook-harness" },
      React.createElement(
        "button",
        {
          type: "button",
          onClick: () => void onSubmitMountVolume(mountData, setError),
        },
        "Submit mount",
      ),
      rootError
        ? React.createElement("p", { role: "alert" }, rootError)
        : null,
    );
  }

  return renderWithTestStore(React.createElement(MountHarness));
}

describe("useMountVolume", () => {
  beforeEach(() => {
    toastInfoMock.mockClear();
    toastErrorMock.mockClear();
    confirmMock.mockClear();
    sentryBreadcrumbMock.mockClear();
    sentryCaptureMock.mockClear();
  });

  it("mounts successfully, toasts, and clears selection", async () => {
    const { screen, waitFor } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;

    const onCleared = vi.fn();

    await withTestHandlers(
      [
        http.post(mountUrl, () =>
          HttpResponse.json({
            path: "/mnt/data",
            root: "/",
            type: "ADDON",
            is_mounted: true,
          }),
        ),
      ],
      async () => {
        await renderMountHarness({ onCleared });

        expect(
          screen.getByTestId("volumes-mount-hook-harness"),
        ).toBeInTheDocument();

        const user = userEvent.setup();
        await user.click(
          screen.getByRole("button", { name: /submit mount/i }),
        );

        await waitFor(() => {
          expect(toastInfoMock).toHaveBeenCalledWith(
            "Volume /mnt/data mounted successfully.",
          );
        });
        expect(toastErrorMock).not.toHaveBeenCalled();
        expect(screen.queryByRole("alert")).toBeNull();
        await waitFor(() => {
          expect(onCleared).toHaveBeenCalledTimes(1);
        });
      },
    );
  });

  it("surfaces mount failures via toast and setError root, then clears", async () => {
    const { screen, waitFor } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;

    const onCleared = vi.fn();

    await withTestHandlers(
      [
        http.post(mountUrl, () =>
          HttpResponse.json(
            { detail: "mount failed", status: 500 },
            { status: 500 },
          ),
        ),
      ],
      async () => {
        await renderMountHarness({ onCleared });

        const user = userEvent.setup();
        await user.click(
          screen.getByRole("button", { name: /submit mount/i }),
        );

        const alert = await screen.findByRole("alert");
        expect(alert.textContent).toContain("mount failed");
        await waitFor(() => {
          expect(toastErrorMock).toHaveBeenCalledTimes(1);
        });
        expect(toastErrorMock.mock.calls[0]?.[0]).toContain("mount failed");
        expect(confirmMock).not.toHaveBeenCalled();
        await waitFor(() => {
          expect(onCleared).toHaveBeenCalledTimes(1);
        });
      },
    );
  });

  it("reports the retried suggested path when the suggestion retry fails", async () => {    const { screen, waitFor } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;

    const onCleared = vi.fn();
    confirmMock.mockResolvedValueOnce({ reason: "confirm" as const });
    const suggested = "/mnt/data_fixed";
    let calls = 0;

    await withTestHandlers(
      [
        http.post(mountUrl, () => {
          calls += 1;
          if (calls === 1) {
            return HttpResponse.json(
              { detail: `Message: bad chars\nSuggestedPath: ${suggested}` },
              { status: 406 },
            );
          }
          return HttpResponse.json(
            { detail: "still failing", status: 500 },
            { status: 500 },
          );
        }),
      ],
      async () => {
        await renderMountHarness({ onCleared });

        const user = userEvent.setup();
        await user.click(
          screen.getByRole("button", { name: /submit mount/i }),
        );

        await waitFor(() => {
          expect(sentryBreadcrumbMock).toHaveBeenCalled();
        });
        const breadcrumb = sentryBreadcrumbMock.mock.calls[0]?.[0] as {
          data?: { path?: string };
        };
        expect(breadcrumb?.data?.path).toBe(suggested);
        await waitFor(() => {
          expect(toastErrorMock).toHaveBeenCalledTimes(1);
        });
        expect(toastErrorMock.mock.calls[0]?.[0]).toContain("still failing");
        await waitFor(() => {
          expect(onCleared).toHaveBeenCalledTimes(1);
        });
      },
    );
  });

  it("surfaces the invalid-path detail when the suggested path is declined", async () => {
    const { screen, waitFor } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;

    const onCleared = vi.fn();
    confirmMock.mockResolvedValueOnce({ reason: "cancel" as const });
    const suggested = "/mnt/data_fixed";

    await withTestHandlers(
      [
        http.post(mountUrl, () =>
          HttpResponse.json(
            { detail: `Message: bad chars\nSuggestedPath: ${suggested}` },
            { status: 406 },
          ),
        ),
      ],
      async () => {
        await renderMountHarness({ onCleared });

        const user = userEvent.setup();
        await user.click(
          screen.getByRole("button", { name: /submit mount/i }),
        );

        const alert = await screen.findByRole("alert");
        expect(alert.textContent).toContain("is invalid");
        expect(alert.textContent).toContain(suggested);
        await waitFor(() => {
          expect(sentryBreadcrumbMock).toHaveBeenCalled();
        });
        const breadcrumb = sentryBreadcrumbMock.mock.calls[0]?.[0] as {
          data?: { path?: string };
        };
        expect(breadcrumb?.data?.path).toBe("/mnt/data");
        await waitFor(() => {
          expect(onCleared).toHaveBeenCalledTimes(1);
        });
      },
    );
  });

  it("rejects invalid selection without calling the API", async () => {
    const { screen, waitFor } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;

    const onCleared = vi.fn();

    await withTestHandlers([], async () => {
      const React = await import("react");
      const { renderWithTestStore } = await import("/test/testing");
      const { useMountVolume } = await import("../hooks/useMountVolume");

      function InvalidHarness() {
        const { onSubmitMountVolume } = useMountVolume({
          selectedPartition: undefined,
          onCleared: onCleared as () => void,
        });
        const [rootError, setRootError] = React.useState<string | null>(null);
        return React.createElement(
          "div",
          null,
          React.createElement(
            "button",
            {
              type: "button",
              onClick: () =>
                void onSubmitMountVolume(undefined, (_name, error) =>
                  setRootError(error.message),
                ),
            },
            "Submit invalid",
          ),
          rootError
            ? React.createElement("p", { role: "alert" }, rootError)
            : null,
        );
      }

      await renderWithTestStore(React.createElement(InvalidHarness));
      const user = userEvent.setup();
      await user.click(screen.getByRole("button", { name: /submit invalid/i }));

      const alert = await screen.findByRole("alert");
      expect(alert.textContent).toContain("Invalid selection");
      await waitFor(() => {
        expect(toastErrorMock).toHaveBeenCalledTimes(1);
      });
      expect(onCleared).not.toHaveBeenCalled();
    });
  });
});
