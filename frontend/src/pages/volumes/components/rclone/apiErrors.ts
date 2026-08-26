/**
 * Extracts a human-readable message from an RTK Query error whose payload is
 * the generated ErrorModel, falling back to the raw Error message.
 *
 * Huma wraps handler errors as `{ detail: <generic>, errors: [{message:
 * <cause>}] }`; the specific cause in `errors` is what users need, so it is
 * preferred over the generic wrapper text.
 */
export function extractApiErrorMessage(
  err: unknown,
  fallback = "Request failed",
): string {
  const payload = (err as { data?: unknown } | null)?.data;
  if (payload && typeof payload === "object") {
    const model = payload as Record<string, unknown>;
    if (Array.isArray(model.errors)) {
      for (const e of model.errors) {
        const msg =
          typeof e === "string"
            ? e
            : ((e as { message?: unknown })?.message ?? "");
        if (typeof msg === "string" && msg.length > 0) {
          return msg;
        }
      }
    }
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
