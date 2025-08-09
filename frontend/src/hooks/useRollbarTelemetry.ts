import { useRollbar } from '@rollbar/react';
import telemetryService from '../services/telemetryService';

/**
 * Hook that provides Rollbar functionality with telemetry mode checking
 * This hook ensures that errors and events are only reported based on the current telemetry mode
 */
export const useRollbarTelemetry = () => {
    const rollbar = useRollbar();

    const reportError = (error: Error | string, extraData?: Record<string, any>) => {
        if (!telemetryService.getIsConfigured()) {
            return; // Silently ignore if not configured
        }

        const mode = telemetryService.getCurrentMode();
        if (mode === 'Disabled' || mode === 'Ask') {
            return; // Don't report if disabled or asking
        }

        // Report errors for both All and Errors modes
        if (mode === 'All' || mode === 'Errors') {
            rollbar.error(error, extraData);
            console.debug('Error reported to Rollbar:', error);
        }
    };

    const reportEvent = (event: string, data?: Record<string, any>) => {
        if (!telemetryService.getIsConfigured()) {
            return; // Silently ignore if not configured
        }

        const mode = telemetryService.getCurrentMode();
        // Only report events in All mode
        if (mode !== 'All') {
            return;
        }

        const eventData = {
            ...data,
            event_type: event,
            timestamp: new Date().toISOString(),
        };

        rollbar.info(`Event: ${event}`, eventData);
        console.debug('Event reported to Rollbar:', event, eventData);
    };

    return {
        reportError,
        reportEvent,
        isEnabled: telemetryService.isEnabled(),
        currentMode: telemetryService.getCurrentMode(),
    };
};
