import { useMemo, useRef } from "react";
import { type Disk, useGetApiVolumesQuery } from "../store/sratApi";
import { useGetServerEventsQuery } from "../store/wsApi";

// GetApiVolumesApiResponse is a union (Disk[] | null | ErrorModel); narrow it
// before treating it as the disks array so an error response cannot leak into
// the derived value as `undefined`.
const isDiskArray = (value: unknown): value is Disk[] => Array.isArray(value);

export function useVolume() {
  const { data: evdata, error: everror } = useGetServerEventsQuery();
  const { data, error, isLoading } = useGetApiVolumesQuery();

  // Single derived value: SSE (live) wins once a payload arrives, REST is the
  // fallback until then. No local state and no effects, so there is no race
  // between the two sources and optimistic edits are never clobbered.
  //
  // Reference stability: the WebSocket stream re-parses JSON on every message,
  // producing a new Disk[] reference even when the content is unchanged.
  // Downstream effects in Volumes.tsx depend on this array's identity, so we
  // compare serialised snapshots and return the previous reference when the
  // content is identical, preventing unnecessary re-render cascades.
  const prevResultRef = useRef<{ json: string; disks: Disk[] } | null>(null);

  const disks = useMemo<Disk[]>(() => {
    const next = evdata?.volumes ?? (isDiskArray(data) ? data : []);
    const nextJson = JSON.stringify(next);
    if (prevResultRef.current?.json === nextJson) {
      return prevResultRef.current.disks;
    }
    const result = { json: nextJson, disks: next };
    prevResultRef.current = result;
    return next;
  }, [evdata?.volumes, data]);

  return {
    disks,
    // REST gates loading only until the first SSE payload arrives; after that
    // the live stream is authoritative (a failing SSE query must not render an
    // empty list while REST is still loading).
    isLoading: isLoading && !evdata?.volumes,
    error: error ?? everror,
  };
}
