import AutorenewIcon from "@mui/icons-material/Autorenew";
import BuildIcon from "@mui/icons-material/Build";
import CheckCircleIcon from "@mui/icons-material/CheckCircle";
import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "bun:test";
import "../../../test/setup";
import { IconProgress } from "../IconProgress";

describe("IconProgress", () => {
  const DEFAULT_WAIT_TIMEOUT = 1500;

  it("cycles through icons while running", async () => {
    render(
      <IconProgress
        icons={[AutorenewIcon, BuildIcon]}
        animationSpeed={50}
        variant="determinate"
        value={40}
      />,
    );

    expect(screen.getByTestId("AutorenewIcon")).not.toBeNull();

    await waitFor(
      () => {
        expect(screen.getByTestId("BuildIcon")).not.toBeNull();
      },
      { timeout: DEFAULT_WAIT_TIMEOUT },
    );
  });

  it("renders complete icon when progress reaches 100", () => {
    render(
      <IconProgress
        icons={[AutorenewIcon, BuildIcon]}
        completeIcon={CheckCircleIcon}
        variant="determinate"
        value={100}
      />,
    );

    expect(screen.getByTestId("CheckCircleIcon")).not.toBeNull();
    expect(screen.queryByTestId("AutorenewIcon")).toBeNull();
  });

  it("falls back to default icon when icons array is empty", () => {
    render(<IconProgress icons={[]} />);

    expect(screen.getByTestId("HourglassEmptyIcon")).not.toBeNull();
  });
});
