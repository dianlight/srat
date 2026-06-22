import {
  Block as BlockIcon,
  PowerSettingsNew as PowerIcon,
} from "@mui/icons-material";
import { Box, IconButton, Tooltip, Typography } from "@mui/material";
import type React from "react";
import { useNavigate } from "react-router";
import { useLabMode } from "../../../hooks/useLabMode";
import {
  type Disk,
  Enabled,
  usePostApiDiskByDiskIdHdidleIgnoreSuggestionMutation,
} from "../../../store/sratApi";

interface HDIdleSuggestionBadgeProps {
  /** The full Disk DTO. The badge derives its visibility from is_rotational
   * and hdidle_device.{enabled, suggestion_ignored}. */
  disk: Disk | undefined;
}

/**
 * HDIdleSuggestionBadge renders an inline action chip on a dashboard disk
 * row, suggesting the user enable HDIdle for an unmonitored HDD.
 *
 * Visibility (all conditions must hold):
 *   1. Lab Mode is on (otherwise the entire HDIdle subsystem is hidden)
 *   2. disk.is_rotational === true (strict — `null`/unknown means SSD-or-USB
 *      enclosure, fail-closed: do not suggest)
 *   3. No active config OR config exists with enabled='no' AND suggestion not
 *      yet ignored
 *
 * Two actions:
 *   - Enable: navigate to /volumes (where the per-disk card lives)
 *   - Ignore: POST /api/disk/{id}/hdidle/ignore-suggestion (persists the
 *     dismissal so the badge does not reappear on next page load)
 */
export const HDIdleSuggestionBadge: React.FC<HDIdleSuggestionBadgeProps> = ({
  disk,
}) => {
  const { labMode } = useLabMode();
  const [ignoreSuggestion, { isLoading: isIgnoring }] =
    usePostApiDiskByDiskIdHdidleIgnoreSuggestionMutation();
  const navigate = useNavigate();

  if (!labMode || !disk) return null;
  if (disk.is_rotational !== true) return null;

  const cfg = disk.hdidle_device;
  // Visible when:
  //   - no per-disk record yet (cfg=undefined, or cfg with no enabled set), OR
  //   - record exists but disk is disabled AND user hasn't ignored the badge
  const alreadyEnabled =
    cfg?.enabled === Enabled.Yes || cfg?.enabled === Enabled.Custom;
  const suggestionIgnored = cfg?.suggestion_ignored === true;
  if (alreadyEnabled || suggestionIgnored) return null;

  const handleEnable = () => {
    // Send the user to the per-disk card; it lives in the volumes page.
    // The disk_id query parameter (if supported) lets the page auto-expand
    // the matching row.
    if (disk.id) {
      navigate(`/volumes?disk=${encodeURIComponent(disk.id)}`);
    } else {
      navigate(`/volumes`);
    }
  };

  const handleIgnore = async () => {
    if (!disk.id) return;
    try {
      await ignoreSuggestion({ diskId: disk.id }).unwrap();
    } catch (err) {
      // Non-fatal: if the dismissal fails the badge reappears on next load.
      console.warn("[HDIdleSuggestionBadge] ignore-suggestion failed:", err);
    }
  };

  return (
    <Box
      sx={{
        display: "inline-flex",
        alignItems: "center",
        gap: 0.5,
        ml: 1,
        px: 0.75,
        py: 0.25,
        borderRadius: 1,
        bgcolor: "action.hover",
      }}
      data-testid="hdidle-suggestion-badge"
    >
      <Typography variant="caption" sx={{ color: "text.secondary" }}>
        Enable HDIdle?
      </Typography>
      <Tooltip title="Enable HDIdle for this disk" arrow>
        <span>
          <IconButton
            size="small"
            onClick={handleEnable}
            aria-label="enable hdidle"
          >
            <PowerIcon fontSize="inherit" />
          </IconButton>
        </span>
      </Tooltip>
      <Tooltip title="Don't show this suggestion again" arrow>
        <span>
          <IconButton
            size="small"
            onClick={handleIgnore}
            disabled={isIgnoring}
            aria-label="ignore hdidle suggestion"
          >
            <BlockIcon fontSize="inherit" />
          </IconButton>
        </span>
      </Tooltip>
    </Box>
  );
};
