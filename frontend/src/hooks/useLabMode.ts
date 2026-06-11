import { useGetApiSettingsQuery } from "../store/sratApi";

/**
 * useLabMode reads `experimental_lab_mode` from /api/settings.
 *
 * Lab Mode is the gating flag for all HDIdle UI surfaces (dashboard
 * suggestion badge, per-disk settings card, ignore-suggestion endpoint).
 * When false, the backend itself returns 403 on every /api/disk/{id}/hdidle/*
 * route — the frontend must therefore hide the surfaces rather than render
 * them and fail.
 *
 * Returns `false` while the settings query is loading, on error, or when
 * settings are missing — fail-closed semantics: never render Lab-Mode-only
 * UI optimistically.
 */
export function useLabMode(): { labMode: boolean; isLoading: boolean } {
  const { data, isLoading } = useGetApiSettingsQuery();
  return {
    labMode:
      data != null && "experimental_lab_mode" in data
        ? Boolean(data.experimental_lab_mode)
        : false,
    isLoading,
  };
}
