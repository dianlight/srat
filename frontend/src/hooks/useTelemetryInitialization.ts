import { useEffect, useRef } from 'react';
import { useGetSettingsQuery, type Settings, Telemetry_mode } from '../store/sratApi';
import telemetryService, { type TelemetryMode } from '../services/telemetryService';
import { useRollbarTelemetry } from './useRollbarTelemetry';
import packageJson from '../../package.json';

/**
 * Hook to initialize and configure telemetry service based on user settings
 */
export const useTelemetryInitialization = () => {
    const { data: settings, isLoading } = useGetSettingsQuery();
    const { reportEvent } = useRollbarTelemetry();
    const previousTelemetryMode = useRef<Telemetry_mode | undefined>(undefined);
    const hasInitialized = useRef(false);

    useEffect(() => {
        if (isLoading) return;

        // Type guard to ensure settings is a Settings object
        const isValidSettings = (data: any): data is Settings => {
            return data && typeof data === 'object' && 'telemetry_mode' in data;
        };

        if (isValidSettings(settings) && settings.telemetry_mode) {
            const currentTelemetryMode = settings.telemetry_mode;

            // Configure telemetry service with current settings
            telemetryService.configure(currentTelemetryMode as TelemetryMode);

            // Only send events on startup or when telemetry mode changes to All
            const isStartup = !hasInitialized.current;
            const changedToAll = previousTelemetryMode.current !== Telemetry_mode.All &&
                currentTelemetryMode === Telemetry_mode.All;

            if (currentTelemetryMode === Telemetry_mode.All && (isStartup || changedToAll)) {
                // Send a test event to validate configuration
                reportEvent('telemetry_enabled', {
                    version: packageJson.version,
                    environment: process.env.NODE_ENV || 'development',
                });

                reportEvent('app_initialized', {
                    version: packageJson.version, // Use the actual package version
                    timestamp: new Date().toISOString(),
                    userAgent: navigator.userAgent,
                });
            }

            // Update tracking refs
            previousTelemetryMode.current = currentTelemetryMode;
            hasInitialized.current = true;
        }
    }, [settings, isLoading, reportEvent]);

    return {
        isLoading,
        telemetryMode: (settings as Settings)?.telemetry_mode,
    };
};
