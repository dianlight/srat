/* eslint-disable */
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ResizableSplitView } from "../ResizableSplitView";

const STORAGE_KEY = "test.splitPct";

function createMatchMedia(matches: boolean) {
  return () =>
    ({
      matches,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
      onchange: null,
      media: "",
    }) as any;
}

function renderSplitView(props: Record<string, unknown> = {}) {
  return render(
    <ResizableSplitView
      storageKey={STORAGE_KEY}
      leftPanel={<div>Left Panel</div>}
      rightPanel={<div>Right Panel</div>}
      {...props}
    />,
  );
}

describe("ResizableSplitView", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
    // Desktop by default unless a test overrides it.
    (window as any).matchMedia = createMatchMedia(true);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders both panels side by side with a divider on desktop", () => {
    renderSplitView();

    expect(screen.getByText("Left Panel")).toBeInTheDocument();
    expect(screen.getByText("Right Panel")).toBeInTheDocument();
    expect(screen.getByRole("separator")).toBeInTheDocument();
  });

  it("stacks the panels vertically without a divider on mobile", () => {
    (window as any).matchMedia = createMatchMedia(false);

    renderSplitView();

    expect(screen.getByText("Left Panel")).toBeInTheDocument();
    expect(screen.getByText("Right Panel")).toBeInTheDocument();
    expect(screen.queryByRole("separator")).not.toBeInTheDocument();
  });

  it("restores a saved left panel width from localStorage", () => {
    localStorage.setItem(STORAGE_KEY, "42");

    renderSplitView();

    expect(screen.getByRole("separator")).toHaveAttribute(
      "aria-valuenow",
      "42",
    );
  });

  it("falls back to the default width when the saved value is out of range", () => {
    localStorage.setItem(STORAGE_KEY, "999");

    renderSplitView();

    expect(screen.getByRole("separator")).toHaveAttribute(
      "aria-valuenow",
      "30",
    );
  });

  it("falls back to the default width when the saved value is invalid", () => {
    localStorage.setItem(STORAGE_KEY, "not-a-number");

    renderSplitView();

    expect(screen.getByRole("separator")).toHaveAttribute(
      "aria-valuenow",
      "30",
    );
  });

  it("persists the width changed via the keyboard", async () => {
    const user = userEvent.setup();
    renderSplitView();

    const divider = screen.getByRole("separator");
    divider.focus();
    await user.keyboard("{ArrowRight}");

    expect(localStorage.getItem(STORAGE_KEY)).toBe("35");
    expect(divider).toHaveAttribute("aria-valuenow", "35");
  });

  it("decreases the width via the keyboard", async () => {
    const user = userEvent.setup();
    localStorage.setItem(STORAGE_KEY, "50");
    renderSplitView();

    const divider = screen.getByRole("separator");
    divider.focus();
    await user.keyboard("{ArrowLeft}");

    expect(localStorage.getItem(STORAGE_KEY)).toBe("45");
    expect(divider).toHaveAttribute("aria-valuenow", "45");
  });

  it("clamps the keyboard resize to the configured minimum", async () => {
    const user = userEvent.setup();
    localStorage.setItem(STORAGE_KEY, "17");
    renderSplitView();

    const divider = screen.getByRole("separator");
    divider.focus();
    await user.keyboard("{ArrowLeft}{ArrowLeft}{ArrowLeft}");

    expect(localStorage.getItem(STORAGE_KEY)).toBe("15");
  });

  it("persists the width changed by dragging the divider", async () => {
    const getBoundingClientRect = vi
      .spyOn(HTMLElement.prototype, "getBoundingClientRect")
      .mockReturnValue({
        x: 0,
        y: 0,
        top: 0,
        left: 0,
        right: 1000,
        bottom: 600,
        width: 1000,
        height: 600,
        toJSON: () => ({}),
      } as DOMRect);
    const user = userEvent.setup();
    renderSplitView();

    const divider = screen.getByRole("separator");
    await user.pointer({ keys: "[MouseLeft>]", target: divider });
    // Move the pointer to 45% of the container width.
    await user.pointer({ coords: { x: 450, y: 300 } });
    await user.pointer({ keys: "[/MouseLeft]" });

    expect(getBoundingClientRect).toHaveBeenCalled();
    expect(localStorage.getItem(STORAGE_KEY)).toBe("45");
  });

  it("honors a custom storage key so pages keep independent widths", () => {
    localStorage.setItem("other.splitPct", "55");
    renderSplitView({ storageKey: "other.splitPct" });

    expect(screen.getByRole("separator")).toHaveAttribute(
      "aria-valuenow",
      "55",
    );
  });
});
