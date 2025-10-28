import "/workspaces/srat/frontend/test/setup.ts";
import { describe, it, expect, beforeEach } from "bun:test";

// localStorage shim for testing
if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
        setItem: (k: string, v: string) => {
            _store[k] = String(v);
        },
        removeItem: (k: string) => {
            delete _store[k];
        },
        clear: () => {
            for (const k of Object.keys(_store)) delete _store[k];
        },
    };
}

describe("SmartStatusPanel Component", () => {
    beforeEach(() => {
        localStorage.clear();
    });

    // Helper to expand the SMART panel
    async function expandSmartPanel() {
        const { screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const expandButtons = await screen.findAllByRole("button");
        // The expand button is usually the last one or has aria-expanded
        const expandBtn = expandButtons.find((btn) => btn.getAttribute("aria-expanded") === "false");
        if (expandBtn) {
            const user = userEvent.setup();
            await user.click(expandBtn as any);
        }
    }

    it("should not render when smartInfo is undefined", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { SmartStatusPanel } = await import("../SmartStatusPanel");

        const { container } = render(
            React.createElement(SmartStatusPanel, {
                smartInfo: undefined,
                isSmartSupported: false,
            }),
        );

        // Component should return null, so container should be empty
        expect(container.firstChild).toBeFalsy();
    });

    it("should render SMART status when data is available", async () => {
        const React = await import("react");
        const { render, screen, act } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { SmartStatusPanel } = await import("../SmartStatusPanel");

        const mockSmartInfo = {
            disk_type: "SATA",
            temperature: { value: 45, min: 20, max: 80 },
            power_on_hours: { value: 10000 },
            power_cycle_count: { value: 500 },
            enabled: true,
        } as any;

        render(
            React.createElement(SmartStatusPanel, {
                smartInfo: mockSmartInfo,
                isSmartSupported: true,
                isReadOnlyMode: false,
            }),
        );

        // Should display SMART status header
        const header = await screen.findByText("S.M.A.R.T. Status");
        expect(header).toBeTruthy();

        // Expand the panel to see content
        const expandButtons = await screen.findAllByRole("button");
        const expandBtn = expandButtons.find((btn) => btn.getAttribute("aria-expanded") === "false");
        if (expandBtn) {
            const user = userEvent.setup();
            await act(async () => {
                await user.click(expandBtn as any);
            });
        }

        // Should display temperature
        const tempText = await screen.findByText(/45°C/);
        expect(tempText).toBeTruthy();

        // Should display power statistics
        const powerOnHoursText = await screen.findByText(/10,000 hours/);
        expect(powerOnHoursText).toBeTruthy();
    });

    it("should display health status as healthy", async () => {
        const React = await import("react");
        const { render, screen, act } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { SmartStatusPanel } = await import("../SmartStatusPanel");

        const mockSmartInfo = {
            disk_type: "SATA",
            temperature: { value: 45, min: 20, max: 80 },
            power_on_hours: { value: 10000 },
            power_cycle_count: { value: 500 },
            enabled: true,
        } as any;

        const mockHealthStatus = {
            passed: true,
            overall_status: "healthy",
        } as any;

        render(
            React.createElement(SmartStatusPanel, {
                smartInfo: mockSmartInfo,
                healthStatus: mockHealthStatus,
                isSmartSupported: true,
                isReadOnlyMode: false,
            }),
        );

        // Expand the panel to see content
        const expandButtons = await screen.findAllByRole("button");
        const expandBtn = expandButtons.find((btn) => btn.getAttribute("aria-expanded") === "false");
        if (expandBtn) {
            const user = userEvent.setup();
            await act(async () => {
                await user.click(expandBtn as any);
            });
        }

        // Should display healthy status
        const healthChip = await screen.findByText("All attributes healthy");
        expect(healthChip).toBeTruthy();
    });

    it("should display failing attributes when health check fails", async () => {
        const React = await import("react");
        const { render, screen, act } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { SmartStatusPanel } = await import("../SmartStatusPanel");

        const mockSmartInfo = {
            disk_type: "SATA",
            temperature: { value: 45, min: 20, max: 80 },
            power_on_hours: { value: 10000 },
            power_cycle_count: { value: 500 },
            enabled: true,
        } as any;

        const mockHealthStatus = {
            passed: false,
            failing_attributes: ["Reallocated_Sector_Ct", "Current_Pending_Sector"],
            overall_status: "failing",
        } as any;

        render(
            React.createElement(SmartStatusPanel, {
                smartInfo: mockSmartInfo,
                healthStatus: mockHealthStatus,
                isSmartSupported: true,
                isReadOnlyMode: false,
            }),
        );

        // Expand the panel to see content
        const expandButtons = await screen.findAllByRole("button");
        const expandBtn = expandButtons.find((btn) => btn.getAttribute("aria-expanded") === "false");
        if (expandBtn) {
            const user = userEvent.setup();
            await act(async () => {
                await user.click(expandBtn as any);
            });
        }

        // Should display failing status
        const issuesText = await screen.findByText("Issues detected");
        expect(issuesText).toBeTruthy();

        // Should display failing attributes
        const failingAttr1 = await screen.findByText("Reallocated_Sector_Ct");
        expect(failingAttr1).toBeTruthy();

        const failingAttr2 = await screen.findByText("Current_Pending_Sector");
        expect(failingAttr2).toBeTruthy();
    });

    it("should display test status when running", async () => {
        const React = await import("react");
        const { render, screen, act } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { SmartStatusPanel } = await import("../SmartStatusPanel");

        const mockSmartInfo = {
            disk_type: "SATA",
            temperature: { value: 45, min: 20, max: 80 },
            power_on_hours: { value: 10000 },
            power_cycle_count: { value: 500 },
            enabled: true,
        } as any;

        const mockTestStatus = {
            status: "running",
            test_type: "short",
            percent_complete: 50,
        } as any;

        render(
            React.createElement(SmartStatusPanel, {
                smartInfo: mockSmartInfo,
                testStatus: mockTestStatus,
                isSmartSupported: true,
                isReadOnlyMode: false,
            }),
        );

        // Expand the panel to see content
        const expandButtons = await screen.findAllByRole("button");
        const expandBtn = expandButtons.find((btn) => btn.getAttribute("aria-expanded") === "false");
        if (expandBtn) {
            const user = userEvent.setup();
            await act(async () => {
                await user.click(expandBtn as any);
            });
        }

        // Should display running status
        const runningText = await screen.findByText("running");
        expect(runningText).toBeTruthy();

        // Should display progress percentage
        const progressText = await screen.findByText("50%");
        expect(progressText).toBeTruthy();
    });

    it("should disable test actions when test is running", async () => {
        const React = await import("react");
        const { render, screen, act, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { SmartStatusPanel } = await import("../SmartStatusPanel");

        const mockSmartInfo = {
            disk_type: "SATA",
            temperature: { value: 45, min: 20, max: 80 },
            power_on_hours: { value: 10000 },
            power_cycle_count: { value: 500 },
            enabled: true,
        } as any;

        const mockTestStatus = {
            status: "running",
            test_type: "short",
            percent_complete: 50,
        } as any;

        const { container } = render(
            React.createElement(SmartStatusPanel, {
                smartInfo: mockSmartInfo,
                testStatus: mockTestStatus,
                isSmartSupported: true,
                isReadOnlyMode: false,
            }),
        );

        // Expand the panel to see content
        const expandButtons = await screen.findAllByRole("button");
        const expandBtn = expandButtons.find((btn) => btn.getAttribute("aria-expanded") === "false");
        if (expandBtn) {
            const user = userEvent.setup();
            await act(async () => {
                await user.click(expandBtn as any);
            });
        }

        // Start Test button should be disabled
        const startButtons = await within(container).findAllByText("Start Test");
        expect(startButtons).toHaveLength(1);
        expect((startButtons[0] as HTMLButtonElement).disabled).toBe(true);
    });

    it("should disable all actions in read-only mode", async () => {
        const React = await import("react");
        const { render, screen, act, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { SmartStatusPanel } = await import("../SmartStatusPanel");

        const mockSmartInfo = {
            disk_type: "SATA",
            temperature: { value: 45, min: 20, max: 80 },
            power_on_hours: { value: 10000 },
            power_cycle_count: { value: 500 },
            enabled: true,
        } as any;

        const { container } = render(
            React.createElement(SmartStatusPanel, {
                smartInfo: mockSmartInfo,
                isSmartSupported: true,
                isReadOnlyMode: true,
            }),
        );

        // Expand the panel to see content
        const expandButtons = await screen.findAllByRole("button");
        const expandBtn = expandButtons.find((btn) => btn.getAttribute("aria-expanded") === "false");
        if (expandBtn) {
            const user = userEvent.setup();
            await act(async () => {
                await user.click(expandBtn as any);
            });
        }

        // All action buttons should be disabled
        const startButtons = await within(container).findAllByText("Start Test");
        expect(startButtons).toHaveLength(1);
        expect((startButtons[0] as HTMLButtonElement).disabled).toBe(true);

        const enableButtons = await within(container).findAllByText("Enable SMART");
        expect(enableButtons).toHaveLength(1);
        expect((enableButtons[0] as HTMLButtonElement).disabled).toBe(true);

        const disableButtons = await within(container).findAllByText("Disable SMART");
        expect(disableButtons).toHaveLength(1);
        expect((disableButtons[0] as HTMLButtonElement).disabled).toBe(true);
    });

    it("should disable all actions when SMART is not supported", async () => {
        const React = await import("react");
        const { render, within } = await import("@testing-library/react");
        const { SmartStatusPanel } = await import("../SmartStatusPanel");

        const mockSmartInfo = {
            disk_type: "SATA",
            temperature: { value: 45, min: 20, max: 80 },
            power_on_hours: { value: 10000 },
            power_cycle_count: { value: 500 },
            enabled: true,
        } as any;

        const { container } = render(
            React.createElement(SmartStatusPanel, {
                smartInfo: mockSmartInfo,
                isSmartSupported: false,
                isReadOnlyMode: false,
            }),
        );

        // Component should not render when SMART is not supported
        const header = within(container).queryByText("S.M.A.R.T. Status");
        expect(header).toBeFalsy();
    });

    it("should call onStartTest when Start Test button is clicked", async () => {
        const React = await import("react");
        const { render, screen, act, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { SmartStatusPanel } = await import("../SmartStatusPanel");

        const mockSmartInfo = {
            disk_type: "SATA",
            temperature: { value: 45, min: 20, max: 80 },
            power_on_hours: { value: 10000 },
            power_cycle_count: { value: 500 },
            enabled: true,
        } as any;

        let startTestCalled = false;

        const { container } = render(
            React.createElement(SmartStatusPanel, {
                smartInfo: mockSmartInfo,
                isSmartSupported: true,
                isReadOnlyMode: false,
                onStartTest: () => {
                    startTestCalled = true;
                },
            }),
        );

        // Expand the panel to see content
        const expandButtons = await screen.findAllByRole("button");
        const expandBtn = expandButtons.find((btn) => btn.getAttribute("aria-expanded") === "false");
        if (expandBtn) {
            const user = userEvent.setup();
            await act(async () => {
                await user.click(expandBtn as any);
            });
        }

        // Click Start Test button
        const startButtons = await within(container).findAllByText("Start Test");
        expect(startButtons).toHaveLength(1);
        const user = userEvent.setup();
        await act(async () => {
            await user.click(startButtons[0]! as any);
        });

        // Dialog should open
        const dialogTitle = await screen.findByText("Start SMART Self-Test");
        expect(dialogTitle).toBeTruthy();

        // The callback will be called when the Start button in the dialog is clicked
        expect(startTestCalled).toBe(false); // Not called yet, dialog just opened
    });

    it("should display temperature range information", async () => {
        const React = await import("react");
        const { render, screen, act, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { SmartStatusPanel } = await import("../SmartStatusPanel");

        const mockSmartInfo = {
            disk_type: "SATA",
            temperature: { value: 45, min: 20, max: 80, overtemp_counter: 0 },
            power_on_hours: { value: 10000 },
            power_cycle_count: { value: 500 },
            enabled: true,
        } as any;

        const { container } = render(
            React.createElement(SmartStatusPanel, {
                smartInfo: mockSmartInfo,
                isSmartSupported: true,
                isReadOnlyMode: false,
            }),
        );

        // Expand the panel to see content
        const expandButtons = await screen.findAllByRole("button");
        const expandBtn = expandButtons.find((btn) => btn.getAttribute("aria-expanded") === "false");
        if (expandBtn) {
            const user = userEvent.setup();
            await act(async () => {
                await user.click(expandBtn as any);
            });
        }

        // Should display temperature range
        const tempRanges = await within(container).findAllByText(/Min: 20°C \/ Max: 80°C/);
        expect(tempRanges).toHaveLength(1);
        expect(tempRanges[0]).toBeTruthy();
    });
});
