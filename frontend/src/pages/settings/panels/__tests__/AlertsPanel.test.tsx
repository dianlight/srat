import { render, screen } from "@testing-library/react";
import { FormProvider, useForm } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";
import {
  Status2,
  type LabFeature,
  type Settings as ApiSettings,
} from "../../../../store/sratApi";
import { AlertsPanel } from "../AlertsPanel";

vi.mock("../../../../store/sratApi", async () => {
  const actual = await vi.importActual<typeof import("../../../../store/sratApi")>(
    "../../../../store/sratApi",
  );
  return {
    ...actual,
    useGetApiLabFeaturesQuery: () => ({
      data: [
        {
          key: "ha_custom_component",
          name: "HA custom component",
          description: "",
          status: Status2.Alpha,
          available: true,
        },
      ] satisfies LabFeature[],
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
      experimental_lab_mode: false,
      alert_protected_mode: true,
      alert_addon_config_changed: true,
      alert_custom_component: true,
      ...defaultValues,
    } as ApiSettings,
  });

  return (
    <FormProvider {...methods}>
      <AlertsPanel readOnly={false} />
    </FormProvider>
  );
}

describe("AlertsPanel", () => {
  it("renders the protected mode and addon config switches", () => {
    render(<TestHarness />);
    expect(
      screen.getByRole("switch", { name: /protected mode alert/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("switch", { name: /add-on configuration changed alert/i }),
    ).toBeInTheDocument();
  });

  it("hides custom component alerts when lab mode is off", () => {
    render(<TestHarness />);
    expect(
      screen.queryByRole("switch", { name: /custom component alerts/i }),
    ).toBeNull();
  });

  it("shows custom component alerts when lab mode is on", () => {
    render(<TestHarness defaultValues={{ experimental_lab_mode: true }} />);
    expect(
      screen.getByRole("switch", { name: /custom component alerts/i }),
    ).toBeInTheDocument();
  });
});
