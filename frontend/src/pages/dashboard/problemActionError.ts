import { useEffect, useSyncExternalStore } from "react";
import { toast } from "react-toastify";

import {
  type ErrorModel,
  type Problem,
  Severity,
  Status,
} from "../../store/sratApi";
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
  { type: "upsert"; issue: Problem } | { type: "dismiss" };

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

/**
 * Stable toast ID for a problem notification. App.tsx creates the toast with
 * this ID and the problem actions (ignore / re-enable) use it to dismiss or
 * re-open the matching toast without waiting for the next WebSocket event.
 */
export function problemToastId(problemKey: string): string {
  return `problem-${problemKey}`;
}

/** Dismisses a toast by its ID (used for immediate optimistic dismissal). */
export function dismissProblemToast(toastId: string): void {
  toast.dismiss(toastId);
}

/** Creates or updates the single toast for a problem. */
export function showProblemToast(problem: Problem): void {
  const toastId = problemToastId(problem.problem_key);
  const content = problem.title || problem.description || problem.problem_key;
  const isError =
    problem.severity === Severity.Error ||
    problem.severity === Severity.Critical;
  const isWarning = problem.severity === Severity.Warning;
  const toastFn = isError ? toast.error : isWarning ? toast.warn : toast.info;
  const isActive = (toast as unknown as { isActive?: (id: string) => boolean })
    .isActive;

  if (typeof isActive === "function" && isActive(toastId)) {
    toast.update(toastId, {
      render: content,
      data: { error: problem, exclude: false },
      type: isError ? "error" : isWarning ? "warning" : "info",
    } as unknown as Parameters<typeof toast.update>[1]);
    return;
  }

  (toastFn as (message: string, options?: unknown) => void)(content, {
    toastId,
    data: { error: problem, exclude: false },
    autoClose: false,
  } as unknown as Record<string, unknown>);
}

/** A locally requested ignore/re-enable and the server state it started from. */
export interface ProblemIgnoreOverride {
  ignored: boolean;
  baselineUpdatedAt?: string;
}

interface StoredProblemIgnoreOverride extends ProblemIgnoreOverride {
  previous?: ProblemIgnoreOverride | null;
}

// A stale WS `problem` event keeps its original payload after an ignore or
// re-enable. Track the locally requested state so cards, toasts and the badge
// react immediately, then clear the override once a server-sourced update
// (mutation response, REST refetch or live WS event) confirms the state.
const localIgnoreOverrides = new Map<string, StoredProblemIgnoreOverride>();
let ignoreOverridesSnapshot: ReadonlyMap<string, ProblemIgnoreOverride> =
  new Map();
const overrideListeners = new Set<() => void>();

function publicOverride(
  override: StoredProblemIgnoreOverride,
): ProblemIgnoreOverride {
  return {
    ignored: override.ignored,
    baselineUpdatedAt: override.baselineUpdatedAt,
  };
}

function publishIgnoreOverrides(): void {
  // useSyncExternalStore needs a new snapshot identity on every change.
  ignoreOverridesSnapshot = new Map(
    [...localIgnoreOverrides].map(([key, value]) => [
      key,
      publicOverride(value),
    ]),
  );
  for (const listener of overrideListeners) {
    listener();
  }
}

export function setLocalProblemIgnore(
  key: string,
  ignored: boolean,
  baselineUpdatedAt?: string,
): void {
  const existing = localIgnoreOverrides.get(key);
  const previous =
    existing?.previous ?? (existing ? publicOverride(existing) : null) ?? null;
  localIgnoreOverrides.set(key, {
    ignored,
    baselineUpdatedAt,
    previous,
  });
  publishIgnoreOverrides();
}

export function restoreProblemIgnoreOverride(
  key: string,
  previous?: ProblemIgnoreOverride | null,
): void {
  if (!previous) {
    clearLocalProblemIgnore(key);
    return;
  }
  setLocalProblemIgnore(key, previous.ignored, previous.baselineUpdatedAt);
}

export function clearLocalProblemIgnore(key: string): void {
  if (localIgnoreOverrides.delete(key)) {
    publishIgnoreOverrides();
  }
}

export function getLocalProblemIgnore(key: string): boolean | undefined {
  return localIgnoreOverrides.get(key)?.ignored;
}

export function getProblemIgnoreOverride(
  key: string,
): ProblemIgnoreOverride | undefined {
  const override = localIgnoreOverrides.get(key);
  return override ? publicOverride(override) : undefined;
}

export function subscribeProblemIgnoreOverrides(
  listener: () => void,
): () => void {
  overrideListeners.add(listener);
  return () => {
    overrideListeners.delete(listener);
  };
}

function getIgnoreOverridesSnapshot(): ReadonlyMap<
  string,
  ProblemIgnoreOverride
> {
  return ignoreOverridesSnapshot;
}

export function useProblemIgnoreOverrides(): ReadonlyMap<
  string,
  ProblemIgnoreOverride
> {
  return useSyncExternalStore(
    subscribeProblemIgnoreOverrides,
    getIgnoreOverridesSnapshot,
    getIgnoreOverridesSnapshot,
  );
}

/** True when the problem must not surface as an alert (card/list/toast). */
export function isInactiveProblem(
  problem: Pick<Problem, "ignored" | "status"> | null | undefined,
): boolean {
  if (!problem) {
    return false;
  }
  return (
    problem.ignored === true ||
    problem.status === Status.Ignored ||
    problem.status === Status.Dismissed ||
    problem.status === Status.Fixed ||
    problem.status === Status.Deleted
  );
}

