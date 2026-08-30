import { render, screen } from "@testing-library/react";
import { FormProvider, useForm } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";
import { Status2, type LabFeature, type Settings as ApiSettings } from "../../../../store/sratApi";
import { HomeAssistantPanel } from "../HomeAssistantPanel";

const mockState = vi.hoisted(() => {
  let componentConnected = false;
  return {
    getConnected: () => componentConnected,
    setConnected: (value: boolean) => {
      componentConnected = value;
    },
  };
});

// Suppress RTK Query network calls — we only care about disabled state
vi.mock("../../../../store/sratApi", async () => {
  const actual = await vi.importActual<typeof import("../../../../store/sratApi")>(
    "../../../../store/sratApi",
  );
  return {
    ...actual,
    useGetApiCapabilitiesQuery: () => ({
      data: {
        lib_smart_available: false,
      },
      isLoading: false,
    }),
    useGetApiSettingsHomeassistantCustomComponentStatusQuery: () => ({
      data: { connected: mockState.getConnected() },
      isLoading: false,
    }),
    useGetApiLabFeaturesQuery: () => ({
      data: [
        { key: "ha_custom_component", name: "HA custom component", description: "", status: Status2.Alpha, available: true },
      ] satisfies LabFeature[],
      isLoading: false,
    }),
  };
});

vi.mock("../../HomeAssistantCustomComponentPanel", () => ({
  HomeAssistantCustomComponentPanel: () => <div>Custom Component Panel</div>,
}));

function TestHarness({
  readOnly = false,
  defaultValues = {},
}: {
  readOnly?: boolean;
  defaultValues?: Partial<ApiSettings>;
}) {
  const methods = useForm<ApiSettings>({
    defaultValues: {
      mdns_registration: false,
      experimental_lab_mode: false,
      use_component_mdns_proxy: true,
      ...defaultValues,
    } as ApiSettings,
  });

  return (
    <FormProvider {...methods}>
      <HomeAssistantPanel readOnly={readOnly} />
    </FormProvider>
  );
}

describe("HomeAssistantPanel – mDNS proxy toggle", () => {
  it("renders the Use Home Assistant mDNS Proxy switch", () => {
    render(<TestHarness />);
    expect(
      screen.getByRole("switch", { name: /Use Home Assistant mDNS Proxy/i }),
    ).toBeInTheDocument();
  });

  it("disables the proxy switch when the HA component is not connected", () => {
    render(<TestHarness />);
    const toggle = screen.getByRole("switch", {
      name: /Use Home Assistant mDNS Proxy/i,
    });
    expect((toggle as HTMLInputElement).disabled).toBe(true);
  });

  it("disables the proxy switch when readOnly is true", () => {
    render(<TestHarness readOnly={true} />);
    const toggle = screen.getByRole("switch", {
      name: /Use Home Assistant mDNS Proxy/i,
    });
    expect((toggle as HTMLInputElement).disabled).toBe(true);
  });

  it("disables the proxy switch when the master mDNS switch is off", () => {
    mockState.setConnected(true);
    render(<TestHarness defaultValues={{ mdns_registration: false }} />);
    const toggle = screen.getByRole("switch", {
      name: /Use Home Assistant mDNS Proxy/i,
    });
    expect((toggle as HTMLInputElement).disabled).toBe(true);
    mockState.setConnected(false);
  });

  it("enables the proxy switch when the master switch is on and connected", () => {
    mockState.setConnected(true);
    render(<TestHarness defaultValues={{ mdns_registration: true }} />);
    const toggle = screen.getByRole("switch", {
      name: /Use Home Assistant mDNS Proxy/i,
    });
    expect((toggle as HTMLInputElement).disabled).toBe(false);
    mockState.setConnected(false);
  });
});
