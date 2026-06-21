import { render, screen } from "@testing-library/react";
import { FormProvider, useForm } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";
import type { Settings as ApiSettings } from "../../../../store/sratApi";
import { HomeAssistantPanel } from "../HomeAssistantPanel";

// Suppress RTK Query network calls — we only care about disabled state
vi.mock("../../../../store/sratApi", async () => {
  const actual = await vi.importActual<typeof import("../../../../store/sratApi")>(
    "../../../../store/sratApi",
  );
  return {
    ...actual,
    useGetApiCapabilitiesQuery: () => ({
      data: { lib_smart_available: false },
      isLoading: false,
    }),
    useGetApiSettingsHomeassistantCustomComponentStatusQuery: () => ({
      data: { connected: false },
      isLoading: false,
    }),
  };
});

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
});
