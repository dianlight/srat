import React from 'react';
import { useErrorBoundary } from 'react-use-error-boundary';
import telemetryService from '../services/telemetryService';

interface ErrorBoundaryWrapperProps {
    children: React.ReactNode;
}

export const ErrorBoundaryWrapper: React.FC<ErrorBoundaryWrapperProps> = ({ children }) => {
    const [error, resetError] = useErrorBoundary((error, errorInfo) => {
        // Log error to console for development
        console.error('ErrorBoundary caught an error:', error, errorInfo);

        // Report error to telemetry service
        telemetryService.reportError(error instanceof Error ? error : new Error(String(error)), {
            errorInfo,
            timestamp: new Date().toISOString(),
            userAgent: navigator.userAgent,
            url: window.location.href,
        });
    });

    // If there's an error, you could render a fallback UI here
    if (error) {
        return (
            <div style={{ padding: '20px', textAlign: 'center' }}>
                <h2>Something went wrong</h2>
                <p>An unexpected error occurred. The error has been reported.</p>
                <button onClick={resetError}>Try again</button>
            </div>
        );
    }

    return <>{children}</>;
};

// Hook to manually report errors from components
export const useErrorReporting = () => {
    return {
        reportError: (error: Error | string, extraData?: Record<string, any>) => {
            telemetryService.reportError(error, extraData);
        },
        reportEvent: (event: string, data?: Record<string, any>) => {
            telemetryService.reportEvent(event, data);
        },
    };
};
