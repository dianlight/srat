import { useState, useEffect } from 'react';
import {
    useGetSettingsQuery,
    useGetTelemetryInternetConnectionQuery,
    Telemetry_mode,
    type Settings,
} from '../store/sratApi';

/**
 * Hook to determine if the telemetry modal should be shown
 * @returns {boolean} true if modal should be shown
 */
export const useTelemetryModal = () => {
    const [shouldShow, setShouldShow] = useState(false);
    const [hasChecked, setHasChecked] = useState(false);

    const { data: settings, isLoading: isSettingsLoading } = useGetSettingsQuery();
    const { data: internetConnection, isLoading: isInternetLoading } = useGetTelemetryInternetConnectionQuery();

    useEffect(() => {
        // Don't check until both settings and internet connectivity are loaded
        if (isSettingsLoading || isInternetLoading || hasChecked) {
            return;
        }

        // Type guard to ensure settings is a Settings object and not an error
        const isValidSettings = (data: any): data is Settings => {
            return data && typeof data === 'object' && 'telemetry_mode' in data;
        };

        // Only show modal if:
        // 1. User has not yet chosen a telemetry preference (mode is "Ask")
        // 2. Internet connection is available
        // 3. Settings are loaded and valid
        if (
            isValidSettings(settings) &&
            settings.telemetry_mode === Telemetry_mode.Ask &&
            internetConnection === true
        ) {
            setShouldShow(true);
        }

        setHasChecked(true);
    }, [settings, internetConnection, isSettingsLoading, isInternetLoading, hasChecked]);

    return {
        shouldShow,
        dismiss: () => setShouldShow(false),
    };
};
