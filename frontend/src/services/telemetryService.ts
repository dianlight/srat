import Rollbar from 'rollbar';
import packageJson from '../../package.json';

// Telemetry modes matching the backend enum
export type TelemetryMode = 'Ask' | 'All' | 'Errors' | 'Disabled';

interface TelemetryConfig {
    accessToken: string;
    environment: string;
    version: string;
}

// Export Rollbar configuration for use with @rollbar/react Provider
export const createRollbarConfig = (accessToken: string): Rollbar.Configuration => ({
    accessToken,
    environment: process.env.NODE_ENV || 'development',
    codeVersion: packageJson.version,
    captureUncaught: true,
    captureUnhandledRejections: true,
    payload: {
        client: {
            javascript: {
                code_version: packageJson.version,
                source_map_enabled: true,
            }
        }
    },
    enabled: accessToken !== 'disabled',
});

class TelemetryService {
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
        const newIsConfigured = mode === 'All' || mode === 'Errors';

        // Only log if the configuration status actually changes
        if (this.isConfigured !== newIsConfigured) {
            this.isConfigured = newIsConfigured;

            if (this.isConfigured) {
                console.debug(`Rollbar telemetry configured: ${mode}`);
            } else {
                console.debug(`Rollbar telemetry disabled: ${mode}`);
            }
        } else {
            this.isConfigured = newIsConfigured;
        }
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

    /**
     * Check if the service is configured
     */
    getIsConfigured(): boolean {
        return this.isConfigured;
    }

    /**
     * Get the access token
     */
    getAccessToken(): string {
        return this.config.accessToken;
    }
}
// Create a singleton instance
const telemetryService = new TelemetryService({
    accessToken: process.env.ROLLBAR_CLIENT_ACCESS_TOKEN || 'disabled', // Set via environment at build time
    environment: process.env.NODE_ENV || 'development',
    version: packageJson.version, // Use the version from package.json
});

export default telemetryService;
