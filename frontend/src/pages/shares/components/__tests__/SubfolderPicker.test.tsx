import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { useGetApiSharesFoldersQuery } from "../../../../store/sratApi";
import { SubfolderPicker } from "../SubfolderPicker";

const listings: Record<string, string[]> = {
  "": ["emperor", "Rodina"],
  Rodina: ["photos", "vacation"],
  "Rodina/photos": [],
};

type FoldersHook = typeof useGetApiSharesFoldersQuery;

function foldersHook(
  result: (path: string) => ReturnType<FoldersHook> | Record<string, unknown>,
): FoldersHook {
  return ((arg: { mountPath: string; path?: string }, options?: { skip?: boolean }) =>
    options?.skip
      ? { data: undefined, isFetching: false, error: undefined }
      : result(arg.path ?? "")) as unknown as FoldersHook;
}

const listingHook = foldersHook((path) => ({
  data: { mount_path: "/mnt/DATA1", path, folders: listings[path] ?? [], truncated: false },
  isFetching: false,
  error: undefined,
}));

function renderPicker(props: Partial<Parameters<typeof SubfolderPicker>[0]> = {}) {
  const onSelect = vi.fn();
  const onClose = vi.fn();
  render(
    <SubfolderPicker
      open
      mountPath="/mnt/DATA1"
      volumeLabel="DATA1"
      onSelect={onSelect}
      onClose={onClose}
      testOverrides={{ useGetApiSharesFoldersQuery: listingHook }}
      {...props}
    />,
  );
  return { onSelect, onClose };
}

describe("SubfolderPicker", () => {
  it("browses into nested folders and picks one", async () => {
    const user = userEvent.setup();
    const { onSelect, onClose } = renderPicker();

    const folders = screen.getByRole("list", { name: /folders/i });
    expect(within(folders).getAllByRole("button").map((b) => b.textContent)).toEqual(["emperor", "Rodina"]);

    await user.click(within(folders).getByRole("button", { name: "Rodina" }));
    await user.click(screen.getByRole("button", { name: "photos" }));
    expect(screen.getByText(/no folders here/i)).toBeTruthy();

    await user.click(screen.getByRole("button", { name: 'Use "photos"' }));
    expect(onSelect).toHaveBeenCalledWith("Rodina/photos");
    expect(onClose).toHaveBeenCalled();
  });

  it("filters folders and goes back with the breadcrumbs", async () => {
    const user = userEvent.setup();
    renderPicker({ initialPath: "Rodina" });

    await user.type(screen.getByLabelText(/filter folders/i), "VAC");
    const folders = screen.getByRole("list", { name: /folders/i });
    expect(within(folders).getAllByRole("button").map((b) => b.textContent)).toEqual(["vacation"]);

    await user.clear(screen.getByLabelText(/filter folders/i));
    await user.type(screen.getByLabelText(/filter folders/i), "zzz");
    expect(screen.getByText(/no folder matches/i)).toBeTruthy();

    const crumbs = screen.getByRole("navigation", { name: /current folder/i });
    await user.click(within(crumbs).getByRole("button", { name: "DATA1" }));
    expect(screen.getByRole("button", { name: "emperor" })).toBeTruthy();
    expect(screen.getByRole("button", { name: /use the whole volume/i })).toBeTruthy();
  });

  it("names a new folder that is created when the share is saved", async () => {
    const user = userEvent.setup();
    const { onSelect } = renderPicker();

    await user.type(screen.getByLabelText("New folder"), "pavel{Enter}");
    expect(screen.getByText(/does not exist yet/i)).toBeTruthy();

    await user.type(screen.getByLabelText("New folder"), "docs");
    await user.click(screen.getByRole("button", { name: /add new folder/i }));
    const crumbs = screen.getByRole("navigation", { name: /current folder/i });
    await user.click(within(crumbs).getByRole("button", { name: "pavel" }));
    expect(screen.getByText(/does not exist yet/i)).toBeTruthy();

    await user.click(screen.getByRole("button", { name: 'Use "pavel"' }));
    expect(onSelect).toHaveBeenCalledWith("pavel");
  });

  it("refuses invalid new folder names", async () => {
    const user = userEvent.setup();
    renderPicker();

    await user.type(screen.getByLabelText("New folder"), "..");

    expect(screen.getByText(/are not allowed/i)).toBeTruthy();
    expect(
      (screen.getByRole("button", { name: /add new folder/i }) as HTMLButtonElement).disabled,
    ).toBe(true);
  });

  it("shows why a folder cannot be listed", () => {
    renderPicker({
      testOverrides: {
        useGetApiSharesFoldersQuery: foldersHook(() => ({
          data: undefined,
          isFetching: false,
          error: { status: 422, data: { detail: "subfolder points outside the volume" } },
        })),
      },
    });

    expect(screen.getByText(/points outside the volume/i)).toBeTruthy();
  });

  it("shows loading and tells when the listing was truncated", () => {
    const { unmount } = render(
      <SubfolderPicker
        open
        mountPath="/mnt/DATA1"
        onSelect={() => {}}
        onClose={() => {}}
        testOverrides={{
          useGetApiSharesFoldersQuery: foldersHook(() => ({ data: undefined, isFetching: true, error: undefined })),
        }}
      />,
    );
    expect(screen.getByLabelText(/loading folders/i)).toBeTruthy();
    unmount();

    renderPicker({
      testOverrides: {
        useGetApiSharesFoldersQuery: foldersHook((path) => ({
          data: { mount_path: "/mnt/DATA1", path, folders: ["a", "b"], truncated: true },
          isFetching: false,
          error: undefined,
        })),
      },
    });
    expect(screen.getByText(/only the first 2 folders are shown/i)).toBeTruthy();
  });
});
