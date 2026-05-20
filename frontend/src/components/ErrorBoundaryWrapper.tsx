import * as Sentry from "@sentry/react";
import type React from "react";
import { useSentryTelemetry } from "../hooks/useSentryTelemetry";
import { getCurrentEnv } from "../macro/Environment" with { type: "macro" };

interface ErrorBoundaryWrapperProps {
  children: React.ReactNode;
}

// Custom fallback UI component
const ErrorFallback: React.FC<{
  error: Error | null;
  resetError: () => void;
}> = ({ error, resetError }) => (
  <div style={{ padding: "20px", textAlign: "center" }}>
    <h2>Something went wrong</h2>
    <p>An unexpected error occurred. The error has been reported.</p>
    <button type="button" onClick={resetError}>
      Try again
    </button>
    {error && getCurrentEnv() === "development" && (
      <details style={{ marginTop: "10px", textAlign: "left" }}>
        <summary>Error details (development only)</summary>
        <pre style={{ fontSize: "12px", color: "#666" }}>
          {error.toString()}
        </pre>
      </details>
    )}
  </div>
);

export const ErrorBoundaryWrapper: React.FC<ErrorBoundaryWrapperProps> = ({
  children,
}) => {
  return (
    <Sentry.ErrorBoundary
      fallback={({ error, resetError }) => (
        <ErrorFallback
          error={
            error instanceof Error
              ? error
              : error
                ? new Error(String(error))
                : null
          }
          resetError={resetError}
        />
      )}
      beforeCapture={(scope, error, componentStack) => {
        scope.setContext("boundary", {
          timestamp: new Date().toISOString(),
          userAgent: navigator.userAgent,
          url: window.location.href,
          error: String(error),
          componentStack,
        });
      }}
    >
      {children}
    </Sentry.ErrorBoundary>
  );
};

// Hook to manually report errors from components (updated to use new telemetry system)
export const useErrorReporting = () => {
  const { reportError, reportEvent } = useSentryTelemetry();

  return {
    reportError,
    reportEvent,
  };
};
