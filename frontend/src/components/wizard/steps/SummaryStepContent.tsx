import DoneIcon from "@mui/icons-material/Done";
import HourglassEmptyIcon from "@mui/icons-material/HourglassEmpty";
import { DialogContent, Divider, Stack, Typography } from "@mui/material";
import { useEffect, useState } from "react";
import { type DataDirtyTracker, Telemetry_mode } from "../../../store/sratApi";
import { IconProgress } from "../../IconProgress";
import type { WizardCollectedData } from "../types";
import type { WizardPartitionOption } from "../utils";

interface SummaryStepContentProps {
  data: WizardCollectedData;
  selectedPartition?: WizardPartitionOption;
  isProcessing: boolean;
  dirtyTracking?: DataDirtyTracker;
}

function formatTelemetryMode(mode: Telemetry_mode | undefined): string {
  switch (mode) {
    case Telemetry_mode.All:
      return "Send usage data and error reports";
    case Telemetry_mode.Errors:
      return "Send only error reports";
    case Telemetry_mode.Disabled:
      return "Do not send telemetry";
    default:
      return "Use the default telemetry mode";
  }
}

function formatLabMode(enabled: boolean | undefined): string {
  return enabled
    ? "Enabled — experimental lab features will be visible"
    : "Disabled — only stable features will be shown";
}

const isDirtyTrackingClean = (dirtyTracking: DataDirtyTracker | undefined) => {
  if (!dirtyTracking) {
    return false;
  }

  return Object.values(dirtyTracking).every((value) => !value);
};

export function SummaryStepContent({
  data,
  selectedPartition,
  isProcessing,
  dirtyTracking,
}: SummaryStepContentProps) {
  const [processingFinalStatus, setProcessingFinalStatus] = useState(0);
  useEffect(() => {
    if (isProcessing && !isDirtyTrackingClean(dirtyTracking)) {
      setProcessingFinalStatus(100);
    }
  }, [isProcessing, dirtyTracking]);
  return (
    <DialogContent>
      <Stack spacing={2}>
        <Typography variant="body2" color="text.secondary">
          Review the selected settings before SRAT applies them.
        </Typography>

        <Stack spacing={1}>
          <Typography variant="subtitle2">Security</Typography>
          <Typography variant="body2" color="text.secondary" component="div">
            {isProcessing && (
              <IconProgress
                icons={[HourglassEmptyIcon]}
                animationSpeed={700}
                completeIcon={DoneIcon}
                completeIconColor="success"
                variant="determinate"
                value={dirtyTracking?.settings ? 50 : processingFinalStatus}
              />
            )}
            Hostname: {data.security?.hostname || "Not set"}
          </Typography>
          <Typography variant="body2" color="text.secondary" component="div">
            {isProcessing && (
              <IconProgress
                icons={[HourglassEmptyIcon]}
                animationSpeed={700}
                completeIcon={DoneIcon}
                completeIconColor="success"
                variant="determinate"
                value={dirtyTracking?.settings ? 50 : processingFinalStatus}
              />
            )}
            Workgroup: {data.security?.workgroup || "Not set"}
          </Typography>
          <Typography variant="body2" color="text.secondary" component="div">
            {isProcessing && (
              <IconProgress
                icons={[HourglassEmptyIcon]}
                animationSpeed={700}
                completeIcon={DoneIcon}
                completeIconColor="success"
                variant="determinate"
                value={dirtyTracking?.users ? 50 : processingFinalStatus}
              />
            )}
            Password:{" "}
            {data.security?.newPassword
              ? "Will be updated"
              : "Keep current password"}
          </Typography>
        </Stack>

        <Divider />

        <Stack spacing={1}>
          <Typography variant="subtitle2">Network</Typography>
          <Typography variant="body2" color="text.secondary" component="div">
            {isProcessing && (
              <IconProgress
                icons={[HourglassEmptyIcon]}
                animationSpeed={700}
                completeIcon={DoneIcon}
                completeIconColor="success"
                variant="determinate"
                value={dirtyTracking?.settings ? 50 : processingFinalStatus}
              />
            )}
            {data.network?.bind_all_interfaces
              ? "Samba will bind to all interfaces"
              : `Interfaces: ${data.network?.interfaces?.join(", ") || "Use current selection"}`}
          </Typography>
        </Stack>

        <Divider />

        <Stack spacing={1}>
          <Typography variant="subtitle2">First Share</Typography>
          {data.firstShare?.partitionId && selectedPartition ? (
            <>
              <Typography
                variant="body2"
                color="text.secondary"
                component="div"
              >
                {isProcessing && (
                  <IconProgress
                    icons={[HourglassEmptyIcon]}
                    animationSpeed={700}
                    completeIcon={DoneIcon}
                    completeIconColor="success"
                    variant="determinate"
                    value={dirtyTracking?.shares ? 50 : processingFinalStatus}
                  />
                )}
                Partition: {selectedPartition.displayName}
              </Typography>
              <Typography
                variant="body2"
                color="text.secondary"
                component="div"
              >
                {isProcessing && (
                  <IconProgress
                    icons={[HourglassEmptyIcon]}
                    animationSpeed={700}
                    completeIcon={DoneIcon}
                    completeIconColor="success"
                    variant="determinate"
                    value={dirtyTracking?.shares ? 50 : processingFinalStatus}
                  />
                )}
                Share name: {data.firstShare.shareName || "Not set"}
              </Typography>
              <Typography
                variant="body2"
                color="text.secondary"
                component="div"
              >
                {isProcessing && (
                  <IconProgress
                    icons={[HourglassEmptyIcon]}
                    animationSpeed={700}
                    completeIcon={DoneIcon}
                    completeIconColor="success"
                    variant="determinate"
                    value={dirtyTracking?.shares ? 50 : processingFinalStatus}
                  />
                )}
                Usage: {data.firstShare.usage}
              </Typography>
            </>
          ) : (
            <Typography variant="body2" color="text.secondary">
              No first share will be configured right now.
            </Typography>
          )}
        </Stack>

        <Divider />

        <Stack spacing={1}>
          <Typography variant="subtitle2">Telemetry</Typography>
          <Typography variant="body2" color="text.secondary" component="div">
            {isProcessing && (
              <IconProgress
                icons={[HourglassEmptyIcon]}
                animationSpeed={700}
                completeIcon={DoneIcon}
                completeIconColor="success"
                variant="determinate"
                value={dirtyTracking?.settings ? 50 : processingFinalStatus}
              />
            )}
            {formatTelemetryMode(data.telemetry?.telemetry_mode)}
          </Typography>
        </Stack>

        <Divider />

        <Stack spacing={1}>
          <Typography variant="subtitle2">Experimental Lab</Typography>
          <Typography variant="body2" color="text.secondary" component="div">
            {isProcessing && (
              <IconProgress
                icons={[HourglassEmptyIcon]}
                animationSpeed={700}
                completeIcon={DoneIcon}
                completeIconColor="success"
                variant="determinate"
                value={dirtyTracking?.settings ? 50 : processingFinalStatus}
              />
            )}
            {formatLabMode(data.labMode?.experimental_lab_mode)}
          </Typography>
        </Stack>
      </Stack>
    </DialogContent>
  );
}
