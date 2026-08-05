import { render, screen } from "@testing-library/react";
import { FormProvider, useForm } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";
import type { Settings as ApiSettings } from "../../../../store/sratApi";
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
      addon_mdns_registration: false,
      ...defaultValues,
    } as ApiSettings,
  });

  return (
    <FormProvider {...methods}>
      <HomeAssistantPanel readOnly={readOnly} />
    </FormProvider>
  );
}

describe("HomeAssistantPanel – mDNS toggle", () => {
  it("renders the mDNS Registration switch", () => {
    render(<TestHarness />);
    expect(
      screen.getByRole("switch", { name: /mDNS Registration/i }),
    ).toBeInTheDocument();
  });

  it("disables the mDNS switch when the HA component is not connected", () => {
    render(<TestHarness />);
    const toggle = screen.getByRole("switch", { name: /mDNS Registration/i });
    expect((toggle as HTMLInputElement).disabled).toBe(true);
  });

  it("disables the mDNS switch when readOnly is true", () => {
    render(<TestHarness readOnly={true} />);
    const toggle = screen.getByRole("switch", { name: /mDNS Registration/i });
    expect((toggle as HTMLInputElement).disabled).toBe(true);
  });

  it("disables the HA mDNS switch when addon-side direct mDNS is enabled", () => {
    mockState.setConnected(true);
    render(
      <TestHarness defaultValues={{ addon_mdns_registration: true }} />,
    );
    const haToggle = screen.getByRole("switch", { name: /mDNS Registration/i });
    expect((haToggle as HTMLInputElement).disabled).toBe(true);
    mockState.setConnected(false);
  });
});
