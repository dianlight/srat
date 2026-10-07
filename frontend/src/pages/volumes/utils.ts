import * as Sentry from "@sentry/react";
import type { useConfirm } from "material-ui-confirm";
import type { NavigateFunction } from "react-router";
import { toast } from "react-toastify";
import { type LocationState, TabIDs } from "../../store/locationState";
import type {
  Disk,
  Partition,
  PatchApiVolumeSettingsApiArg,
} from "../../store/sratApi";
import { decodeEscapeSequence } from "../../utils/decodeEscapeSequence";

export { decodeEscapeSequence } from "../../utils/decodeEscapeSequence";

export function updatePartitionLabelInDisks(
  disks: Disk[] | undefined,
  partitionId: string,
  label: string,
): Disk[] {
  if (!Array.isArray(disks) || disks.length === 0) {
    return [];
  }

  let hasChanges = false;
  const nextDisks = disks.map((disk) => {
    const partitionEntries = Object.entries(disk.partitions || {});
    if (partitionEntries.length === 0) {
      return disk;
    }

    let diskChanged = false;
    const nextPartitions = Object.fromEntries(
      partitionEntries.map(([key, partition]) => {
        if (
          !partition ||
          partition.id !== partitionId ||
          partition.name === label
        ) {
          return [key, partition];
        }

        diskChanged = true;
        hasChanges = true;
        return [key, { ...partition, name: label }];
      }),
    );

    return diskChanged ? { ...disk, partitions: nextPartitions } : disk;
  });

  return hasChanges ? nextDisks : disks;
}

export function getFilesystemLabelValidation(
  label: string,
  labelRule: unknown,
  optional = false,
): {
  isValid: boolean;
  helperText?: string;
} {
  const normalizedRule = typeof labelRule === "string" ? labelRule.trim() : "";
  const normalizedLabel = label.trim();

  if (!normalizedRule) {
    return {
      isValid: optional || normalizedLabel.length > 0,
    };
  }

  const acceptedFormatHint = `Accepted format: ${normalizedRule}`;

  if (normalizedLabel.length === 0) {
    return {
      isValid: optional,
      helperText: acceptedFormatHint,
    };
  }

  try {
    const isValid = new RegExp(normalizedRule).test(normalizedLabel);
    return {
      isValid,
      helperText: isValid
        ? acceptedFormatHint
        : `Invalid label. ${acceptedFormatHint}`,
    };
  } catch {
    return {
      isValid: true,
      helperText: acceptedFormatHint,
    };
  }
}

export function getDiskIdentifier(disk: Disk, fallbackIndex: number): string {
  const stableId =
    disk.id || disk.legacy_device_name || disk.device_path || disk.serial;
  if (stableId) return stableId;
  // Backend defect indicator: a disk with none of id/legacy_device_name/
  // device_path/serial would re-point persisted selection/expansion state
  // whenever the snapshot order changes, so make the fallback visible.
  console.warn(
    `Disk #${fallbackIndex} has no stable identifier (id/legacy_device_name/device_path/serial); using index-based key "disk-${fallbackIndex}" which is not stable across reordering.`,
  );
  return `disk-${fallbackIndex}`;
}

export function getPartitionIdentifier(
  diskIdentifier: string,
  partition: Partition,
  partitionKey: string | undefined,
  fallbackIndex: number,
): string {
  const stableBase =
    partition.id ||
    partition.uuid ||
    partition.device_path ||
    partition.legacy_device_name ||
    partition.legacy_device_path ||
    partitionKey;
  if (stableBase) return `${diskIdentifier}::${stableBase}`;
  // Same backend-defect indicator as getDiskIdentifier: warn when only the
  // index-based fallback is available.
  console.warn(
    `Partition #${fallbackIndex} has no stable identifier (id/uuid/device_path/legacy names/key); using index-based key "part-${fallbackIndex}" which is not stable across reordering.`,
  );
  return `${diskIdentifier}::part-${fallbackIndex}`;
}