/** True when the server problem itself represents an ignored alert. */
export function isIgnoredProblemState(
  problem: Pick<Problem, "ignored" | "status"> | null | undefined,
): boolean {
  return problem?.ignored === true || problem?.status === Status.Ignored;
}

/** True when the server problem is eligible to surface as an active alert. */
export function isActiveProblemState(
  problem: Pick<Problem, "ignored" | "status"> | null | undefined,
): boolean {
  return Boolean(problem) && !isInactiveProblem(problem);
}

export type ProblemListSource =
  readonly (Problem | null | undefined)[] | ErrorModel | null | undefined;

function toProblemArray(problems: ProblemListSource): Problem[] {
  return Array.isArray(problems)
    ? problems.filter((problem): problem is Problem => Boolean(problem))
    : [];
}

function parseUpdatedAt(updatedAt: string | undefined): number | undefined {
  if (!updatedAt) {
    return undefined;
  }
  const parsed = Date.parse(updatedAt);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function isLiveProblemOlder(
  candidate: Pick<Problem, "updated_at">,
  current: Pick<Problem, "updated_at">,
): boolean {
  const candidateTime = parseUpdatedAt(candidate.updated_at);
  const currentTime = parseUpdatedAt(current.updated_at);
  return (
    candidateTime !== undefined &&
    currentTime !== undefined &&
    candidateTime < currentTime
  );
}

function isLiveProblemNewer(
  candidate: Pick<Problem, "updated_at">,
  baselineUpdatedAt: string | undefined,
): boolean {
  const candidateTime = parseUpdatedAt(candidate.updated_at);
  const baselineTime = parseUpdatedAt(baselineUpdatedAt);
  return (
    candidateTime !== undefined &&
    baselineTime !== undefined &&
    candidateTime > baselineTime
  );
}

/**
 * Overlays the latest WS problem on the REST list. Equal or missing timestamps
 * keep the WS precedence used before the ignore rework; an explicitly older
 * WS payload is treated as stale and cannot resurrect a reconciled card.
 */
export function mergeLiveProblems(
  problems: ProblemListSource,
  liveProblem: Problem | null | undefined,
): Problem[] {
  const merged = toProblemArray(problems);
  if (!liveProblem?.problem_key) {
    return merged;
  }
  const index = merged.findIndex(
    (problem) => problem.problem_key === liveProblem.problem_key,
  );
  if (index < 0) {
    return [liveProblem, ...merged];
  }
  const existing = merged[index];
  if (existing && isLiveProblemOlder(liveProblem, existing)) {
    return merged;
  }
  return merged.map((problem, currentIndex) =>
    currentIndex === index ? liveProblem : problem,
  );
}

/** Returns the authoritative problem for a key, including a live WS overlay. */
export function getAuthoritativeProblem(
  problems: ProblemListSource,
  liveProblem: Problem | null | undefined,
  problemKey?: string,
): Problem | undefined {
  const targetKey = problemKey ?? liveProblem?.problem_key;
  if (!targetKey) {
    return undefined;
  }
  return mergeLiveProblems(problems, liveProblem).find(
    (problem) => problem.problem_key === targetKey,
  );
}

/**
 * An override stays active only while it disagrees with the latest server
 * state. A matching state confirms the click, and a newer `updated_at` means
 * an external server change must win over the local request.
 */
export function isProblemIgnoreOverrideActive(
  override: ProblemIgnoreOverride | undefined,
  serverProblem: Pick<Problem, "ignored" | "status" | "updated_at"> | undefined,
): boolean {
  if (!override) {
    return false;
  }
  if (!serverProblem) {
    return true;
  }
  const requestedStateMatches = override.ignored
    ? isIgnoredProblemState(serverProblem)
    : isActiveProblemState(serverProblem);
  if (requestedStateMatches) {
    return false;
  }
  return !isLiveProblemNewer(serverProblem, override.baselineUpdatedAt);
}

/** Clears overrides confirmed by REST/WS state. */
export function reconcileProblemIgnoreFromServer(
  problems: ProblemListSource,
  liveProblem: Problem | null | undefined,
): void {
  if (localIgnoreOverrides.size === 0) {
    return;
  }
  const byKey = new Map(
    mergeLiveProblems(problems, liveProblem)
      .filter((problem) => Boolean(problem.problem_key))
      .map((problem) => [problem.problem_key, problem] as const),
  );
  for (const [key, override] of localIgnoreOverrides) {
    if (!isProblemIgnoreOverrideActive(override, byKey.get(key))) {
      clearLocalProblemIgnore(key);
    }
  }
}

/** Type guard for a successful problem mutation response. */
export function isProblemResponse(value: unknown): value is Problem {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const candidate = value as Partial<Problem>;
  return (
    typeof candidate.problem_key === "string" &&
    candidate.problem_key.length > 0 &&
    typeof candidate.ignored === "boolean" &&
    typeof candidate.status === "string"
  );
}

/** Clears an override when the mutation response confirms the requested state. */
export function reconcileProblemIgnoreMutationResult(
  response: unknown,
): boolean {
  if (!isProblemResponse(response)) {
    return false;
  }
  const before = getLocalProblemIgnore(response.problem_key);
  reconcileProblemIgnoreFromServer([response], undefined);
  return (
    before !== undefined &&
    getLocalProblemIgnore(response.problem_key) !== before
  );
}

/** Reconciles from server snapshots inside a component lifecycle. */
export function useReconcileProblemIgnoreOverrides(
  problems: ProblemListSource,
  liveProblem: Problem | null | undefined,
): void {
  useEffect(() => {
    reconcileProblemIgnoreFromServer(problems, liveProblem);
  }, [problems, liveProblem]);
}
