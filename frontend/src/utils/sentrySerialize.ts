import type { ErrorModel } from "../store/sratApi";

const MAX_DEPTH = 5;
const MAX_STRING_LENGTH = 2000;
const MAX_ARRAY_LENGTH = 50;

export interface DescribedMutationError {
  status?: number | string;
  detail: string;
  data?: unknown;
}

const truncate = (value: string): string =>
  value.length > MAX_STRING_LENGTH
    ? `${value.slice(0, MAX_STRING_LENGTH)}…[truncated ${value.length - MAX_STRING_LENGTH} chars]`
    : value;

const isPlainRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

const extractBackendDetail = (data: unknown): string | undefined => {
  if (typeof data === "string") {
    const trimmed = data.trim();
    return trimmed.length > 0 ? trimmed : undefined;
  }
  if (!isPlainRecord(data)) {
    return undefined;
  }
  const model = data as Partial<ErrorModel> & Record<string, unknown>;
  const parts: string[] = [];
  if (typeof model.detail === "string" && model.detail.trim().length > 0) {
    parts.push(model.detail.trim());
  }
  if (typeof model.title === "string" && model.title.trim().length > 0) {
    // Keep title only when it adds signal beyond detail.
    if (!parts.some((part) => part.includes(model.title as string))) {
      parts.push(model.title.trim());
    }
  }
  if (typeof model.message === "string" && model.message.trim().length > 0) {
    parts.push(model.message.trim());
  }
  if (Array.isArray(model.errors)) {
    for (const entry of model.errors.slice(0, 3)) {
      if (
        isPlainRecord(entry) &&
        typeof entry.message === "string" &&
        entry.message.trim().length > 0
      ) {
        const location =
          typeof entry.location === "string" && entry.location.trim().length > 0
            ? ` (${entry.location.trim()})`
            : "";
        parts.push(`${entry.message.trim()}${location}`);
      }
    }
  }
  return parts.length > 0 ? parts.join(" | ") : undefined;
};

export const serializeSentryValue = (
  value: unknown,
  depth = 0,
  seen = new WeakSet<object>(),
): unknown => {
  if (
    value === null ||
    typeof value === "string" ||
    typeof value === "number" ||
    typeof value === "boolean"
  ) {
    return typeof value === "string" ? truncate(value) : value;
  }
  if (typeof value === "undefined") {
    return "undefined";
  }
  if (typeof value === "bigint") {
    return `${value.toString()}n`;
  }
  if (typeof value === "symbol" || typeof value === "function") {
    return String(value);
  }
  if (value instanceof Error) {
    const serialized: Record<string, unknown> = {
      name: value.name,
      message: value.message,
    };
    if (typeof value.stack === "string") {
      serialized.stack = truncate(value.stack);
    }
    const cause = (value as Error & { cause?: unknown }).cause;
    if (cause !== undefined) {
      serialized.cause =
        depth < MAX_DEPTH
          ? serializeSentryValue(cause, depth + 1, seen)
          : "[Max depth]";
    }
    // Include enumerable custom props (e.g. RTK SerializedError extras).
    for (const key of Object.keys(value)) {
      if (key === "name" || key === "message" || key === "stack") {
        continue;
      }
      serialized[key] =
        depth < MAX_DEPTH
          ? serializeSentryValue(
              (value as unknown as Record<string, unknown>)[key],
              depth + 1,
              seen,
            )
          : "[Max depth]";
    }
    return serialized;
  }
  if (depth >= MAX_DEPTH) {
    return "[Max depth]";
  }
  if (typeof value === "object") {
    if (seen.has(value)) {
      return "[Circular]";
    }
    seen.add(value);
    try {
      if (Array.isArray(value)) {
        return value
          .slice(0, MAX_ARRAY_LENGTH)
          .map((entry) => serializeSentryValue(entry, depth + 1, seen));
      }
      const out: Record<string, unknown> = {};
      for (const [key, entry] of Object.entries(
        value as Record<string, unknown>,
      )) {
        out[key] = serializeSentryValue(entry, depth + 1, seen);
      }
      return out;
    } finally {
      seen.delete(value);
    }
  }
  try {
    return truncate(String(value));
  } catch {
    return "[Unserializable]";
  }
};