export function getMountpointIdentifier(
  partitionIdentifier: string,
  mountpointKey: string,
): string {
  return `${partitionIdentifier}::mp::${mountpointKey}`;
}

// Task 044: the backend synthesizes a whole-disk partition entry for raw disks
// (no partition table) so they stay actionable (mount/unmount/format). The
// backend may legitimately detect a filesystem written directly to the whole
// disk (e.g. vfat superfloppy). Such an entry always carries the disk's own
// legacy device name (real partitions have a numeric suffix, like sdc1), so
// name equality is the discriminator. It is not a real partition and must not
// be counted or labeled as one.
export function isSynthesizedWholeDiskPartition(
  disk: Disk,
  partition: Partition,
): boolean {
  return (
    partition.legacy_device_name != null &&
    partition.legacy_device_name === disk.legacy_device_name
  );
}

export function getRealPartitions(disk: Disk): Partition[] {
  return Object.values(disk.partitions || {}).filter(
    (partition) => !isSynthesizedWholeDiskPartition(disk, partition),
  );
}

// Shared RTK/Huma API error parsing (#1344). Backend volume errors surface as
// Huma problem payloads in `err.data` with the human message in `detail`
// (mount 422/406 embeds service Details as "Key: value" lines, unmount 406
// sends the Detail value). `message` is a fallback, then HTTP status.
export interface ParsedVolumeApiError {
  message: string;
  code: string | number;
  detail?: string;
  status?: number;
  errorData: Record<string, unknown>;
}

/** Returns the trimmed string, or undefined when blank/non-string. */
function nonBlank(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() ? value.trim() : undefined;
}

/** First non-blank `message` inside a Huma `errors` array, if any. */
function firstNestedMessage(
  payload: Record<string, unknown>,
): string | undefined {
  const nested = payload.errors;
  if (!Array.isArray(nested)) return undefined;
  for (const entry of nested) {
    if (entry && typeof entry === "object") {
      const message = nonBlank((entry as Record<string, unknown>).message);
      if (message) return message;
    }
  }
  return undefined;
}

/**
 * Parses an RTK Query rejection into a display-ready message plus the
 * backend `detail`, HTTP status, and raw payload for Sentry context.
 */
export function parseVolumeApiError(err: unknown): ParsedVolumeApiError {
  const errorData =
    typeof err === "object" && err !== null && "data" in err
      ? ((err as { data?: unknown }).data ?? {})
      : {};
  const payload =
    typeof errorData === "object" && errorData !== null
      ? (errorData as Record<string, unknown>)
      : {};
  const rawStatus =
    typeof err === "object" && err !== null && "status" in err
      ? (err as { status?: unknown }).status
      : payload.status;
  const status = typeof rawStatus === "number" ? rawStatus : undefined;
  const detail = nonBlank(payload.detail);
  const fallbackMessage =
    nonBlank(payload.message) ?? firstNestedMessage(payload);
  const message =
    detail ??
    fallbackMessage ??
    (status !== undefined ? String(status) : "Unknown error");
  const code =
    status ?? (typeof payload.status === "string" ? payload.status : "Error");
  return {
    message: message || "Unknown error",
    code,
    detail,
    status,
    errorData: payload,
  };
}

/**
 * Serializes an RTK error for Sentry context without producing "[Object]"
 * placeholders. JSON-safe, truncates long strings, preserves detail/message.
 */
