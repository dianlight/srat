import { useCallback, useMemo } from "react";
import { type LabFeature, useGetApiLabFeaturesQuery } from "../store/sratApi";

/**
 * useLabFeatures reads /api/lab_features — the server-side lab-feature
 * registry (single source of truth, maintained in
 * backend/src/service/lab_features.go).
 *
 * Alpha features are omitted entirely from the response in release
 * (production) builds, so the frontend never needs env logic: what the
 * server does not send, the UI cannot render.
 *
 * `isAvailable(key)` gates individual lab surfaces:
 * - alpha → always true here (server only sends them in dev/prerelease)
 * - beta  → true only when experimental_lab_mode is enabled
 *
 * Returns an empty list while loading / on error — fail-closed semantics,
 * never render lab-only UI optimistically.
 */
export function useLabFeatures(): {
  features: LabFeature[];
  isAvailable: (key: string) => boolean;
  isLoading: boolean;
} {
  const { data, isLoading } = useGetApiLabFeaturesQuery();
  // GetApiLabFeaturesApiResponse is LabFeature[] | ErrorModel. RTK Query sets
  // `data` to undefined on error (never to ErrorModel), but the union type
  // requires narrowing before accessing array fields.
  const features = useMemo(
    () => (data as LabFeature[] | undefined) ?? [],
    [data],
  );
  const featureMap = useMemo(
    () => new Map(features.map((f) => [f.key, f])),
    [features],
  );
  // Stable identity so consumers can safely use isAvailable in useMemo deps.
  const isAvailable = useCallback(
    (key: string) => Boolean(featureMap.get(key)?.available),
    [featureMap],
  );
  return { features, isAvailable, isLoading };
}
