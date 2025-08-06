import Rollbar from 'rollbar';

// Telemetry modes matching the backend enum
export type TelemetryMode = 'Ask' | 'All' | 'Errors' | 'Disabled';

interface TelemetryConfig {
    accessToken: string;
    environment: string;
    version: string;
}

class TelemetryService {
    private rollbar: Rollbar | null = null;
    private mode: TelemetryMode = 'Ask';
    private config: TelemetryConfig;
    private isConfigured = false;

    constructor(config: TelemetryConfig) {
        this.config = config;
    }

    /**
     * Configure the telemetry service with the given mode
     */
    configure(mode: TelemetryMode): void {
        this.mode = mode;

        // Shutdown existing Rollbar instance
        if (this.rollbar) {
            // Rollbar doesn't have a shutdown method, but we can set it to null
            this.rollbar = null;
            this.isConfigured = false;
        }

        // Only initialize Rollbar if mode is All or Errors
        if (mode === 'All' || mode === 'Errors') {
            this.rollbar = new Rollbar({
                accessToken: this.config.accessToken,
                environment: this.config.environment,
                codeVersion: this.config.version,
                captureUncaught: mode === 'All' || mode === 'Errors', // Capture uncaught exceptions
                captureUnhandledRejections: mode === 'All' || mode === 'Errors', // Capture unhandled promises
                payload: {
                    client: {
                        javascript: {
                            code_version: this.config.version,
                            source_map_enabled: false,
                        }
                    }
                },
                // Rollbar is enabled since we're in All or Errors mode
                enabled: true,
            });

            this.isConfigured = true;
            console.log(`Rollbar telemetry configured: ${mode}`);

            // Send a test event if mode is All
            if (mode === 'All') {
                this.reportEvent('telemetry_enabled', {
                    version: this.config.version,
                    environment: this.config.environment,
                });
            }
        } else {
            console.log(`Rollbar telemetry disabled: ${mode}`);
        }
    }

    /**
     * Report an error to the telemetry service
     */
    reportError(error: Error | string, extraData?: Record<string, any>): void {
        if (!this.isConfigured || !this.rollbar) {
            return; // Silently ignore if not configured
        }

        if (this.mode === 'Disabled' || this.mode === 'Ask') {
            return; // Don't report if disabled or asking
        }

        // Report errors for both All and Errors modes
        if (this.mode === 'All' || this.mode === 'Errors') {
            this.rollbar.error(error, extraData);
            console.debug('Error reported to Rollbar:', error);
        }
    }

    /**
     * Report a telemetry event to the service
     */
    reportEvent(event: string, data?: Record<string, any>): void {
        if (!this.isConfigured || !this.rollbar) {
            return; // Silently ignore if not configured
        }

        // Only report events in All mode
        if (this.mode !== 'All') {
            return;
        }

        const eventData = {
            ...data,
            event_type: event,
            timestamp: new Date().toISOString(),
        };

        this.rollbar.info(`Event: ${event}`, eventData);
        console.debug('Event reported to Rollbar:', event, eventData);
    }

    /**
     * Get current telemetry mode
     */
    getCurrentMode(): TelemetryMode {
        return this.mode;
    }

    /**
     * Check if telemetry is enabled (All or Errors mode)
     */
    isEnabled(): boolean {
        return this.mode === 'All' || this.mode === 'Errors';
    }
}

// Create a singleton instance
const telemetryService = new TelemetryService({
    accessToken: 'YOUR_ROLLBAR_CLIENT_ACCESS_TOKEN', // This should be configured via environment
    environment: process.env.NODE_ENV || 'development',
    version: '1.0.0', // This should match the app version
});

export default telemetryService;
