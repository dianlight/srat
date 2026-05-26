import { createTheme, ThemeProvider } from "@mui/material/styles";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { Provider } from "react-redux";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { Settings } from "../Settings";
import { createTestStore, getMswServer } from "/test/testing";

type RenderResult = {
  user: ReturnType<typeof userEvent.setup>;
};

async function renderSettings(): Promise<RenderResult> {
  const store = await createTestStore();
  const theme = createTheme();
  const user = userEvent.setup();

  render(
    <Provider store={store}>
      <ThemeProvider theme={theme}>
        <Settings />
      </ThemeProvider>
    </Provider>,
  );

  await screen.findByPlaceholderText("Search settings...");
  return { user };
}

async function clickTreeItemByLabel(
  user: ReturnType<typeof userEvent.setup>,
  label: string,
) {
  const labels = await screen.findAllByText(label);
  const treeItemLabel = labels.find((element) =>
    element.closest('[role="treeitem"]'),
  );
  expect(treeItemLabel).toBeTruthy();
  await user.click(treeItemLabel as HTMLElement);
}

describe("Settings", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    cleanup();
  });

  it("renders core layout with General panel and action bar", async () => {
    await renderSettings();

    expect(screen.getByRole("button", { name: /setup wizard/i })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /general/i })).toBeInTheDocument();
    expect(
      screen.getByText(/update settings below and apply your changes/i),
    ).toBeInTheDocument();

    expect(screen.getByRole("button", { name: /^reset$/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /^apply$/i })).toBeDisabled();
  });

  it("filters settings tree from search input", async () => {
    const { user } = await renderSettings();

    await user.type(screen.getByPlaceholderText("Search settings..."), "telemetry");

    const treeItems = screen.getAllByRole("treeitem");
    const treeText = treeItems.map((item) => item.textContent ?? "").join(" ");

    expect(treeText).toMatch(/telemetry/i);
    expect(treeText).not.toMatch(/homeassistant/i);
  });

  it("selects Devices panel and persists selection", async () => {
    const { user } = await renderSettings();

    await clickTreeItemByLabel(user, "Devices");

    expect(screen.getByRole("heading", { name: /^devices$/i })).toBeInTheDocument();
    expect(localStorage.getItem("srat_settings_selected")).toBe("devices");
    expect(
      screen.getByRole("switch", { name: /multi channel mode/i }),
    ).toBeInTheDocument();
  });

  it("switches to app configuration and hides global bottom apply bar", async () => {
    const server = getMswServer();
    server.use(
      http.get(/.*\/api\/settings\/app-config$/, () =>
        HttpResponse.json({
          options: {
            auto_update: true,
            log_level: "debug",
          },
          runtime_config: {
            auto_update: true,
            log_level: "debug",
          },
          requires_restart: true,
        }),
      ),
      http.get(/.*\/api\/settings\/app-config\/schema$/, () =>
        HttpResponse.json({
          description: "Configure the current app.",
          long_description: "Schema-driven settings for the running app.",
          requires_restart: true,
          fields: [
            {
              name: "auto_update",
              constraint: "bool",
              description: "Auto update",
              optional: false,
            },
            {
              name: "log_level",
              constraint: "str",
              description: "Logging verbosity",
              optional: false,
              options: ["debug", "info"],
            },
          ],
        }),
      ),
    );

    const { user } = await renderSettings();

    await clickTreeItemByLabel(user, "App Configuration");

    expect(
      screen.getByRole("heading", { name: /app configuration/i }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^apply$/i })).toBeNull();
    expect(
      screen.getByRole("button", { name: /apply app configuration/i }),
    ).toBeInTheDocument();
  });

  it("hides rendered runtime configuration when runtime matches options", async () => {
    const sameConfig = {
      auto_update: true,
      log_level: "debug",
    };

    const server = getMswServer();
    server.use(
      http.get(/.*\/api\/settings$/, () =>
        HttpResponse.json({
          hostname: "homeassistant",
          workgroup: "WORKGROUP",
          telemetry_mode: "Disabled",
          bind_all_interfaces: true,
          interfaces: [],
        }),
      ),
      http.get(/.*\/api\/hostname$/, () => HttpResponse.json("homeassistant")),
      http.get(/.*\/api\/settings\/app-config$/, () =>
        HttpResponse.json({
          options: sameConfig,
          runtime_config: sameConfig,
          requires_restart: true,
        }),
      ),
      http.get(/.*\/api\/settings\/app-config\/schema$/, () =>
        HttpResponse.json({
          description: "Configure the current app.",
          long_description: "Schema-driven settings for the running app.",
          requires_restart: true,
          fields: [
            {
              name: "auto_update",
              constraint: "bool",
              description: "Auto update",
              optional: false,
            },
            {
              name: "log_level",
              constraint: "str",
              description: "Logging verbosity",
              optional: false,
              options: ["debug", "info"],
            },
          ],
        }),
      ),
    );

    const { user } = await renderSettings();
    await clickTreeItemByLabel(user, "App Configuration");

    expect(screen.queryByText(/rendered runtime configuration/i)).toBeNull();
  });

  it("renders aligned General switches with accessible names", async () => {
    await renderSettings();

    expect(
      screen.getByRole("switch", { name: /local master/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("switch", { name: /compatibility mode/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("switch", { name: /allow guest/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByLabelText(/smart mode/i),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("switch", { name: /experimental lab mode/i }),
    ).toBeInTheDocument();
  });

  it("shows Home Assistant lab features only when experimental lab mode is enabled", async () => {
    const server = getMswServer();
    server.use(
      http.get(/.*\/api\/settings$/, () =>
        HttpResponse.json({
          hostname: "homeassistant",
          workgroup: "WORKGROUP",
          telemetry_mode: "Disabled",
          bind_all_interfaces: true,
          interfaces: [],
          experimental_lab_mode: false,
        }),
      ),
      http.get(/.*\/api\/capabilities$/, () =>
        HttpResponse.json({
          support_nfs: true,
        }),
      ),
    );

    const { user } = await renderSettings();
    await clickTreeItemByLabel(user, "HomeAssistant");

    expect(screen.queryByRole("switch", { name: /use nfs for ha/i })).toBeNull();
    expect(screen.queryByText(/srat custom component/i)).toBeNull();

    server.use(
      http.get(/.*\/api\/settings\/homeassistant\/custom-component\/status$/, () =>
        HttpResponse.json({
          installed: false,
          latest_version: "1.2.3",
          restart_required: false,
        }),
      ),
    );

    await clickTreeItemByLabel(user, "General");
    await user.click(screen.getByRole("switch", { name: /experimental lab mode/i }));

    await clickTreeItemByLabel(user, "HomeAssistant");

    expect(screen.getByRole("switch", { name: /use nfs for ha/i })).toBeInTheDocument();
    expect(screen.getByText(/srat custom component/i)).toBeInTheDocument();
  });
});