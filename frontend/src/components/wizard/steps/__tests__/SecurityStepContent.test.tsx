import { render, screen } from "@testing-library/react";
import { useForm } from "react-hook-form";
import { FormContainer } from "react-hook-form-mui";
import { describe, expect, it } from "vitest";
import type { User } from "../../../../store/sratApi";
import { SecurityStepContent } from "../SecurityStepContent";

interface SecurityFormValues {
    hostname: string;
    workgroup: string;
    newPassword: string;
    confirmPassword: string;
}

function RenderStep({ adminUser, rootError }: { adminUser?: User; rootError?: string }) {
    const formContext = useForm<SecurityFormValues>({
        defaultValues: {
            hostname: "",
            workgroup: "",
            newPassword: "",
            confirmPassword: "",
        },
    });

    return (
        <FormContainer formContext={formContext} onSuccess={() => {}}>
            <SecurityStepContent adminUser={adminUser} rootError={rootError} />
        </FormContainer>
    );
}

const adminUserWithDefaultPassword: User = {
    username: "admin",
    is_admin: true,
    has_default_password: true,
};

describe("SecurityStepContent", () => {
    it("shows a warning when the admin user still has the default password", () => {
        render(<RenderStep adminUser={adminUserWithDefaultPassword} />);

        expect(screen.getByText(/default password/i)).toBeTruthy();
        expect(screen.getByText(/changeme!/)).toBeTruthy();
    });

    it("does not show the default-password warning when the password was changed", () => {
        const adminUser: User = {
            username: "admin",
            is_admin: true,
            has_default_password: false,
        };
        render(<RenderStep adminUser={adminUser} />);

        expect(screen.queryByText(/default password/i)).toBeNull();
    });

    it("does not show the default-password warning when no admin user is available", () => {
        render(<RenderStep />);

        expect(screen.queryByText(/default password/i)).toBeNull();
    });

    it("shows the root error alert when provided", () => {
        render(<RenderStep rootError="Failed to save the configuration" />);

        expect(screen.getByText("Failed to save the configuration")).toBeTruthy();
    });
});
