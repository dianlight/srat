import { useMemo, useRef } from "react";
import { type SharedResource, useGetApiSharesQuery } from "../store/sratApi";
import { useGetServerEventsQuery } from "../store/wsApi";

const MAX_SHARE_NAME_LENGTH = 128;

function toRecords(shares: SharedResource[]): Record<string, SharedResource> {
  const result: Record<string, SharedResource> = {};
  for (const share of shares) {
    const key = share.name;
    if (key) {
      if (encodeURIComponent(key).length > MAX_SHARE_NAME_LENGTH) {
        console.warn(
          `Share with name "${key}" exceeds maximum length (${MAX_SHARE_NAME_LENGTH}) after encoding and will be skipped.`,
        );
        continue;
      }
      result[key] = share;
    }
  }
  return result;
}

// GetApiSharesApiResponse is a union (SharedResource[] | ErrorModel); narrow
// it before treating it as the shares array so an error response cannot leak
// into the derived value.
const isShareArray = (value: unknown): value is SharedResource[] =>
  Array.isArray(value);

export function useShare() {
  const { data, error, isLoading } = useGetApiSharesQuery();
  const { data: evdata, error: everror } = useGetServerEventsQuery();

  // Single derived value: WS (live) wins once a payload arrives, REST is the
  // fallback until then. No local state and no effects, so there is no race
  // between the two sources and the first REST response renders immediately
  // instead of waiting an extra effect cycle (or a late WS event).
  //
  // Reference stability: the WebSocket stream re-parses JSON on every message,
  // producing a new SharedResource[] reference even when the content is
  // unchanged. Downstream effects in Shares.tsx depend on this value's
  // identity, so we compare serialised snapshots and return the previous
  // reference when the content is identical, preventing unnecessary
  // re-render cascades.
  const prevResultRef = useRef<{
    json: string;
    shares: SharedResource[];
  } | null>(null);

  const shares = useMemo<SharedResource[]>(() => {
    const next = evdata?.shares ?? (isShareArray(data) ? data : []);
    const nextJson = JSON.stringify(next);
    if (prevResultRef.current?.json === nextJson) {
      return prevResultRef.current.shares;
    }
    const result = { json: nextJson, shares: next };
    prevResultRef.current = result;
    return next;
  }, [evdata?.shares, data]);

  const shareRecords = useMemo(() => toRecords(shares), [shares]);

  return {
    shares: shareRecords,
    // REST gates loading only until the first WS shares payload arrives; after
    // that the live stream is authoritative (a REST refetch triggered by a
    // dirty-tracking invalidation must not render an empty list while it is
    // still loading).
    isLoading: isLoading && !evdata?.shares,
    error: error ?? everror,
  };
}
