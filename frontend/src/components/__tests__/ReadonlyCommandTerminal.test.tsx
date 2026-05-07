import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import {
    createTerminalLines,
    ReadonlyCommandTerminal,
} from "../ReadonlyCommandTerminal";

describe("ReadonlyCommandTerminal", () => {
  it("renders empty text when there are no lines", () => {
    render(<ReadonlyCommandTerminal lines={[]} />);

    expect(screen.getByText("No output available.")).toBeTruthy();
  });

  it("renders stdout, stderr, and info lines with channel labels", () => {
    render(
      <ReadonlyCommandTerminal
        lines={[
          { channel: "stdout", line: "ok line", timestamp: 1 },
          { channel: "stderr", line: "error line", timestamp: 2 },
          { channel: "info", line: "internal message", timestamp: 3 },
        ]}
      />,
    );

    expect(screen.getByText("ok line")).toBeTruthy();
    expect(screen.getByText("error line")).toBeTruthy();
    expect(screen.getByText("internal message")).toBeTruthy();
    expect(screen.getByText("[stdout]", { exact: false })).toBeTruthy();
    expect(screen.getByText("[stderr]", { exact: false })).toBeTruthy();
    expect(screen.getByText("[info]", { exact: false })).toBeTruthy();
  });

  it("infers filesystem note channels from their content", () => {
    const inferredLines = createTerminalLines(
      [
        "Starting filesystem check...",
        "fsck.fat 4.2 (2021-01-31)",
        "ERROR: inode bitmap mismatch",
      ],
      "stdout",
      1,
    );

    expect(inferredLines).toEqual([
      { channel: "info", line: "Starting filesystem check...", timestamp: 1 },
      { channel: "stdout", line: "fsck.fat 4.2 (2021-01-31)", timestamp: 2 },
      { channel: "stderr", line: "inode bitmap mismatch", timestamp: 3 },
    ]);
  });
});
