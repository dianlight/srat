import type {
  RcloneAuthStartResponse,
  RcloneDiffResult,
  RcloneLink,
  RcloneProvidersResponse,
} from "../../../../store/sratApi";

/**
 * Runtime guards for generated union responses
 * (`SuccessType | ErrorModel`). RTK Query keeps the union in its typings,
 * so consumers must narrow before touching success-only fields.
 */

export function isRcloneLink(v: unknown): v is RcloneLink {
  return typeof v === "object" && v !== null && "target_kind" in v;
}

export function isProvidersResponse(v: unknown): v is RcloneProvidersResponse {
  return typeof v === "object" && v !== null && "library_available" in v;
}

export function isAuthStartResponse(v: unknown): v is RcloneAuthStartResponse {
  return typeof v === "object" && v !== null && "auth_url" in v;
}

export function isDiffResult(v: unknown): v is RcloneDiffResult {
  return typeof v === "object" && v !== null && "entries" in v;
}
