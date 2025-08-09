import { useEffect } from "react";
import { useConsoleErrorCallback } from "../hooks/useConsoleErrorCallback";
import { useRollbarTelemetry } from "../hooks/useRollbarTelemetry";

/**
 * Mount this component once to forward console.error calls to Rollbar.
 * It respects telemetry mode via useRollbarTelemetry.
 */
export const ConsoleErrorToRollbar: React.FC = () => {
    const { reportError } = useRollbarTelemetry();

    // Register a console.error callback and forward to Rollbar
    useConsoleErrorCallback((...args: unknown[]) => {
        // Prefer first argument as error/message
        const [first, ...rest] = args;

        // Build a safe payload; avoid sending gigantic objects blindly
        const extras: Record<string, unknown> = {};
        if (rest.length > 0) {
            // Stringify safely with fallbacks
            extras.console_args = rest.map((v) => {
                try {
                    if (v instanceof Error) return { name: v.name, message: v.message, stack: v.stack };
                    if (typeof v === "string" || typeof v === "number" || typeof v === "boolean") return v;
                    return JSON.parse(JSON.stringify(v));
                } catch {
                    return String(v);
                }
            });
        }

        if (first instanceof Error) {
            reportError(first, extras);
        } else if (typeof first === "string") {
            reportError(first, extras);
        } else if (first) {
            // Fallback: stringify unknown first arg
            let message = "console.error called";
            try {
                message = typeof first === "object" ? JSON.stringify(first) : String(first);
            } catch {
                message = String(first);
            }
            reportError(message, extras);
        } else {
            reportError("console.error called with no arguments", extras);
        }
    });

    // Component renders nothing
    useEffect(() => { }, []);
    return null;
};
