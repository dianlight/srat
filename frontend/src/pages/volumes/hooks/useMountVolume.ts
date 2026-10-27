import * as Sentry from "@sentry/react";
import { useConfirm } from "material-ui-confirm";
import { useCallback } from "react";
import { toast } from "react-toastify";
import {
  type MountPointData,
  type Partition,
  usePostApiVolumeMountMutation,
} from "../../../store/sratApi";
import {
  extractSuggestedMountPath,
  parseVolumeApiError,
  serializeErrorForSentry,
} from "../utils";

function isMountPointData(value: unknown): value is MountPointData {
  return (
    typeof value === "object" &&
    value !== null &&
    "path" in value &&
    typeof (value as { path?: unknown }).path === "string"
  );
}

export type MountFormSetError = (
  name: "root",
  error: { message: string },
) => void;

export interface UseMountVolumeParams {
  selectedPartition?: Partition;
  onCleared?: () => void;
}

export interface UseMountVolumeResult {
  onSubmitMountVolume: (
    data?: MountPointData,
    setError?: MountFormSetError,
  ) => Promise<void>;
  isMounting: boolean;
}

export function useMountVolume({
  selectedPartition,
  onCleared,
}: UseMountVolumeParams = {}): UseMountVolumeResult {
  const confirm = useConfirm();
  const [mountVolume, mountResult] = usePostApiVolumeMountMutation();

  const onSubmitMountVolume = useCallback(
    async (data?: MountPointData, setError?: MountFormSetError) => {
      if (!selectedPartition || !data?.path || !data.root) {
        const message = "Cannot mount: Invalid selection or missing data.";
        toast.error(message);
        console.error("Mount validation failed:", {
          selectedPartition,
          data,
        });
        setError?.("root", { message });
        return;
      }

      const submitData: MountPointData = {
        ...data,
        device_id: selectedPartition.id,
      };

      const showMountError = (
        errorData: unknown,
        err: unknown,
        attemptedPath?: string,
      ) => {
        const path = attemptedPath ?? submitData.path;
        const parsed =
          errorData && typeof errorData === "object"
            ? parseVolumeApiError({
                data: errorData,
                status: (err as { status?: unknown })?.status,
              })
            : parseVolumeApiError(err);
        const mountError = new Error(`Mount failed: ${parsed.message}`);
        (mountError as { cause?: unknown }).cause = err;
        Sentry.addBreadcrumb({
          category: "volume.mount",
          message: `Mount ${path ?? "unknown path"}`,
          data: {
            path,
            root: submitData.root,
            device: selectedPartition?.id,
            code: parsed.code,
          },
          level: "error",
        });
        Sentry.captureException(mountError, {
          contexts: {
            volume_mount: {
              path,
              root: submitData.root,
              device: selectedPartition?.id,
              ...serializeErrorForSentry(err),
            },
          },
        });
        // #1354: log the Error itself first so the console→Sentry
        // forwarder titles on `Mount failed: <detail>` instead of a bare
        // "Mount Error:" with empty suffix.
        console.error(mountError, {
          path,
          device: selectedPartition?.id,
        });
        const message = `${String(parsed.code)}: ${parsed.message}`;
        toast.error(message, {
          data: { error: (errorData ?? err) as unknown },
        });
        setError?.("root", { message });
      };

      const attemptMount = (
        payload: MountPointData,
        allowSuggestionRetry: boolean,
      ): Promise<void> =>
        mountVolume({
          mountPointData: payload,
        })
          .unwrap()
          .then((res) => {
            const mountedPath = isMountPointData(res)
              ? res.path
              : selectedPartition.name;
            toast.info(`Volume ${mountedPath} mounted successfully.`);
          })
          .catch((err): Promise<void> => {
            const errorData = err?.data || {};
            const suggested = extractSuggestedMountPath(errorData);
            if (
              allowSuggestionRetry &&
              suggested &&
              suggested !== payload.path
            ) {
              return confirm({
                title: "Invalid mount path",
                description: `The path ${payload.path} contains characters the database cannot store. Retry with the suggested path ${suggested}?`,
                confirmationText: "Use suggested path",
                cancellationText: "Cancel",
              }).then(({ reason }): Promise<void> => {
                if (reason === "confirm") {
                  return attemptMount({ ...payload, path: suggested }, false);
                }
                const detail = `${payload.path} is invalid. Suggested path: ${suggested}`;
                showMountError(
                  {
                    ...(typeof errorData === "object" && errorData !== null
                      ? errorData
                      : {}),
                    detail,
                  },
                  err,
                  payload.path,
                );
                return Promise.resolve();
              });
            }
            showMountError(errorData, err, payload.path);
            return Promise.resolve();
          });

      try {
        await attemptMount(submitData, true);
      } finally {
        onCleared?.();
      }
    },
    [confirm, mountVolume, onCleared, selectedPartition],
  );

  return { onSubmitMountVolume, isMounting: mountResult.isLoading };
}
