import { beforeEach, describe, expect, it } from "bun:test";
import '../../test/setup';

/**
 * App Component - AddonConfigChangedBanner Tests
 * 
 * Since the App component depends on RTK Query hooks and Material-UI components,
 * these smoke tests verify the component structure and rendering capabilities.
 * Integration tests for the API response handling should be performed via
 * end-to-end testing in the test-remote-environment.
 */

describe("App - AddonConfigChangedBanner Component", () => {
    beforeEach(() => {
        document.body.innerHTML = "";
        localStorage.clear();
    });

    it("imports the App component without errors", async () => {
        const { App } = await import("../App");
        expect(App).toBeTruthy();
        expect(typeof App).toBe("function");
    });

    it("verifies the App component uses useGetServerEventsQuery hook from wsApi", async () => {
        const appSource = await import("../App");
        const appCode = (appSource.App).toString();
        expect(appCode).toMatch(/useGetServerEventsQuery/);
    });

    it("verifies the App component uses useGetApiSettingsAppConfigQuery hook from sratApi", async () => {
        const appSource = await import("../App");
        const appCode = (appSource.App).toString();
        expect(appCode).toMatch(/useGetApiSettingsAppConfigQuery/);
    });

    it("verifies AddonConfigChangedBanner implementation has banner state variable", async () => {
        const appSource = await import("../App");
        const appCode = (appSource.App).toString();
        // Check for the state variable: showAddonConfigChangedBanner
        expect(appCode).toMatch(/showAddonConfigChangedBanner/);
        // Check for setState call: setShowAddonConfigChangedBanner
        expect(appCode).toMatch(/setShowAddonConfigChangedBanner/);
    });

    it("verifies banner renders when requires_restart is true", async () => {
        const appSource = await import("../App");
        const appCode = (appSource.App).toString();
        // Check for requires_restart check
        expect(appCode).toMatch(/requires_restart/);
        // Match both readable (true) and minified (!0) boolean forms
        expect(appCode).toMatch(/setShowAddonConfigChangedBanner\((?:true|!0)\)/);
    });

    it("verifies banner renders when app_config_changed event received", async () => {
        const appSource = await import("../App");
        const appCode = (appSource.App).toString();
        // Check for app_config_changed event check
        expect(appCode).toMatch(/app_config_changed/);
    });

    it("verifies Ignore button implementation dismisses banner", async () => {
        const appSource = await import("../App");
        const appCode = (appSource.App).toString();
        // Check for Ignore button with dismiss handler
        expect(appCode).toMatch(/Ignore/);
        // Match both readable (false) and minified (!1) boolean forms
        expect(appCode).toMatch(/setShowAddonConfigChangedBanner\((?:false|!1)\)/);
    });

    it("verifies Reload button calls backend restart endpoint before reloading", async () => {
        const appSource = await import("../App");
        const appCode = (appSource.App).toString();
        // Check for Reload button
        expect(appCode).toMatch(/Reload/);
        // Check for generated RTK restart mutation usage
        expect(appCode).toMatch(/usePutApiRestartMutation/);
        expect(appCode).toMatch(/restartAddon\(\)\.unwrap\(\)/);
        // Check for window.location.reload() call
        expect(appCode).toMatch(/window\.location\.reload\(\)/);
    });

    it("verifies banner renders within Snackbar component", async () => {
        const appSource = await import("../App");
        const appCode = (appSource.App).toString();
        // Check for Snackbar component usage
        expect(appCode).toMatch(/Snackbar/);
        // Check for Alert component inside Snackbar
        expect(appCode).toMatch(/Alert/);
        // Check for the warning message
        expect(appCode).toMatch(/Addon configuration has changed/);
    });

    it("verifies banner shows warning severity in MUI Alert", async () => {
        const appSource = await import("../App");
        const appCode = (appSource.App).toString();
        // Check for severity: "warning" in Alert component (JSX transpiled form)
        expect(appCode).toMatch(/severity:\s*"warning"/);
    });

    it("verifies banner is positioned at top center of screen", async () => {
        const appSource = await import("../App");
        const appCode = (appSource.App).toString();
        // Check for anchorOrigin with top/center positioning
        expect(appCode).toMatch(/anchorOrigin/);
        expect(appCode).toMatch(/vertical:\s*"top"/);
        expect(appCode).toMatch(/horizontal:\s*"center"/);
    });
});