export function serializeErrorForSentry(err: unknown): Record<string, unknown> {
  const parsed = parseVolumeApiError(err);
  const seen = new Set<unknown>();
  const safe = (value: unknown, depth = 0): unknown => {
    if (depth > 3) return "[truncated]";
    if (value instanceof Error) {
      return { name: value.name, message: value.message, stack: value.stack };
    }
    if (
      typeof value === "string" ||
      typeof value === "number" ||
      typeof value === "boolean" ||
      value == null
    ) {
      return typeof value === "string" && value.length > 2000
        ? `${value.slice(0, 2000)}…[truncated]`
        : value;
    }
    if (typeof value !== "object") return String(value);
    if (seen.has(value)) return "[circular]";
    seen.add(value);
    if (Array.isArray(value)) return value.map((v) => safe(v, depth + 1));
    const out: Record<string, unknown> = {};
    for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
      try {
        out[k] = safe(v, depth + 1);
      } catch {
        out[k] = String(v);
      }
    }
    return out;
  };
  return {
    message: parsed.message,
    code: parsed.code,
    detail: parsed.detail,
    status: parsed.status,
    error: safe(err),
  };
}

// extractSuggestedMountPath parses a mount-failure payload for the backend's
// SuggestedPath hint (#1091). The 406 detail embeds service Details as
// "Key: value" lines, so scan detail plus any nested error messages.
export function extractSuggestedMountPath(
  errorData: unknown,
): string | undefined {
  if (!errorData || typeof errorData !== "object") return undefined;
  const data = errorData as Record<string, unknown>;
  const texts: string[] = [];
  if (typeof data.detail === "string") texts.push(data.detail);
  if (typeof data.message === "string") texts.push(data.message);
  const nested = data.errors;
  if (Array.isArray(nested)) {
    for (const entry of nested) {
      if (entry && typeof entry === "object") {
        const message = (entry as Record<string, unknown>).message;
        if (typeof message === "string") texts.push(message);
      }
    }
  }
  for (const text of texts) {
    const match = /SuggestedPath:\s*(\S+)/.exec(text);
    if (match?.[1]) return match[1];
  }
  return undefined;
}

export function findPartitionByMountPath(
  sourceDisks: Disk[] | undefined,
  mountPath: string,
): { disk: Disk; partition: Partition } | undefined {
  return (sourceDisks ?? [])
    .flatMap((disk) =>
      Object.values(disk.partitions || {}).map((partition) => ({
        disk,
        partition,
      })),
    )
    .find(({ partition }) =>
      Object.values(partition.mount_point_data || {}).some(
        (mpd) => mpd.path === mountPath,
      ),
    );
}

export function navigateToCreateShare(
  navigate: NavigateFunction,
  partition: Partition,
): void {
  const firstMountPointData = Object.values(
    partition.mount_point_data || {},
  )[0];
  if (firstMountPointData?.path) {
    navigate("/", {
      state: {
        tabId: TabIDs.SHARES,
        newShareData: firstMountPointData,
      } as LocationState,
    });
  } else {
    toast.warn(
      "Cannot create share: Partition is not mounted or has no mount path.",
    );
  }
}

export function navigateToShare(
  navigate: NavigateFunction,
  partition: Partition,
): void {
  const share = Object.values(partition.mount_point_data || {})[0]?.share;
  if (share?.name) {
    navigate("/", {
      state: { tabId: TabIDs.SHARES, shareName: share.name } as LocationState,
    });
  }
}

export type ConfirmDialog = ReturnType<typeof useConfirm>;

export type UnmountTrigger = (args: { mountPath: string; force: boolean }) => {
  unwrap(): Promise<unknown>;
};

