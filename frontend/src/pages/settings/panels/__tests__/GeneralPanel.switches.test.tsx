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
    useGetApiCapabilitiesQuery: () => ({
      data: { lib_smart_available: false },
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
      smart_mode: "legacy",
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
      screen.getByLabelText(/smart mode/i),
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
    render(
      <TestHarness defaultValues={{ use_component_mdns_proxy: false }} />,
    );

    const toggle = screen.getByRole("switch", {
      name: /Samba mDNS Announce/i,
    });
    expect((toggle as HTMLInputElement).disabled).toBe(false);
  });
});
