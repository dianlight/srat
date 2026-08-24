/**
 * Extracts a human-readable message from an RTK Query error whose payload is
 * the generated ErrorModel, falling back to the raw Error message.
 */
export function extractApiErrorMessage(
  err: unknown,
  fallback = "Request failed",
): string {
  const payload = (err as { data?: unknown } | null)?.data;
  if (payload && typeof payload === "object") {
    const model = payload as Record<string, unknown>;
    if (typeof model.detail === "string" && model.detail.length > 0) {
      return model.detail;
    }
    if (typeof model.message === "string" && model.message.length > 0) {
      return model.message;
    }
  }
  if (err instanceof Error && err.message) {
    return err.message;
  }
  return fallback;
}
