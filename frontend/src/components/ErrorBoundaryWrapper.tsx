import React from 'react';
import { ErrorBoundary } from '@rollbar/react';
import { useRollbarTelemetry } from '../hooks/useRollbarTelemetry';
import { getCurrentEnv } from '../macro/Environment' with { type: 'macro' };

interface ErrorBoundaryWrapperProps {
    children: React.ReactNode;
}

// Custom fallback UI component
const ErrorFallback: React.FC<{ error: Error | null; resetError: () => void }> = ({ error, resetError }) => (
    <div style={{ padding: '20px', textAlign: 'center' }}>
        <h2>Something went wrong</h2>
        <p>An unexpected error occurred. The error has been reported.</p>
        <button onClick={resetError}>Try again</button>
        {error && getCurrentEnv() === 'development' && (
            <details style={{ marginTop: '10px', textAlign: 'left' }}>
                <summary>Error details (development only)</summary>
                <pre style={{ fontSize: '12px', color: '#666' }}>
                    {error.toString()}
                </pre>
            </details>
        )}
    </div>
);

export const ErrorBoundaryWrapper: React.FC<ErrorBoundaryWrapperProps> = ({ children }) => {
    return (
        <ErrorBoundary
            fallbackUI={ErrorFallback}
            extra={(error, errorInfo) => ({
                timestamp: new Date().toISOString(),
                userAgent: navigator.userAgent,
                url: window.location.href,
                ...errorInfo,
            })}
        >
            {children}
        </ErrorBoundary>
    );
};

// Hook to manually report errors from components (updated to use new telemetry system)
export const useErrorReporting = () => {
    const { reportError, reportEvent } = useRollbarTelemetry();

    return {
        reportError,
        reportEvent,
    };
};
