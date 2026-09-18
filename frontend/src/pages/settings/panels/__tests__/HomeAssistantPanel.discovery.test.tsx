import { render, screen } from "@testing-library/react";
import { FormProvider, useForm } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";
import { Status2, type LabFeature, type Settings as ApiSettings } from "../../../../store/sratApi";
import { HomeAssistantPanel } from "../HomeAssistantPanel";

// Suppress RTK Query network calls — we only care about switch rendering
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
      data: { connected: false },
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
      experimental_lab_mode: false,
      enable_ha_discovery: false,
      ...defaultValues,
    } as ApiSettings,
  });

  return (
    <FormProvider {...methods}>
      <HomeAssistantPanel readOnly={readOnly} />
    </FormProvider>
  );
}

describe("HomeAssistantPanel – HA discovery toggle", () => {
  it("renders the Advertise to Home Assistant switch", () => {
    render(<TestHarness />);
    expect(
      screen.getByRole("switch", { name: /Advertise to Home Assistant/i }),
    ).toBeInTheDocument();
  });

  it("disables the switch when readOnly is true", () => {
    render(<TestHarness readOnly={true} />);
    const toggle = screen.getByRole("switch", {
      name: /Advertise to Home Assistant/i,
    });
    expect((toggle as HTMLInputElement).disabled).toBe(true);
  });

  it("enables the switch when not readOnly", () => {
    render(<TestHarness />);
    const toggle = screen.getByRole("switch", {
      name: /Advertise to Home Assistant/i,
    });
    expect((toggle as HTMLInputElement).disabled).toBe(false);
  });
});
