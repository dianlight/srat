import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogTitle,
} from "@mui/material";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { FormContainer } from "react-hook-form-mui";
import { FirstShareStepContent } from "../../../components/wizard/steps/FirstShareStepContent";
import type { FirstShareFormData } from "../../../components/wizard/types";
import type { WizardPartitionOption } from "../../../components/wizard/utils";
import { sanitizeWizardShareName } from "../../../components/wizard/utils";
import {
  type MountPointData,
  type Partition,
  Type,
  Usage,
  usePostApiShareMutation,
  usePostApiVolumeMountMutation,
} from "../../../store/sratApi";

interface MountShareWizardProps {
  open: boolean;
  onClose: () => void;
  partition: Partition;
  action: "mount" | "share";
}

/**
 * Thin public wrapper enforcing the guard-before-hooks rule.
 * Hooks only run when the dialog is actually open.
 */
export function MountShareWizard(props: MountShareWizardProps) {
  if (!props.open) {
    return null;
  }
  return <MountShareWizardInner {...props} />;
}

function MountShareWizardInner({
  open,
  onClose,
  partition,
  action,
}: MountShareWizardProps) {
  const [mountVolume] = usePostApiVolumeMountMutation();
  const [createShare] = usePostApiShareMutation();

  const suggestedName = sanitizeWizardShareName(
    partition.name || partition.legacy_device_name || partition.id || "Share",
  );

  const formContext = useForm<FirstShareFormData>({
    defaultValues: {
      partitionId: partition.id ?? "",
      shareName: suggestedName,
      usage: Usage.None,
    },
  });

  const { setValue, setError, formState } = formContext;

  // Sync partition selection whenever the dialog opens or partition changes.
  useEffect(() => {
    if (!open) return;
    setValue("partitionId", partition.id ?? "", { shouldDirty: false });
    setValue("shareName", suggestedName, { shouldDirty: false });
  }, [open, partition.id, suggestedName, setValue]);

  // Build a single-option list so FirstShareStepContent renders the locked selection.
  const lockedOption: WizardPartitionOption = {
    partitionId: partition.id ?? "",
    displayName:
      partition.name ||
      partition.legacy_device_name ||
      partition.legacy_device_path ||
      partition.device_path ||
      partition.id ||
      "Unknown Partition",
    suggestedShareName: suggestedName,
    partition,
  };

  const handleSubmit = async (data: FirstShareFormData) => {
    try {
      let mountPointData: MountPointData;

      if (action === "mount") {
        const mountPath = `/mnt/${sanitizeWizardShareName(partition.name || partition.id || "share")}`;
        mountPointData = (await mountVolume({
          mountPointData: {
            device_id: partition.id,
            path: mountPath,
            root: "/",
            type: Type.Addon,
            is_to_mount_at_startup: true,
          },
        }).unwrap()) as MountPointData;
      } else {
        // "share" action — partition already mounted; use existing mount point.
        const existing = Object.values(partition.mount_point_data || {})[0];
        if (!existing) {
          setError("root", {
            message: "Could not find mount point data for this partition.",
          });
          return;
        }
        mountPointData = existing as MountPointData;
      }

      const shareName = data.shareName?.trim();
      if (shareName) {
        await createShare({
          sharedResourcePostData: {
            name: shareName,
            usage: data.usage || Usage.None,
            mount_point_data: mountPointData,
          },
        }).unwrap();
      }

      onClose();
    } catch {
      setError("root", {
        message:
          "Failed to complete the operation. Please try again or use the Volumes / Shares tabs.",
      });
    }
  };

  const title = action === "mount" ? "Mount & Share Partition" : "Create Share";
  const submitLabel = action === "mount" ? "Mount & Share" : "Create Share";

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="sm"
      fullWidth
      aria-labelledby="mount-share-wizard-title"
    >
      <FormContainer formContext={formContext} onSuccess={handleSubmit}>
        <DialogTitle id="mount-share-wizard-title">{title}</DialogTitle>
        <FirstShareStepContent
          availablePartitions={[lockedOption]}
          hasAvailablePartitions={true}
          isVolumesLoading={false}
          selectedPartitionId={partition.id ?? ""}
          lockPartition
        />
        {formState.errors.root && (
          <Alert severity="error" sx={{ mx: 3, mb: 1 }}>
            {formState.errors.root.message}
          </Alert>
        )}
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button type="button" onClick={onClose}>
            Cancel
          </Button>
          <Button
            type="submit"
            variant="contained"
            disabled={formState.isSubmitting}
            loading={formState.isSubmitting}
          >
            {formState.isSubmitting ? "Saving…" : submitLabel}
          </Button>
        </DialogActions>
      </FormContainer>
    </Dialog>
  );
}
