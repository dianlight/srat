import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FormProvider, useForm } from "react-hook-form";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { Settings as ApiSettings } from "../../../../store/sratApi";
import { GeneralPanel } from "../GeneralPanel";

const mockState = vi.hoisted(() => ({
  capabilities: { lib_smart_available: false } as Record<string, unknown>,
}));

vi.mock("../../../../store/sratApi", async () => {
  const actual = await vi.importActual<
    typeof import("../../../../store/sratApi")
  >("../../../../store/sratApi");

  return {
    ...actual,
    useGetApiHostnameQuery: () => ({
      data: "srat-host",
      isLoading: false,
      refetch: () => ({
        unwrap: async () => "srat-host",
      }),
    }),
    useGetApiCapabilitiesQuery: () => ({
      data: mockState.capabilities,
      isLoading: false,
    }),
  };
});

function TestHarness({
  defaultValues = {},
}: {
  defaultValues?: Partial<ApiSettings>;
}) {
  const methods = useForm<ApiSettings>({
    defaultValues: {
      hostname: "srat-host",
      workgroup: "WORKGROUP",
      local_master: false,
      compatibility_mode: false,
      allow_guest: false,
      smart_on: true,
      experimental_lab_mode: false,
      mdns_registration: false,
      ...defaultValues,
    } as ApiSettings,
  });

  return (
    <FormProvider {...methods}>
      <GeneralPanel readOnly={false} />
    </FormProvider>
  );
}

describe("GeneralPanel switch accessibility", () => {
  beforeEach(() => {
    mockState.capabilities = { lib_smart_available: false };
  });

  it("exposes core switches with semantic accessible names", () => {
    render(<TestHarness />);

    expect(
      screen.getByRole("switch", {
        name: /local master/i,
      }),
    ).toBeInTheDocument();

    expect(
      screen.getByRole("switch", {
        name: /compatibility mode/i,
      }),
    ).toBeInTheDocument();

    expect(
      screen.getByRole("switch", {
        name: /allow guest/i,
      }),
    ).toBeInTheDocument();

    expect(
      screen.getByRole("switch", { name: /smart enabled/i }),
    ).toBeInTheDocument();
  });

  it("renders the Samba mDNS Announce switch", () => {
    render(<TestHarness />);

    const toggle = screen.getByRole("switch", {
      name: /Samba mDNS Announce/i,
    });
    expect(toggle).toBeInTheDocument();
    expect((toggle as HTMLInputElement).disabled).toBe(false);
  });

  it("keeps the Samba mDNS Announce switch enabled regardless of the proxy setting", () => {
    render(<TestHarness defaultValues={{ use_component_mdns_proxy: false }} />);

    const toggle = screen.getByRole("switch", {
      name: /Samba mDNS Announce/i,
    });
    expect((toggle as HTMLInputElement).disabled).toBe(false);
  });
});

describe("GeneralPanel SMART backend tooltip", () => {
  beforeEach(() => {
    mockState.capabilities = { lib_smart_available: false };
  });

  it("does not render the active backend as inline text", () => {
    render(<TestHarness />);
    expect(screen.queryByText(/Active backend:/)).toBeNull();
  });

  it("shows the active backend in the tooltip on hover (direct)", async () => {
    mockState.capabilities = {
      lib_smart_available: true,
      smart_backend: "direct",
    };
    const user = userEvent.setup();
    render(<TestHarness />);

    const toggle = screen.getByRole("switch", { name: /smart enabled/i });
    await user.hover(toggle);

    const tooltip = await screen.findByRole("tooltip");
    expect(tooltip.textContent).toMatch(/Active backend:/);
    expect(tooltip.textContent).toMatch(/direct \(lib backend\)/);
  });

  it("shows the active backend in the tooltip on hover (legacy)", async () => {
    mockState.capabilities = {
      lib_smart_available: false,
      smart_backend: "legacy",
    };
    const user = userEvent.setup();
    render(<TestHarness />);

    const toggle = screen.getByRole("switch", { name: /smart enabled/i });
    await user.hover(toggle);

    const tooltip = await screen.findByRole("tooltip");
    expect(tooltip.textContent).toMatch(/Active backend:/);
    expect(tooltip.textContent).toMatch(/legacy \(smartctl exec\)/);
  });
});
