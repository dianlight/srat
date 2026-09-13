import semver from "semver";
import { type NewsItem, useGetDiscussionsQuery } from "../store/githubRestApi";
import { useGetServerEventsQuery } from "../store/wsApi";

export type { NewsType } from "../store/githubRestApi";
export type { NewsItem };

/** Normalize a raw version string for semver comparison. */
export function normalizeVersion(raw?: string): string | null {
  if (!raw) {
    return null;
  }
  const cleaned = raw.trim().replace(/^[vV]/, "");
  if (!cleaned) {
    return null;
  }
  if (semver.valid(cleaned)) {
    return cleaned;
  }
  return semver.coerce(cleaned)?.version ?? null;
}

/**
 * A release is news only when its version is strictly newer than the
 * running build. Unknown running version fails open (show); unknown
 * item version fails closed (hide — cannot prove it is newer).
 */
export function isNewerRelease(
  itemVersion: string | undefined,
  runningVersion: string | undefined,
): boolean {
  const item = normalizeVersion(itemVersion);
  if (!item) {
    return false;
  }
  const running = normalizeVersion(runningVersion);
  if (!running) {
    return true;
  }
  try {
    return semver.gt(item, running);
  } catch {
    return false;
  }
}

// Hook wrapper around the RTK Query endpoint. Keeping the same return
// shape as the legacy hook for compatibility with existing consumers.
export function useGithubNews() {
  const { data, isLoading, error } = useGetDiscussionsQuery();
  const { data: serverEvents } = useGetServerEventsQuery();
  const runningVersion = serverEvents?.hello?.build_version;

  let mappedError: Error | null = null;
  if (error) {
    try {
      // error may be a serialized object from fetchBaseQuery
      const anyErr = error as unknown as { status?: number; error?: unknown };
      if (anyErr?.status) {
        mappedError = new Error(
          `GitHub API request failed: ${String(anyErr.status)}`,
        );
      } else if (anyErr?.error) {
        mappedError = new Error(String(anyErr.error));
      } else {
        mappedError = new Error(JSON.stringify(anyErr));
      }
    } catch {
      mappedError = new Error("Failed to fetch GitHub news");
    }
  }

  const news = (data ?? []).filter((item) =>
    item.type === "release"
      ? isNewerRelease(item.version, runningVersion)
      : true,
  );

  return { news, isLoading, error: mappedError };
}