export function requestUnmountVolume(args: {
  confirm: ConfirmDialog;
  unmount: UnmountTrigger;
  partition: Partition;
  force?: boolean;
  isSelected: boolean;
  onCleared: () => void;
}): void {
  const { confirm, unmount, partition, isSelected, onCleared } = args;
  const force = args.force ?? false;
  console.debug("Umount Request", partition, "Force:", force);
  const mountData = Object.values(partition.mount_point_data || {})[0];
  if (!mountData?.path) {
    toast.error("Cannot unmount: Missing mount point path.");
    console.error("Missing mount path for partition:", partition);
    return;
  }
  const displayName = decodeEscapeSequence(partition.name || "this volume");
  const mountPath = mountData.path;
  confirm({
    title: `Unmount ${displayName}?`,
    description: `Do you really want to ${force ? "forcefully " : ""}unmount the Volume ${displayName} (${partition.legacy_device_name}) mounted at ${mountPath}?`,
    confirmationText: force ? "Force Unmount" : "Unmount",
    cancellationText: "Cancel",
    confirmationButtonProps: { color: force ? "error" : "primary" },
    acknowledgement: `Please confirm this action carefully. Unmounting may lead to data loss or corruption if the volume is in use. ${force ? "NOTE:Configured shares will be disabled!" : ""}`,
  }).then(({ reason }) => {
    if (reason !== "confirm") return;
    unmount({ mountPath, force })
      .unwrap()
      .then(() => {
        toast.info(`Volume ${displayName} unmounted successfully.`);
        if (isSelected) onCleared();
      })
      .catch((err) => {
        const parsed = parseVolumeApiError(err);
        const sentryError = new Error(
          `Unmount failed for ${displayName}: ${parsed.message}`,
        );
        (sentryError as { cause?: unknown }).cause = err;
        Sentry.addBreadcrumb({
          category: "volume.unmount",
          message: `Unmount ${displayName}`,
          data: {
            mountPath,
            force,
            device: partition.id ?? partition.legacy_device_name,
            code: parsed.code,
          },
          level: "error",
        });
        Sentry.captureException(sentryError, {
          contexts: {
            volume_unmount: {
              mountPath,
              force,
              device: partition.id ?? partition.legacy_device_name,
              ...serializeErrorForSentry(err),
            },
          },
        });
        console.error(
          `Unmount Error for ${displayName} at ${mountPath}:`,
          sentryError,
        );
        toast.error(
          `Error unmounting ${displayName}: ${String(parsed.code)}: ${parsed.message}`,
          { data: { error: err } },
        );
      });
  });
}

export type PatchSettingsTrigger = (args: PatchApiVolumeSettingsApiArg) => {
  unwrap(): Promise<unknown>;
};

export function requestAutomountToggle(args: {
  patchSettings: PatchSettingsTrigger;
  readOnly: boolean;
  partition: Partition;
}): void {
  const { patchSettings, readOnly, partition } = args;
  if (readOnly) return;
  console.debug("Toggling automount for partition:", partition);
  const partitionName = decodeEscapeSequence(partition.name || "this volume");
  const mountEntries = Object.entries(partition.mount_point_data || {});
  if (mountEntries.length === 0) return;
  // Fire one PATCH per mount point and aggregate the results so a
  // multi-mount partition produces a single summary toast instead of
  // N interleaved toasts and uncoordinated failure handling.
  const patchPromises = mountEntries.map(async ([path, mountData]) => {
    if (!mountData.path) {
      throw new Error(`Cannot toggle automount: ${path} Missing point data.`);
    }
    const newAutomountState = !mountData.is_to_mount_at_startup;
    console.debug(
      partition,
      mountData,
      "Toggling automount to",
      newAutomountState,
    );
    await patchSettings({
      patchMountPointData: {
        ...mountData,
        is_to_mount_at_startup: newAutomountState,
        share: undefined,
      },
    }).unwrap();
  });
  void Promise.allSettled(patchPromises).then((settled) => {
    const ok = settled.filter((r) => r.status === "fulfilled").length;
    const failed = settled.length - ok;
    settled.forEach((result, index) => {
      if (result.status === "rejected") {
        console.error(
          `Error toggling automount for ${partitionName} (${mountEntries[index][0]}):`,
          result.reason,
        );
      }
    });
    if (failed === 0) {
      toast.info(`Automount updated for ${partitionName}.`);
    } else if (ok === 0) {
      toast.error(
        `Failed to update automount for ${partitionName} (${failed} mount point${failed === 1 ? "" : "s"}).`,
      );
    } else {
      toast.warn(
        `Automount partially updated for ${partitionName}: ${ok} ok, ${failed} failed.`,
      );
    }
  });
}
