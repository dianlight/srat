import { beforeEach, describe, expect, it } from "bun:test";
import "../../../../test/setup";

describe("DashboardTourStep", () => {
    beforeEach(() => {
        // Clear any global state before each test
    });

    it("exports DashboardSteps array correctly", async () => {
        const { DashboardSteps } = await import("../DashboardTourStep");

        expect(Array.isArray(DashboardSteps)).toBe(true);
        expect(DashboardSteps.length).toBeGreaterThan(0);
    });

    it("has correct number of dashboard tour steps", async () => {
        const { DashboardSteps } = await import("../DashboardTourStep");

        // Should have 9 steps (step0 through step8)
        expect(DashboardSteps.length).toBe(9);
    });

    it("all steps have required selector property", async () => {
        const { DashboardSteps } = await import("../DashboardTourStep");

        DashboardSteps.forEach((step, index) => {
            expect(step.selector).toBeTruthy();
            expect(typeof step.selector).toBe("string");
            expect(step.selector).toContain(`reactour__tab`);
            expect(step.selector).toContain(`__step${index}`);
        });
    });

    it("all steps have content property", async () => {
        const { DashboardSteps } = await import("../DashboardTourStep");

        DashboardSteps.forEach((step) => {
            expect(step.content).toBeTruthy();
        });
    });

    it("welcome step (step0) has correct structure", async () => {
        const { DashboardSteps } = await import("../DashboardTourStep");

        const welcomeStep = DashboardSteps[0];
        if (welcomeStep) {
            expect(welcomeStep.selector).toContain("step0");
            expect(welcomeStep.content).toBeTruthy();
            // Welcome step should not have action function
            expect(welcomeStep.action).toBeUndefined();
        }
    });

    it("tab navigation step (step1) has correct structure", async () => {
        const { DashboardSteps } = await import("../DashboardTourStep");

        const tabStep = DashboardSteps[1];
        if (tabStep) {
            expect(tabStep.selector).toContain("step1");
            expect(tabStep.content).toBeTruthy();
            // Tab step should not have action function
            expect(tabStep.action).toBeUndefined();
        }
    });

    it("welcome and news step (step2) has correct structure and action", async () => {
        const { DashboardSteps } = await import("../DashboardTourStep");

        const welcomeNewsStep = DashboardSteps[2];
        if (welcomeNewsStep) {
            expect(welcomeNewsStep.selector).toContain("step2");
            expect(welcomeNewsStep.content).toBeTruthy();
            expect(welcomeNewsStep.position).toBe("center");
            expect(typeof welcomeNewsStep.action).toBe("function");
        }
    });

    it("actionable items step (step3) has correct structure with mutation observables", async () => {
        const { DashboardSteps } = await import("../DashboardTourStep");

        const actionableStep = DashboardSteps[3];
        if (actionableStep) {
            expect(actionableStep.selector).toContain("step3");
            expect(actionableStep.content).toBeTruthy();
            expect(Array.isArray(actionableStep.mutationObservables)).toBe(true);
            expect(actionableStep.mutationObservables?.[0]).toContain("step3");
            expect(typeof actionableStep.action).toBe("function");
        }
    });

    it("metrics overview step (step4) has correct structure and action", async () => {
        const { DashboardSteps } = await import("../DashboardTourStep");

        const metricsStep = DashboardSteps[4];
        if (metricsStep) {
            expect(metricsStep.selector).toContain("step4");
            expect(metricsStep.content).toBeTruthy();
            expect(typeof metricsStep.action).toBe("function");
        }
    });

    it("process metrics step (step5) has correct structure and action", async () => {
        const { DashboardSteps } = await import("../DashboardTourStep");

        const processStep = DashboardSteps[5];
        if (processStep) {
            expect(processStep.selector).toContain("step5");
            expect(processStep.content).toBeTruthy();
            expect(typeof processStep.action).toBe("function");
        }
    });

    it("disk health step (step6) has correct structure and action", async () => {
        const { DashboardSteps } = await import("../DashboardTourStep");

        const diskStep = DashboardSteps[6];
        if (diskStep) {
            expect(diskStep.selector).toContain("step6");
            expect(diskStep.content).toBeTruthy();
            expect(typeof diskStep.action).toBe("function");
        }
    });

    it("network health step (step7) has correct structure and action", async () => {
        const { DashboardSteps } = await import("../DashboardTourStep");

        const networkStep = DashboardSteps[7];
        if (networkStep) {
            expect(networkStep.selector).toContain("step7");
            expect(networkStep.content).toBeTruthy();
            expect(typeof networkStep.action).toBe("function");
        }
    });

    it("samba status step (step8) has correct structure and action", async () => {
        const { DashboardSteps } = await import("../DashboardTourStep");

        const sambaStep = DashboardSteps[8];
        if (sambaStep) {
            expect(sambaStep.selector).toContain("step8");
            expect(sambaStep.content).toBeTruthy();
            expect(typeof sambaStep.action).toBe("function");
        }
    });

    it("action functions can be called without errors", async () => {
        const { DashboardSteps } = await import("../DashboardTourStep");

        // Test action functions for steps that have them (steps 2-8)
        const stepsWithActions = DashboardSteps.slice(2);

        stepsWithActions.forEach((step) => {
            if (step.action) {
                // Mock element
                const mockElement = document.createElement('div');

                // Should not throw error when called
                expect(() => {
                    step.action!(mockElement);
                }).not.toThrow();
            }
        });
    });

    it("tour event types are properly imported and used", async () => {
        const { TourEventTypes } = await import("../../../utils/TourEvents");

        // Verify that TourEventTypes has the expected dashboard events
        expect(TourEventTypes.DASHBOARD_STEP_2).toBeTruthy();
        expect(TourEventTypes.DASHBOARD_STEP_3).toBeTruthy();
        expect(TourEventTypes.DASHBOARD_STEP_4).toBeTruthy();
        expect(TourEventTypes.DASHBOARD_STEP_5).toBeTruthy();
        expect(TourEventTypes.DASHBOARD_STEP_6).toBeTruthy();
        expect(TourEventTypes.DASHBOARD_STEP_7).toBeTruthy();
        expect(TourEventTypes.DASHBOARD_STEP_8).toBeTruthy();
    });

    it("tab IDs are properly imported and used in selectors", async () => {
        const { DashboardSteps } = await import("../DashboardTourStep");
        const { TabIDs } = await import("../../../store/locationState");

        // Verify TabIDs.DASHBOARD is used in all selectors
        DashboardSteps.forEach((step) => {
            expect(step.selector).toContain(`tab${TabIDs.DASHBOARD}`);
        });
    });

    it("material-ui components are properly imported", async () => {
        // This test verifies the imports work correctly
        const { Box, Divider, Typography } = await import("@mui/material");

        expect(Box).toBeTruthy();
        expect(Divider).toBeTruthy();
        expect(Typography).toBeTruthy();
    });

    it("reactour step type is properly imported", async () => {
        // This test verifies the StepType import works correctly
        const stepTypeModule = await import("@reactour/tour");

        expect(stepTypeModule).toBeTruthy();
        // StepType is a type, so we can't directly test it, but we can verify the module imports
    });

    it("step selectors use consistent data-tutor attribute pattern", async () => {
        const { DashboardSteps } = await import("../DashboardTourStep");

        DashboardSteps.forEach((step, index) => {
            expect(step.selector).toMatch(/\[data-tutor="reactour__tab\d+__step\d+"\]/);
            expect(step.selector).toContain(`step${index}`);
        });
    });

    it("steps with mutation observables have matching selectors", async () => {
        const { DashboardSteps } = await import("../DashboardTourStep");

        const stepWithMutationObservables = DashboardSteps.find(step => step.mutationObservables);

        if (stepWithMutationObservables) {
            expect(stepWithMutationObservables.mutationObservables?.[0]).toContain("step3");
            expect(stepWithMutationObservables.selector).toContain("step3");
        }
    });
});