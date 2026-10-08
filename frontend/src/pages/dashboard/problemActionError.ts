import type { Problem } from "../../store/sratApi";
import { describeMutationError } from "../../utils/sentrySerialize";

export type ProblemAction = "dismiss" | "ignore" | "re-enable";

export interface ProblemActionFailure {
  /** Sentry/console message: `Failed to <action> problem <key>: <detail>`. */
  message: string;
  /** Toast text: message plus a retry hint when retrying may help. */
  toastMessage: string;
  problemKey: string;
  status?: number | string;
  detail: string;
  retryable: boolean;
  response?: unknown;
}

/** True for 5xx / FETCH_ERROR / timeout statuses where a retry may help. */
export function isRetryableProblemStatus(
  status: number | string | undefined,
): boolean {
  if (typeof status === "number") {
    return status >= 500;
  }
  if (typeof status === "string") {
    const normalized = status.trim().toUpperCase();
    if (
      normalized === "FETCH_ERROR" ||
      normalized === "TIMEOUT_ERROR" ||
      normalized === "TIMEOUT"
    ) {
      return true;
    }
    const asNumber = Number(normalized);
    if (Number.isFinite(asNumber)) {
      return asNumber >= 500;
    }
  }
  return false;
}

/**
 * Parses an RTK Query rejection for problem actions (dismiss / ignore /
 * re-enable) detail-first — `detail`, then `message`, then `status` — via
 * `describeMutationError`, and builds serialized Sentry/toast payloads.
 */
export function describeProblemActionFailure(
  action: ProblemAction,
  problemKey: string,
  err: unknown,
): ProblemActionFailure {
  const described = describeMutationError(err);
  const status = described.status;
  const detail = described.detail;
  const retryable = isRetryableProblemStatus(status);
  const message = `Failed to ${action} problem ${problemKey}: ${detail}`;
  const toastMessage = retryable ? `${message} Retrying may help.` : message;
  return {
    message,
    toastMessage,
    problemKey,
    status,
    detail,
    retryable,
    response: described.data,
  };
}

export type ReenableTarget =
  | { type: "upsert"; issue: Problem }
  | { type: "dismiss" };

/**
 * Resolves a re-enable action to either an upsert of the known problem or a
 * fallback dismiss when the key is gone from the list (e.g. removed via SSE).
 * Extracted for unit testing — the fallback is not reachable deterministically
 * through the UI.
 */
export function resolveReenableTarget(
  problems: readonly (Problem | null | undefined)[],
  id: string,
): ReenableTarget {
  const issue = problems.find((problem) => problem?.problem_key === id);
  if (issue?.problem_key) {
    return { type: "upsert", issue };
  }
  return { type: "dismiss" };
}
