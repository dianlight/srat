/* eslint-disable */
import { http, HttpResponse } from "msw";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { withTestHandlers } from "/test/testing";

const { toastInfoMock, toastErrorMock, confirmMock } = vi.hoisted(() => {
  const toastInfoMock = vi.fn((..._args: unknown[]) => undefined);
  const toastErrorMock = vi.fn((..._args: unknown[]) => undefined);
  const confirmMock = vi.fn(() =>
    Promise.resolve({ reason: "cancel" as const }),
  );
  return { toastInfoMock, toastErrorMock, confirmMock };
});

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
});
