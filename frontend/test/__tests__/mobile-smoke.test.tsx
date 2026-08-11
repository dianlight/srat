/**
 * Mobile 375px smoke test — task 10 of docs/tasks/045_full-mobile-support.md.
 *
 * Renders the full App in a real Chromium (Vitest browser mode) at a 375px
 * viewport and asserts:
 *  1. No document-level horizontal overflow (scrollWidth <= clientWidth) on
 *     Dashboard / Volumes / Shares / Users / Settings.
 *  2. The Setup Wizard opens full-screen with a vertical stepper at 375px.
 *  3. No RTK Query PARSING_ERROR-style console errors during the walk.
 *
 * Data comes from the shared MSW handlers (generated + custom + streaming),
 * so no live backend is required. The wrapper mirrors src/index.tsx (minus
 * DevInspector, which only connects to the local dev MCP server). The theme
 * must include `cssVariables` + `colorSchemes` — NavBar renders nothing when
 * `useColorScheme()` has no mode (it returns null).
 *
 * This file is browser-only: the `vitest/browser` module is unavailable under
 * the happy-dom suite, so the whole describe block is skipped there. Run it
 * with `mise run //frontend:test:browser`.
 */
import { CssBaseline } from "@mui/material";
import { ThemeProvider, createTheme } from "@mui/material/styles";
import { TourProvider } from "@reactour/tour";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ConfirmProvider } from "material-ui-confirm";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { BrowserPage } from "vitest/browser";
import { renderWithTestStore } from "/test/testing";

let page: BrowserPage | undefined;
try {
  ({ page } = await import("vitest/browser"));
} catch {
  page = undefined;
}

// DonationButton fetches funding info from real GitHub URLs; the shared MSW
// handlers don't cover them (NavBar.test.tsx mocks it for the same reason).
vi.mock("../../src/components/DonationButton", () => ({
  DonationButton: () => null,
}));

// Mirrors the production theme in src/index.tsx: the color scheme system is
// what gives `useColorScheme()` a mode, which NavBar requires to render.
const theme = createTheme({
  cssVariables: {
    colorSchemeSelector: "data-color-mode",
  },
  colorSchemes: {
    light: true,
    dark: true,
  },
});

const MOBILE_TABS = ["Volumes", "Shares", "Users", "Settings"] as const;

function assertNoHorizontalOverflow(where: string) {
  const { scrollWidth, clientWidth } = document.documentElement;
  expect(
    scrollWidth,
    `${where} overflows horizontally: scrollWidth=${scrollWidth} clientWidth=${clientWidth}`,
  ).toBeLessThanOrEqual(clientWidth);
}

// console.error args can be structured (RTK Query error objects, Error
// instances); String() alone would flatten them to "[object Object]" and hide
// a nested PARSING_ERROR. Serialize so the filter below can see them.
function argToString(arg: unknown): string {
  if (typeof arg === "string") return arg;
  if (arg instanceof Error) return arg.message;
  try {
    return JSON.stringify(arg) ?? String(arg);
  } catch {
    return String(arg);
  }
}

describe.runIf(!!page)("mobile 375px smoke", () => {
  let consoleErrors: string[] = [];
  let errorSpy: ReturnType<typeof vi.spyOn> | undefined;

  beforeEach(async () => {
    localStorage.clear();
    consoleErrors = [];
    errorSpy = vi.spyOn(console, "error").mockImplementation((...args: unknown[]) => {
      consoleErrors.push(args.map(argToString).join(" "));
    });
  });

  afterEach(() => {
    // Always restore the spy so a failed assertion does not leak into later
    // browser tests.
    errorSpy?.mockRestore();
  });

  it(
    "walks all main tabs at 375px without horizontal overflow and opens the wizard full-screen",
    { timeout: 240_000 },
    async () => {
      await page!.viewport(375, 844);

      localStorage.setItem("srat_tab", "0"); // Dashboard
      const { App } = await import("../../src/App");
      const appTree: ReactElement = (
        <ThemeProvider theme={theme}>
          <CssBaseline />
          <ConfirmProvider>
            <MemoryRouter initialEntries={["/"]}>
              <TourProvider steps={[]}>
                <App />
              </TourProvider>
            </MemoryRouter>
          </ConfirmProvider>
        </ThemeProvider>
      );
      const result = await renderWithTestStore(appTree);

      // Wait for the app to mount and MSW-backed data to arrive.
      await waitFor(
        () => {
          expect(result.container.querySelector("header")).toBeTruthy();
        },
        { timeout: 60_000 },
      );
      await new Promise((resolve) => setTimeout(resolve, 3000));
      assertNoHorizontalOverflow("Dashboard");

      // Walk the remaining tabs through the xs hamburger menu.
      for (const tab of MOBILE_TABS) {
        await userEvent.click(
          screen.getByRole("button", { name: "navigation menu" }),
        );
        const item = await screen.findByRole("menuitem", { name: tab });
        await userEvent.click(item);
        await new Promise((resolve) => setTimeout(resolve, 3000));
        assertNoHorizontalOverflow(tab);
      }

      // Setup wizard: open from the Settings page and verify the xs layout.
      const wizardButton = screen.getByRole("button", {
        name: /setup wizard/i,
      });
      await userEvent.click(wizardButton);
      await waitFor(
        () => {
          expect(document.querySelector(".MuiDialog-root")).toBeTruthy();
        },
        { timeout: 15_000 },
      );
      const paper = document.querySelector(".MuiDialog-paper");
      expect(paper?.className).toContain("MuiDialog-paperFullScreen");
      const stepper = document.querySelector(".MuiStepper-root");
      expect(stepper?.className).toContain("MuiStepper-vertical");
      assertNoHorizontalOverflow("SetupWizard");
      await userEvent.keyboard("{Escape}");

      // No PARSING_ERROR-style failures while walking the app.
      const parsingErrors = consoleErrors.filter((entry) =>
        entry.includes("PARSING_ERROR"),
      );
      expect(parsingErrors).toEqual([]);
    },
  );
});
