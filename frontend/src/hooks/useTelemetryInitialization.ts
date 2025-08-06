import { useEffect } from 'react';
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

    useEffect(() => {
        if (isLoading) return;

        // Type guard to ensure settings is a Settings object
        const isValidSettings = (data: any): data is Settings => {
            return data && typeof data === 'object' && 'telemetry_mode' in data;
        };

        if (isValidSettings(settings) && settings.telemetry_mode) {
            // Configure telemetry service with current settings
            telemetryService.configure(settings.telemetry_mode as TelemetryMode);

            // Report app initialization event if telemetry is enabled
            if (settings.telemetry_mode === Telemetry_mode.All) {
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
        }
    }, [settings, isLoading, reportEvent]);

    return {
        isLoading,
        telemetryMode: (settings as Settings)?.telemetry_mode,
    };
};
