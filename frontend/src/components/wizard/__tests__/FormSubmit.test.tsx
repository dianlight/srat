import { expect, it } from "vitest";

it("submits form with MUI Button type=submit inside Dialog + multiple watches", async () => {
    const React = await import("react");
    const { useEffect } = React;
    const { render, screen, waitFor } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { http, HttpResponse } = await import("msw");
    const { getMswServer } = await import("/test/testing");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { useForm } = await import("react-hook-form");
    const { FormContainer, TextFieldElement, PasswordElement } = await import("react-hook-form-mui");
    const { Dialog, DialogContent, DialogActions, Button } = await import("@mui/material");
    const { store } = await import("../../../store/store");
    const { useGetApiSettingsQuery, useGetApiHostnameQuery, useGetApiUsersQuery } = await import("../../../store/sratApi");
    
    const server = getMswServer();
    server.use(
        http.get("/api/settings", () => HttpResponse.json({ hostname: "mynas", workgroup: "WORKGROUP", telemetry_mode: "Disabled" })),
        http.get("/api/hostname", () => HttpResponse.json("mynas")),
        http.get("/api/users", () => HttpResponse.json([{ is_admin: true, password: "safepassword", name: "admin" }])),
        http.get("/api/nics", () => HttpResponse.json([{ name: "eth0", addrs: [], flags: [], hardwareAddr: "", index: 0, mtu: 1500 }])),
        http.get("/api/volumes", () => HttpResponse.json([]))
    );
    
    let submitted = false;
    
    function TestForm() {
        const { data: settings } = useGetApiSettingsQuery();
        const { data: systemHostname, isLoading: isHostnameFetching } = useGetApiHostnameQuery();
        const { data: users } = useGetApiUsersQuery();
        
        const securityForm = useForm({ defaultValues: { hostname: "", workgroup: "WORKGROUP", newPassword: "", confirmPassword: "" } });
        const networkForm = useForm({ defaultValues: { bind_all_interfaces: true, interfaces: [] as string[] } });
        const firstShareForm = useForm({ defaultValues: { partitionId: "", shareName: "" } });
        const telemetryForm = useForm({ defaultValues: { telemetry_mode: "" } });
        
        // simulate all the watch() calls in SetupWizard
        const bindAll = networkForm.watch("bind_all_interfaces");
        const partitionId = firstShareForm.watch("partitionId");
        const shareName = firstShareForm.watch("shareName");
        
        const adminUser = Array.isArray(users) ? users.find((u: any) => u.is_admin) : undefined;
        
        useEffect(() => {
            const hostname = !isHostnameFetching && systemHostname
                ? (systemHostname as string)
                : (settings as any)?.hostname ?? "";
            securityForm.reset({ hostname, workgroup: (settings as any)?.workgroup ?? "WORKGROUP", newPassword: "", confirmPassword: "" });
            networkForm.reset({ bind_all_interfaces: true, interfaces: [] });
            firstShareForm.reset({ partitionId: "", shareName: "" });
            telemetryForm.reset({ telemetry_mode: (settings as any)?.telemetry_mode ?? "Disabled" });
        }, [settings, systemHostname, isHostnameFetching, securityForm, networkForm, firstShareForm, telemetryForm]);
        
        return React.createElement(Dialog, { open: true },
            React.createElement("div", null, `bindAll: ${bindAll}, partId: ${partitionId}, share: ${shareName}`),
            React.createElement(FormContainer as any, { formContext: securityForm, onSuccess: (data: any) => { console.log("submit!", data); submitted = true; } },
                React.createElement(DialogContent, null,
                    React.createElement(TextFieldElement as any, { name: "hostname", label: "Hostname", rules: { required: true, minLength: 2 } }),
                    React.createElement(TextFieldElement as any, { name: "workgroup", label: "Workgroup", rules: { required: true, minLength: 2 } }),
                    React.createElement(PasswordElement as any, { name: "newPassword", label: "New Password", rules: {
                        validate: (value: string) => {
                            if (!(adminUser as any)?.password && !value) return "Password is required";
                            if (!value) return true;
                            if (value === "changeme!") return "Cannot use the default password";
                            if (value.length < 6) return "At least 6 characters";
                            return true;
                        }
                    }}),
                    React.createElement(PasswordElement as any, { name: "confirmPassword", label: "Confirm Password", rules: {
                        validate: (value: string, formValues: any) =>
                            !formValues.newPassword || value === formValues.newPassword || "Passwords do not match"
                    }})
                ),
                React.createElement(DialogActions, null,
                    React.createElement(Button, { type: "submit", variant: "contained", disabled: securityForm.formState.isSubmitting }, "Next")
                )
            )
        );
    }
    
    render(React.createElement(Provider as any, { store }, React.createElement(TestForm)));
    
    const input = await screen.findByLabelText("Hostname");
    await waitFor(() => {
        expect((input as HTMLInputElement).value.length).toBeGreaterThan(0);
        expect(screen.queryByText(/password is required/i)).toBeNull();
    });
    
    const btn = screen.getByRole("button", { name: "Next" });
    const user = userEvent.setup();
    await user.click(btn);
    await waitFor(() => expect(submitted).toBe(true), { timeout: 5000 });
});
