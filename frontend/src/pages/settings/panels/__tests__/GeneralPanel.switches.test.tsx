import { render, screen } from "@testing-library/react";
import { FormProvider, useForm } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";
import type { Settings as ApiSettings } from "../../../../store/sratApi";
import { GeneralPanel } from "../GeneralPanel";

vi.mock("../../../../store/sratApi", async () => {
  const actual = await vi.importActual<typeof import("../../../../store/sratApi")>(
    "../../../../store/sratApi",
  );

  return {
    ...actual,
    useGetApiHostnameQuery: () => ({
      data: "srat-host",
      isLoading: false,
      refetch: () => ({
        unwrap: async () => "srat-host",
      }),
    }),
  };
});

function TestHarness() {
  const methods = useForm<ApiSettings>({
    defaultValues: {
      hostname: "srat-host",
      workgroup: "WORKGROUP",
      local_master: false,
      compatibility_mode: false,
      allow_guest: false,
      disable_smart: false,
    } as ApiSettings,
  });

  return (
    <FormProvider {...methods}>
      <GeneralPanel readOnly={false} />
    </FormProvider>
  );
}

describe("GeneralPanel switch accessibility", () => {
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
      screen.getByRole("switch", {
        name: /disable smart integration/i,
      }),
    ).toBeInTheDocument();
  });
});
