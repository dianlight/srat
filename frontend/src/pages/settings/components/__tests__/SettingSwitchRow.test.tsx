import { render, screen } from "@testing-library/react";
import { FormProvider, useForm } from "react-hook-form";
import { describe, expect, it } from "vitest";
import { SettingSwitchRow } from "../SettingSwitchRow";

type FormValues = {
  local_master: boolean;
};

function TestHarness() {
  const methods = useForm<FormValues>({
    defaultValues: {
      local_master: false,
    },
  });

  return (
    <FormProvider {...methods}>
      <SettingSwitchRow
        ariaLabel="Local Master"
        control={methods.control}
        helperText="Participate in local master browser elections"
        label="Local Master"
        name="local_master"
      />
    </FormProvider>
  );
}

describe("SettingSwitchRow", () => {
  it("renders an accessible switch with helper text", () => {
    render(<TestHarness />);

    expect(
      screen.getByRole("switch", {
        name: /local master/i,
      }),
    ).toBeInTheDocument();

    expect(
      screen.getByText(/participate in local master browser elections/i),
    ).toBeInTheDocument();
  });
});