export const normalizeSentryExtras = (
  extras?: Record<string, unknown>,
): Record<string, unknown> => {
  if (!extras) {
    return {};
  }
  const normalized: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(extras)) {
    try {
      normalized[key] = serializeSentryValue(value);
    } catch {
      try {
        normalized[key] = String(value);
      } catch {
        normalized[key] = "[Unserializable]";
      }
    }
  }
  return normalized;
};

export const describeMutationError = (
  error: unknown,
): DescribedMutationError => {
  if (error instanceof Error) {
    const cause = (error as Error & { cause?: unknown }).cause;
    const fromCause =
      cause !== undefined && !(cause instanceof Error)
        ? describeMutationError(cause)
        : undefined;
    return {
      detail: error.message.trim().length > 0 ? error.message : String(error),
      data: serializeSentryValue(fromCause?.data ?? cause),
    };
  }
  if (isPlainRecord(error)) {
    const record = error as Record<string, unknown> & {
      status?: unknown;
      data?: unknown;
      message?: unknown;
      error?: unknown;
    };
    if ("status" in record || "data" in record) {
      const status =
        typeof record.status === "number" || typeof record.status === "string"
          ? record.status
          : undefined;
      const backendDetail = extractBackendDetail(record.data);
      if (backendDetail) {
        return {
          status,
          detail:
            status !== undefined
              ? `HTTP ${String(status)}: ${backendDetail}`
              : backendDetail,
          data: serializeSentryValue(record.data),
        };
      }
      if (typeof record.error === "string" && record.error.trim().length > 0) {
        return {
          status,
          detail:
            status !== undefined
              ? `HTTP ${String(status)}: ${record.error.trim()}`
              : record.error.trim(),
          data: serializeSentryValue(record),
        };
      }
      if (
        typeof record.message === "string" &&
        record.message.trim().length > 0
      ) {
        return {
          status,
          detail: record.message.trim(),
          data: serializeSentryValue(record),
        };
      }
      return {
        status,
        detail:
          status !== undefined
            ? `Request failed with status ${String(status)}`
            : "Request failed",
        data: serializeSentryValue(record),
      };
    }
    if (
      typeof record.message === "string" &&
      record.message.trim().length > 0
    ) {
      return {
        detail: record.message.trim(),
        data: serializeSentryValue(record),
      };
    }
  }
  if (typeof error === "string") {
    return {
      detail: error.trim().length > 0 ? error : "Unknown error",
    };
  }
  try {
    const asString = String(error);
    return {
      detail: asString === "[object Object]" ? "Unknown error" : asString,
      data: serializeSentryValue(error),
    };
  } catch {
    return { detail: "Unknown error" };
  }
};

export interface WizardCommitFailure {
  op: string;
  status?: number | string;
  detail: string;
  context?: Record<string, unknown>;
  response?: unknown;
}

export const buildWizardFailureMessage = (
  failures: Pick<WizardCommitFailure, "op" | "detail">[],
): string => {
  const failedOps = failures.map((failure) => failure.op).join(", ");
  const firstDetail = failures[0]?.detail ?? "Unknown error";
  return `Error applying settings in wizard [${failedOps}]: ${firstDetail}`;
};

export const buildWizardFailureToast = (
  failures: Pick<WizardCommitFailure, "op" | "detail">[],
): string => {
  const failedOps = failures.map((failure) => failure.op).join(", ");
  const firstDetail = failures[0]?.detail ?? "Unknown error";
  return `Failed to save ${failedOps}. ${firstDetail} You can configure them later in Settings.`;
};

export const buildWizardFailureExtras = (options: {
  shareName?: string;
  partitionId?: string;
  failures: WizardCommitFailure[];
}): Record<string, unknown> =>
  normalizeSentryExtras({
    wizard: {
      step: "summary",
      shareName: options.shareName || undefined,
      partitionId: options.partitionId || undefined,
    },
    failures: options.failures.map((failure) => ({
      op: failure.op,
      status: failure.status,
      detail: failure.detail,
      context: failure.context,
      response: failure.response,
    })),
  });
