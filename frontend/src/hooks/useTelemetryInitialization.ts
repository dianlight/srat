import { useEffect } from 'react';
import { useGetSettingsQuery, type Settings, Telemetry_mode } from '../store/sratApi';
import telemetryService, { type TelemetryMode } from '../services/telemetryService';
import packageJson from '../../package.json';

/**
 * Hook to initialize and configure telemetry service based on user settings
 */
export const useTelemetryInitialization = () => {
    const { data: settings, isLoading } = useGetSettingsQuery();

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
                telemetryService.reportEvent('app_initialized', {
                    version: packageJson.version, // Use the actual package version
                    timestamp: new Date().toISOString(),
                    userAgent: navigator.userAgent,
                });
            }
        }
    }, [settings, isLoading]);

    return {
        isLoading,
        telemetryMode: (settings as Settings)?.telemetry_mode,
    };
};
