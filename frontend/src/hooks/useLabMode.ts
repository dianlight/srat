import { useLabFeatures } from "./useLabFeatures";

/**
 * useLabMode is a thin compatibility wrapper over the lab-feature registry.
 *
 * HDIdle is registered as a BETA lab feature ("hdidle"), so its availability
 * is exactly `experimental_lab_mode` — the registry is now the single source
 * of truth (backend/src/service/lab_features.go). Keeping this hook avoids
 * churning the callers (dashboard suggestion badge, disk health metrics) and
 * their test mocks.
 *
 * Returns `false` while the query is loading, on error, or when the feature
 * is absent — fail-closed semantics: never render Lab-Mode-only UI
 * optimistically.
 */
export function useLabMode(): { labMode: boolean; isLoading: boolean } {
  const { isAvailable, isLoading } = useLabFeatures();
  return { labMode: isAvailable("hdidle"), isLoading };
}
