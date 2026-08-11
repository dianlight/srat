import { Box, type BoxProps, useMediaQuery, useTheme } from "@mui/material";
import type { MouseEvent as ReactMouseEvent, ReactNode } from "react";
import { useCallback, useEffect, useRef, useState } from "react";

export const DEFAULT_LEFT_PANEL_PCT = 30;
export const MIN_LEFT_PANEL_PCT = 15;
export const MAX_LEFT_PANEL_PCT = 60;
const RESIZE_KEYBOARD_STEP_PCT = 5;

/**
 * Custom attributes needed by the guided tour (reactour) selectors.
 */
export type TourDataTutorProps = {
  "data-tutor"?: string;
};

/**
 * Extra attributes spread onto a layout wrapper (e.g. tour anchors).
 * Layout styling is controlled by the component itself.
 */
export type ResizableSplitViewPanelProps = Omit<BoxProps, "sx"> &
  TourDataTutorProps;

export interface ResizableSplitViewProps {
  /**
   * localStorage key used to persist the left panel width percentage.
   * Keep it unique per page so each panel has an independent width.
   */
  storageKey: string;
  /** Content rendered in the left (tree/navigation) panel. */
  leftPanel: ReactNode;
  /** Content rendered in the right (details) panel. */
  rightPanel: ReactNode;
  /** Initial left panel width in percent when nothing is persisted. */
  defaultLeftPanelPct?: number;
  /** Minimum left panel width in percent (defaults to 15). */
  minLeftPanelPct?: number;
  /** Maximum left panel width in percent (defaults to 60). */
  maxLeftPanelPct?: number;
  /** Extra props spread onto the layout container (e.g. tour anchors). */
  containerProps?: ResizableSplitViewPanelProps;
  /** Extra props spread onto the left panel wrapper. */
  leftPanelProps?: ResizableSplitViewPanelProps;
  /** Extra props spread onto the right panel wrapper. */
  rightPanelProps?: ResizableSplitViewPanelProps;
}

/**
 * Responsive two-panel layout shared by the Volumes, Shares and Users pages.
 *
 * - Mobile (xs): panels stack vertically full-width, matching the Shares layout.
 * - sm and up: panels sit side by side and the left panel width can be resized
 *   by dragging the divider (or with the ArrowLeft/ArrowRight keys), matching
 *   the Volumes layout. The chosen width is persisted per `storageKey`.
 */
export function ResizableSplitView({
  storageKey,
  leftPanel,
  rightPanel,
  defaultLeftPanelPct = DEFAULT_LEFT_PANEL_PCT,
  minLeftPanelPct = MIN_LEFT_PANEL_PCT,
  maxLeftPanelPct = MAX_LEFT_PANEL_PCT,
  containerProps,
  leftPanelProps,
  rightPanelProps,
}: ResizableSplitViewProps) {
  const theme = useTheme();
  const isDesktop = useMediaQuery(theme.breakpoints.up("sm"));

  const [leftPanelPct, setLeftPanelPct] = useState<number>(() => {
    try {
      const saved = localStorage.getItem(storageKey);
      if (saved) {
        const pct = parseFloat(saved);
        if (
          !Number.isNaN(pct) &&
          pct >= minLeftPanelPct &&
          pct <= maxLeftPanelPct
        ) {
          return pct;
        }
      }
    } catch {}
    return defaultLeftPanelPct;
  });

  const isDragging = useRef(false);
  const containerRef = useRef<HTMLDivElement | null>(null);

  const applyPanelPct = useCallback(
    (pct: number) => {
      const clamped = Math.min(maxLeftPanelPct, Math.max(minLeftPanelPct, pct));
      setLeftPanelPct(clamped);
    },
    [maxLeftPanelPct, minLeftPanelPct],
  );

  const handleDividerMouseDown = useCallback((e: ReactMouseEvent) => {
    e.preventDefault();
    isDragging.current = true;
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
  }, []);

  const handleDividerKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLDivElement>) => {
      if (e.key !== "ArrowLeft" && e.key !== "ArrowRight") return;
      e.preventDefault();
      const delta =
        e.key === "ArrowLeft"
          ? -RESIZE_KEYBOARD_STEP_PCT
          : RESIZE_KEYBOARD_STEP_PCT;
      applyPanelPct(leftPanelPct + delta);
    },
    [applyPanelPct, leftPanelPct],
  );

  useEffect(() => {
    const handleMouseMove = (e: globalThis.MouseEvent) => {
      if (!isDragging.current || !containerRef.current) return;
      const rect = containerRef.current.getBoundingClientRect();
      const offsetX = e.clientX - rect.left;
      const pct = (offsetX / rect.width) * 100;
      applyPanelPct(pct);
    };

    const handleMouseUp = () => {
      if (!isDragging.current) return;
      isDragging.current = false;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };

    window.addEventListener("mousemove", handleMouseMove);
    window.addEventListener("mouseup", handleMouseUp);
    return () => {
      window.removeEventListener("mousemove", handleMouseMove);
      window.removeEventListener("mouseup", handleMouseUp);
    };
  }, [applyPanelPct]);

  useEffect(() => {
    try {
      localStorage.setItem(storageKey, String(leftPanelPct));
    } catch {}
  }, [storageKey, leftPanelPct]);

  // Mobile: stack the panels full-width (Shares-style layout).
  if (!isDesktop) {
    return (
      <Box
        {...containerProps}
        sx={{
          display: "flex",
          flexDirection: "column",
          gap: 2,
          minHeight: "calc(100vh - 200px)",
        }}
      >
        <Box
          {...leftPanelProps}
          sx={{
            width: "100%",
          }}
        >
          {leftPanel}
        </Box>
        <Box
          {...rightPanelProps}
          sx={{
            width: "100%",
          }}
        >
          {rightPanel}
        </Box>
      </Box>
    );
  }

  // Desktop: side-by-side panels with a resizable divider (Volumes-style).
  return (
    <Box
      ref={containerRef}
      {...containerProps}
      sx={{
        display: "flex",
        minHeight: "calc(100vh - 200px)",
        gap: 1,
      }}
    >
      <Box
        {...leftPanelProps}
        sx={{
          width: `${leftPanelPct}%`,
          minWidth: 0,
          flexShrink: 0,
        }}
      >
        {leftPanel}
      </Box>

      <Box
        role="separator"
        aria-orientation="vertical"
        aria-label="Resize panels"
        aria-valuenow={Math.round(leftPanelPct)}
        aria-valuemin={minLeftPanelPct}
        aria-valuemax={maxLeftPanelPct}
        tabIndex={0}
        onMouseDown={handleDividerMouseDown}
        onKeyDown={handleDividerKeyDown}
        sx={{
          width: 6,
          flexShrink: 0,
          cursor: "col-resize",
          backgroundColor: "divider",
          borderRadius: 1,
          alignSelf: "stretch",
          transition: "background-color 0.15s",
          "&:hover": {
            backgroundColor: "primary.main",
          },
        }}
      />

      <Box
        {...rightPanelProps}
        sx={{
          flex: 1,
          minWidth: 0,
        }}
      >
        {rightPanel}
      </Box>
    </Box>
  );
}
