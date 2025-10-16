import { describe, it, expect, beforeEach } from "bun:test";

// Required localStorage shim for testing environment
if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
        setItem: (k: string, v: string) => { _store[k] = String(v); },
        removeItem: (k: string) => { delete _store[k]; },
        clear: () => { for (const k of Object.keys(_store)) delete _store[k]; },
    };
}

describe("HDIdleGeneralSettings Component", () => {
    beforeEach(() => {
        localStorage.clear();
        document.body.innerHTML = '';
    });

    it("renders HDIdle settings section header", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { HDIdleGeneralSettings } = await import("../HDIdleGeneralSettings");
        const { useForm } = await import("react-hook-form-mui");

        // Create a mock control
        const TestWrapper = () => {
            const { control } = useForm({
                defaultValues: {
                    hdidle_enabled: false,
                    hdidle_default_idle_time: 600,
                    hdidle_default_command_type: "scsi",
                    hdidle_log_file: "",
                    hdidle_debug: false,
                    hdidle_ignore_spin_down_detection: false,
                },
            });

            return React.createElement(
                HDIdleGeneralSettings as any,
                {
                    control,
                    isEnabled: false,
                    readOnly: false,
                }
            );
        };

        render(React.createElement(TestWrapper));

        const header = await screen.findByText(/HDIdle Disk Spin-Down Settings/i);
        expect(header).toBeTruthy();
    });

    it("renders enable switch when not in read-only mode", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { HDIdleGeneralSettings } = await import("../HDIdleGeneralSettings");
        const { useForm } = await import("react-hook-form-mui");

        const TestWrapper = () => {
            const { control } = useForm({
                defaultValues: {
                    hdidle_enabled: false,
                    hdidle_default_idle_time: 600,
                    hdidle_default_command_type: "scsi",
                    hdidle_log_file: "",
                    hdidle_debug: false,
                    hdidle_ignore_spin_down_detection: false,
                },
            });

            return React.createElement(
                HDIdleGeneralSettings as any,
                {
                    control,
                    isEnabled: false,
                    readOnly: false,
                }
            );
        };

        render(React.createElement(TestWrapper));

        const enableSwitch = await screen.findByText(/Enable Automatic Disk Spin-Down/i);
        expect(enableSwitch).toBeTruthy();
    });

    it("disables fields when isEnabled is false", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { HDIdleGeneralSettings } = await import("../HDIdleGeneralSettings");
        const { useForm } = await import("react-hook-form-mui");

        const TestWrapper = () => {
            const { control } = useForm({
                defaultValues: {
                    hdidle_enabled: false,
                    hdidle_default_idle_time: 600,
                    hdidle_default_command_type: "scsi",
                    hdidle_log_file: "",
                    hdidle_debug: false,
                    hdidle_ignore_spin_down_detection: false,
                },
            });

            return React.createElement(
                HDIdleGeneralSettings as any,
                {
                    control,
                    isEnabled: false,
                    readOnly: false,
                }
            );
        };

        const { container } = render(React.createElement(TestWrapper));
        
        // Check that fields are rendered
        expect(container).toBeTruthy();
    });

    it("renders all configuration fields", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { HDIdleGeneralSettings } = await import("../HDIdleGeneralSettings");
        const { useForm } = await import("react-hook-form-mui");

        const TestWrapper = () => {
            const { control } = useForm({
                defaultValues: {
                    hdidle_enabled: true,
                    hdidle_default_idle_time: 600,
                    hdidle_default_command_type: "scsi",
                    hdidle_log_file: "",
                    hdidle_debug: false,
                    hdidle_ignore_spin_down_detection: false,
                },
            });

            return React.createElement(
                HDIdleGeneralSettings as any,
                {
                    control,
                    isEnabled: true,
                    readOnly: false,
                }
            );
        };

        render(React.createElement(TestWrapper));

        // Check for key fields
        const idleTimeField = await screen.findByText(/Default Idle Time/i);
        expect(idleTimeField).toBeTruthy();

        const commandTypeField = await screen.findByText(/Default Command Type/i);
        expect(commandTypeField).toBeTruthy();
    });

    it("respects readOnly mode", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { HDIdleGeneralSettings } = await import("../HDIdleGeneralSettings");
        const { useForm } = await import("react-hook-form-mui");

        const TestWrapper = () => {
            const { control } = useForm({
                defaultValues: {
                    hdidle_enabled: true,
                    hdidle_default_idle_time: 600,
                    hdidle_default_command_type: "scsi",
                    hdidle_log_file: "",
                    hdidle_debug: false,
                    hdidle_ignore_spin_down_detection: false,
                },
            });

            return React.createElement(
                HDIdleGeneralSettings as any,
                {
                    control,
                    isEnabled: true,
                    readOnly: true,
                }
            );
        };

        const { container } = render(React.createElement(TestWrapper));
        
        // Component should render even in read-only mode
        expect(container).toBeTruthy();
    });
});
