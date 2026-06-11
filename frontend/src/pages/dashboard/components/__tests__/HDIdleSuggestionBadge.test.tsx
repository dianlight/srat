import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Disk } from "../../../../store/sratApi";
import { Enabled } from "../../../../store/sratApi";
import { HDIdleSuggestionBadge } from "../HDIdleSuggestionBadge";

// ---- Mocks ----------------------------------------------------------------

const ignoreSuggestionMock = vi.fn(() => ({
  unwrap: () => Promise.resolve(),
}));

const navigateMock = vi.fn();

vi.mock("react-router", () => ({
  useNavigate: () => navigateMock,
}));

const labModeRef = { value: true };

vi.mock("../../../../hooks/useLabMode", () => ({
  useLabMode: () => ({ labMode: labModeRef.value, isLoading: false }),
}));

vi.mock("../../../../store/sratApi", async () => {
  const actual = await vi.importActual<
    typeof import("../../../../store/sratApi")
  >("../../../../store/sratApi");
  return {
    ...actual,
    usePostApiDiskByDiskIdHdidleIgnoreSuggestionMutation: () => [
      ignoreSuggestionMock,
      { isLoading: false },
    ],
  };
});

// ---- Fixtures -------------------------------------------------------------

const rotationalDisk = (overrides: Partial<Disk> = {}): Disk => ({
  id: "ata-Some_HDD_1234",
  legacy_device_name: "sda",
  is_rotational: true,
  ...overrides,
});

// ---- Tests ----------------------------------------------------------------

describe("HDIdleSuggestionBadge", () => {
  beforeEach(() => {
    labModeRef.value = true;
    ignoreSuggestionMock.mockClear();
    navigateMock.mockClear();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("renders for an unconfigured rotational HDD", () => {
    render(<HDIdleSuggestionBadge disk={rotationalDisk()} />);
    expect(screen.getByTestId("hdidle-suggestion-badge")).toBeInTheDocument();
  });

  it("hides when Lab Mode is off", () => {
    labModeRef.value = false;
    render(<HDIdleSuggestionBadge disk={rotationalDisk()} />);
    expect(screen.queryByTestId("hdidle-suggestion-badge")).toBeNull();
  });

  it("hides for a non-rotational disk", () => {
    render(
      <HDIdleSuggestionBadge disk={rotationalDisk({ is_rotational: false })} />,
    );
    expect(screen.queryByTestId("hdidle-suggestion-badge")).toBeNull();
  });

  it("hides when is_rotational is unknown (treated as non-rotational)", () => {
    render(
      <HDIdleSuggestionBadge disk={rotationalDisk({ is_rotational: undefined })} />,
    );
    expect(screen.queryByTestId("hdidle-suggestion-badge")).toBeNull();
  });

  it("hides when HDIdle is already enabled (Yes)", () => {
    render(
      <HDIdleSuggestionBadge
        disk={rotationalDisk({
          hdidle_device: {
            idle_time: 60,
            power_condition: 0,
            enabled: Enabled.Yes,
          },
        })}
      />,
    );
    expect(screen.queryByTestId("hdidle-suggestion-badge")).toBeNull();
  });

  it("hides when HDIdle is already enabled (Custom)", () => {
    render(
      <HDIdleSuggestionBadge
        disk={rotationalDisk({
          hdidle_device: {
            idle_time: 120,
            power_condition: 0,
            enabled: Enabled.Custom,
          },
        })}
      />,
    );
    expect(screen.queryByTestId("hdidle-suggestion-badge")).toBeNull();
  });

  it("hides when the suggestion has been ignored", () => {
    render(
      <HDIdleSuggestionBadge
        disk={rotationalDisk({
          hdidle_device: {
            idle_time: 0,
            power_condition: 0,
            enabled: Enabled.No,
            suggestion_ignored: true,
          },
        })}
      />,
    );
    expect(screen.queryByTestId("hdidle-suggestion-badge")).toBeNull();
  });

  it("shows when disabled with suggestion not ignored", () => {
    render(
      <HDIdleSuggestionBadge
        disk={rotationalDisk({
          hdidle_device: {
            idle_time: 0,
            power_condition: 0,
            enabled: Enabled.No,
            suggestion_ignored: false,
          },
        })}
      />,
    );
    expect(screen.getByTestId("hdidle-suggestion-badge")).toBeInTheDocument();
  });

  it("hides when disk is undefined", () => {
    render(<HDIdleSuggestionBadge disk={undefined} />);
    expect(screen.queryByTestId("hdidle-suggestion-badge")).toBeNull();
  });

  it("Enable button navigates to volumes page with disk query", () => {
    render(<HDIdleSuggestionBadge disk={rotationalDisk()} />);
    fireEvent.click(screen.getByLabelText("enable hdidle"));
    expect(navigateMock).toHaveBeenCalledWith(
      `/volumes?disk=${encodeURIComponent("ata-Some_HDD_1234")}`,
    );
  });

  it("Enable button falls back to /volumes when disk has no id", () => {
    render(<HDIdleSuggestionBadge disk={rotationalDisk({ id: undefined })} />);
    fireEvent.click(screen.getByLabelText("enable hdidle"));
    expect(navigateMock).toHaveBeenCalledWith("/volumes");
  });

  it("Ignore button calls ignore-suggestion mutation with disk id", () => {
    render(<HDIdleSuggestionBadge disk={rotationalDisk()} />);
    fireEvent.click(screen.getByLabelText("ignore hdidle suggestion"));
    expect(ignoreSuggestionMock).toHaveBeenCalledWith({
      diskId: "ata-Some_HDD_1234",
    });
  });

  it("Ignore button is a no-op when disk has no id", () => {
    render(<HDIdleSuggestionBadge disk={rotationalDisk({ id: undefined })} />);
    fireEvent.click(screen.getByLabelText("ignore hdidle suggestion"));
    expect(ignoreSuggestionMock).not.toHaveBeenCalled();
  });
});
