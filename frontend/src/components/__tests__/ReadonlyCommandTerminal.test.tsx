import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";
import "../../../test/setup";
import { ReadonlyCommandTerminal } from "../ReadonlyCommandTerminal";

describe("ReadonlyCommandTerminal", () => {
  it("renders empty text when there are no lines", () => {
    render(<ReadonlyCommandTerminal lines={[]} />);

    expect(screen.getByText("No output available.")).toBeTruthy();
  });

  it("renders stdout and stderr lines with channel labels", () => {
    render(
      <ReadonlyCommandTerminal
        lines={[
          { channel: "stdout", line: "ok line", timestamp: 1 },
          { channel: "stderr", line: "error line", timestamp: 2 },
        ]}
      />,
    );

    expect(screen.getByText("ok line")).toBeTruthy();
    expect(screen.getByText("error line")).toBeTruthy();
    expect(screen.getByText("[stdout]", { exact: false })).toBeTruthy();
    expect(screen.getByText("[stderr]", { exact: false })).toBeTruthy();
  });
});
