import { Alert, DialogContent, Typography } from "@mui/material";
import {
  AutocompleteElement,
  SelectElement,
  TextFieldElement,
} from "react-hook-form-mui";
import { Usage } from "../../../store/sratApi";
import type { WizardPartitionOption } from "../utils";

interface FirstShareStepContentProps {
  availablePartitions: WizardPartitionOption[];
  hasAvailablePartitions: boolean;
  isVolumesLoading: boolean;
  selectedPartitionId: string;
  /** When true, the partition autocomplete is disabled (selection is pre-locked). */
  lockPartition?: boolean;
}

export function FirstShareStepContent({
  availablePartitions,
  hasAvailablePartitions,
  isVolumesLoading,
  selectedPartitionId,
  lockPartition = false,
}: FirstShareStepContentProps) {
  return (
    <DialogContent>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Select a partition to mount, choose your first share name, and set how
        Home Assistant should use it.
      </Typography>
      {!hasAvailablePartitions && !isVolumesLoading && (
        <Alert severity="info" sx={{ mb: 2 }}>
          No available partitions were found to mount. This usually means all
          partitions are already mounted or only system partitions are present.
        </Alert>
      )}
      <AutocompleteElement
        label="Partition to Mount"
        name="partitionId"
        options={availablePartitions.map((partition) => ({
          id: partition.partitionId,
          label: partition.displayName,
        }))}
        loading={isVolumesLoading}
        matchId
        autocompleteProps={{
          disabled: !hasAvailablePartitions || lockPartition,
          size: "small",
        }}
        textFieldProps={{
          sx: { mb: 2 },
          helperText: hasAvailablePartitions
            ? "Choose which partition should be mounted and used for the share"
            : "No mountable partition available",
        }}
      />
      <TextFieldElement
        name="shareName"
        label="Share Name"
        fullWidth
        placeholder="e.g., Media"
        disabled={!selectedPartitionId}
        sx={{ mb: 2 }}
        rules={{
          validate: (value) =>
            !selectedPartitionId ||
            (value && value.trim().length > 0) ||
            "Share Name is required",
        }}
        helperText="You can add and configure shares later in the Shares tab"
      />
      <SelectElement
        name="usage"
        label="Home Assistant Usage"
        options={[
          { id: Usage.None, label: "None" },
          { id: Usage.Backup, label: "Backup" },
          { id: Usage.Media, label: "Media" },
          { id: Usage.Share, label: "File Share" },
        ]}
        required
        disabled={!hasAvailablePartitions || !selectedPartitionId}
        helperText="How this share is intended to be used in Home Assistant"
      />
    </DialogContent>
  );
}
