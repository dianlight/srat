import {
  Alert,
  Dialog,
  DialogTitle,
  Step,
  StepLabel,
  Stepper,
  Typography,
} from "@mui/material";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useForm } from "react-hook-form";
import { FormContainer } from "react-hook-form-mui";
import { useHealth } from "../../hooks/healthHook";
import {
  type DataDirtyTracker,
  type Disk,
  type ErrorModel,
  type InterfaceStat,
  type MountPointData,
  type Settings,
  Telemetry_mode,
  Type,
  Usage,
  useGetApiHostnameQuery,
  useGetApiNicsQuery,
  useGetApiSettingsQuery,
  useGetApiTelemetryInternetConnectionQuery,
  useGetApiUsersQuery,
  useGetApiVolumesQuery,
  usePostApiShareMutation,
  usePostApiVolumeMountMutation,
  usePutApiSettingsMutation,
  usePutApiUseradminMutation,
} from "../../store/sratApi";
import { SetupWizardActions } from "./SetupWizardActions";
import { FirstShareStepContent } from "./steps/FirstShareStepContent";
import { NetworkStepContent } from "./steps/NetworkStepContent";
import { SecurityStepContent } from "./steps/SecurityStepContent";
import { SummaryStepContent } from "./steps/SummaryStepContent";
import { TelemetryStepContent } from "./steps/TelemetryStepContent";
import type {
  FirstShareFormData,
  NetworkFormData,
  SecurityFormData,
  TelemetryFormData,
  WizardCollectedData,
} from "./types";
import {
  findPartitionById,
  getWizardAvailablePartitions,
  isValidSettings,
  isValidUsers,
  sanitizeWizardShareName,
} from "./utils";

export { getWizardAvailablePartitions } from "./utils";

export const WizardOpenContext = createContext<() => void>(() => {});
export const useOpenWizard = () => useContext(WizardOpenContext);

const STEP_LABELS = [
  "Security",
  "Network",
  "First Share",
  "Telemetry",
  "Summary",
];

interface SetupWizardProps {
  open: boolean;
  onClose: () => void;
  allowSkip?: boolean;
}

const WIZARD_TITLE_ID = "setup-wizard-title";
const WIZARD_DESCRIPTION_ID = "setup-wizard-description";

const isDirtyTrackingClean = (dirtyTracking: DataDirtyTracker | undefined) => {
  if (!dirtyTracking) {
    return false;
  }

  return Object.values(dirtyTracking).every((value) => !value);
};

export function SetupWizard({
  open,
  onClose,
  allowSkip = true,
}: SetupWizardProps) {
  const [activeStep, setActiveStep] = useState(0);
  const [collectedData, setCollectedData] = useState<WizardCollectedData>({});
  const [isFinishing, setIsFinishing] = useState(false);
  const [isWaitingForClean, setIsWaitingForClean] = useState(false);
  const [finishError, setFinishError] = useState<string | null>(null);

  const { data: settings } = useGetApiSettingsQuery();
  const { data: users } = useGetApiUsersQuery();
  const { data: volumes, isLoading: isVolumesLoading } =
    useGetApiVolumesQuery();
  const { data: systemHostname, isLoading: isHostnameFetching } =
    useGetApiHostnameQuery();
  const { data: nics, isLoading: isNicLoading } = useGetApiNicsQuery();
  const { data: internetConnection, isLoading: isCheckingConnection } =
    useGetApiTelemetryInternetConnectionQuery();
  const [updateSettings] = usePutApiSettingsMutation();
  const [updateAdminUser] = usePutApiUseradminMutation();
  const [mountVolume] = usePostApiVolumeMountMutation();
  const [createShare] = usePostApiShareMutation();
  const { health } = useHealth();

  const adminUser = useMemo(
    () => (isValidUsers(users) ? users.find((u) => u.is_admin) : undefined),
    [users],
  );
  const normalizedInternetConnection =
    typeof internetConnection === "boolean" ? internetConnection : undefined;
  const availablePartitions = getWizardAvailablePartitions(volumes as Disk[]);
  const hasAvailablePartitions = availablePartitions.length > 0;
  const selectedSummaryPartition = useMemo(() => {
    const partitionId = collectedData.firstShare?.partitionId;
    if (!partitionId) {
      return undefined;
    }

    return availablePartitions.find(
      (partition) => partition.partitionId === partitionId,
    );
  }, [availablePartitions, collectedData.firstShare?.partitionId]);

  const securityFormContext = useForm<SecurityFormData>({
    defaultValues: {
      hostname: "",
      workgroup: "WORKGROUP",
      newPassword: "",
      confirmPassword: "",
    },
  });
  const networkFormContext = useForm<NetworkFormData>({
    defaultValues: {
      bind_all_interfaces: true,
      interfaces: [],
    },
  });
  const firstShareFormContext = useForm<FirstShareFormData>({
    defaultValues: {
      partitionId: "",
      shareName: "",
      usage: Usage.None,
    },
  });
  const telemetryFormContext = useForm<TelemetryFormData>({
    defaultValues: { telemetry_mode: Telemetry_mode.Errors },
  });

  const bindAllInterfaces = networkFormContext.watch("bind_all_interfaces");
  const selectedPartitionId = firstShareFormContext.watch("partitionId");
  const currentShareName = firstShareFormContext.watch("shareName");

  const wasOpenRef = useRef(false);
  const closeRequestedRef = useRef(false);

  const requestClose = useCallback(() => {
    if (closeRequestedRef.current) {
      return;
    }

    closeRequestedRef.current = true;
    onClose();
  }, [onClose]);

  useEffect(() => {
    const wasOpen = wasOpenRef.current;
    wasOpenRef.current = open;

    if (!open) {
      return;
    }

    // Reset navigation only when dialog transitions from closed → open
    if (!wasOpen) {
      setActiveStep(0);
      setCollectedData({});
      setIsWaitingForClean(false);
      setFinishError(null);
      closeRequestedRef.current = false;
    }

    securityFormContext.reset({
      hostname:
        !isHostnameFetching && systemHostname
          ? (systemHostname as string)
          : isValidSettings(settings)
            ? (settings.hostname ?? "")
            : "",
      workgroup: isValidSettings(settings)
        ? (settings.workgroup ?? "WORKGROUP")
        : "WORKGROUP",
      newPassword: "",
      confirmPassword: "",
    });

    networkFormContext.reset({
      bind_all_interfaces: isValidSettings(settings)
        ? (settings.bind_all_interfaces ?? true)
        : true,
      interfaces: isValidSettings(settings) ? (settings.interfaces ?? []) : [],
    });

    firstShareFormContext.reset({
      partitionId: "",
      shareName: "",
      usage: Usage.None,
    });

    telemetryFormContext.reset({
      telemetry_mode:
        isValidSettings(settings) && settings.telemetry_mode
          ? settings.telemetry_mode
          : Telemetry_mode.Errors,
    });
  }, [
    open,
    settings,
    systemHostname,
    isHostnameFetching,
    securityFormContext,
    networkFormContext,
    firstShareFormContext,
    telemetryFormContext,
    // wasOpenRef is a stable ref — intentionally omitted from deps
    // eslint-disable-next-line react-hooks/exhaustive-deps
  ]);

  useEffect(() => {
    if (!open) {
      return;
    }
    if (!hasAvailablePartitions) {
      firstShareFormContext.setValue("partitionId", "");
      firstShareFormContext.setValue("shareName", "");
      return;
    }

    if (!selectedPartitionId) {
      firstShareFormContext.setValue("shareName", "");
      return;
    }

    const selected = availablePartitions.find(
      (partition) => partition.partitionId === selectedPartitionId,
    );
    if (!selected) {
      return;
    }

    if (!currentShareName || currentShareName.trim() === "") {
      firstShareFormContext.setValue("shareName", selected.suggestedShareName, {
        shouldDirty: false,
      });
    }
  }, [
    open,
    availablePartitions,
    currentShareName,
    firstShareFormContext,
    hasAvailablePartitions,
    selectedPartitionId,
  ]);

  const handleSkip = () => {
    requestClose();
  };

  const handleStepComplete = <T extends keyof WizardCollectedData>(
    key: T,
    data: WizardCollectedData[T],
  ) => {
    setFinishError(null);
    setCollectedData((prev) => ({ ...prev, [key]: data }));
    setActiveStep((prev) => prev + 1);
  };

  useEffect(() => {
    if (!open || !isFinishing) {
      return;
    }

    const dirtyTrackingClean = isDirtyTrackingClean(health?.dirty_tracking);

    if (!isWaitingForClean) {
      if (!dirtyTrackingClean) {
        console.debug("Settings are being applied, waiting for clean state");
        setIsWaitingForClean(true);
        return;
      }

      console.debug("Dirty tracking already clean, closing wizard");
      setIsFinishing(false);
      requestClose();
      return;
    }

    if (dirtyTrackingClean) {
      console.debug("All dirty tracking flags are clean, closing wizard");
      setIsWaitingForClean(false);
      setIsFinishing(false);
      requestClose();
    }
  }, [
    health?.dirty_tracking,
    isWaitingForClean,
    open,
    isFinishing,
    requestClose,
  ]);

  const handleFinish = async () => {
    const allData = collectedData;
    closeRequestedRef.current = false;
    setIsWaitingForClean(false);
    setIsFinishing(true);
    setFinishError(null);

    const allCommitted: Promise<unknown | ErrorModel>[] = [];

    try {
      const updatedSettings: Settings = {
        ...(isValidSettings(settings) ? settings : {}),
        ...(allData.security?.hostname !== undefined && {
          hostname: allData.security.hostname,
        }),
        ...(allData.security?.workgroup !== undefined && {
          workgroup: allData.security.workgroup,
        }),
        ...(allData.network?.bind_all_interfaces !== undefined && {
          bind_all_interfaces: allData.network.bind_all_interfaces,
        }),
        ...(allData.network?.interfaces !== undefined && {
          interfaces: allData.network.interfaces,
        }),
        ...(allData.telemetry?.telemetry_mode !== undefined && {
          telemetry_mode: allData.telemetry.telemetry_mode,
        }),
      };
      allCommitted.push(updateSettings({ settings: updatedSettings }).unwrap());

      const newPassword = allData.security?.newPassword;
      if (newPassword && isValidUsers(users)) {
        const currentAdminUser = users.find((u) => u.is_admin);
        if (currentAdminUser) {
          allCommitted.push(
            updateAdminUser({
              user: { ...currentAdminUser, password: newPassword },
            }).unwrap(),
          );
        }
      }

      const sharePartitionId = allData.firstShare?.partitionId;
      if (sharePartitionId) {
        const selectedPartition = findPartitionById(
          volumes as Disk[] | undefined,
          sharePartitionId,
        );

        if (selectedPartition?.id) {
          const existingMountData = Object.values(
            selectedPartition.mount_point_data || {},
          )[0];
          const mountPath =
            existingMountData?.path ||
            `/mnt/${sanitizeWizardShareName(selectedPartition.name || selectedPartition.id)}`;

          allCommitted.push(
            mountVolume({
              mountPointData: {
                device_id: selectedPartition.id,
                path: mountPath,
                root: "/",
                type: Type.Addon,
                is_to_mount_at_startup: true,
              },
            })
              .unwrap()
              .then((resp) => {
                const shareName = allData.firstShare?.shareName?.trim();
                if (shareName && sharePartitionId) {
                  return createShare({
                    sharedResourcePostData: {
                      name: shareName,
                      usage: allData.firstShare?.usage || Usage.None,
                      mount_point_data: resp as MountPointData,
                    },
                  }).unwrap();
                }
                return resp;
              }),
          );
        }
      }

      await Promise.all(allCommitted)
        .then(() => {
          console.debug(
            "All settings applied successfully, waiting for clean state",
          );
          //setIsWaitingForClean(true);
          //          setTimeout(() => {
          //            if (isDirtyTrackingClean(health?.dirty_tracking)) {
          //              setIsWaitingForClean(false);
          //            }
          //         }, 3000);
        })
        .catch((error) => {
          console.error("Error applying settings in wizard:", error);
          setFinishError(
            "Failed to save some settings. You can configure them later in Settings.",
          );
          setIsFinishing(false);
        });
    } catch (error) {
      console.error("Error applying settings in wizard:", error);
      setFinishError(
        `Failed to save some settings. You can configure them later in Settings. Error: ${(error as Error).message}`,
      );
      setIsFinishing(false);
    }
  };

  const handleBack = () => setActiveStep((prev) => prev - 1);

  return (
    <Dialog
      open={open}
      aria-labelledby={WIZARD_TITLE_ID}
      aria-describedby={WIZARD_DESCRIPTION_ID}
      onClose={(_e, reason) => {
        if (
          allowSkip &&
          reason !== "backdropClick" &&
          reason !== "escapeKeyDown"
        ) {
          handleSkip();
        }
      }}
      maxWidth="sm"
      fullWidth
    >
      <DialogTitle id={WIZARD_TITLE_ID}>
        Setup Wizard
        <Typography
          id={WIZARD_DESCRIPTION_ID}
          variant="body2"
          color="text.secondary"
        >
          Configure your SRAT installation
        </Typography>
      </DialogTitle>
      <Stepper
        activeStep={activeStep}
        sx={{ px: 3, py: 2 }}
        aria-label="Setup wizard progress"
      >
        {STEP_LABELS.map((label) => (
          <Step key={label}>
            <StepLabel>{label}</StepLabel>
          </Step>
        ))}
      </Stepper>
      {finishError && (
        <Alert severity="error" sx={{ mx: 3, mb: 1 }}>
          {finishError}
        </Alert>
      )}

      {activeStep === 0 && (
        <FormContainer
          formContext={securityFormContext}
          onSuccess={(data) => handleStepComplete("security", data)}
        >
          <SecurityStepContent
            adminUser={adminUser}
            rootError={securityFormContext.formState.errors.root?.message}
          />
          <SetupWizardActions
            allowSkip={allowSkip}
            showBack={false}
            onSkip={handleSkip}
            submitLabel="Next"
            submitDisabled={securityFormContext.formState.isSubmitting}
          />
        </FormContainer>
      )}

      {activeStep === 1 && (
        <FormContainer
          formContext={networkFormContext}
          onSuccess={(data) => handleStepComplete("network", data)}
        >
          <NetworkStepContent
            bindAllInterfaces={bindAllInterfaces}
            nics={nics as InterfaceStat[] | undefined}
            isNicLoading={isNicLoading}
          />
          <SetupWizardActions
            allowSkip={allowSkip}
            showBack
            onBack={handleBack}
            onSkip={handleSkip}
            submitLabel="Next"
            submitDisabled={networkFormContext.formState.isSubmitting}
          />
        </FormContainer>
      )}

      {activeStep === 2 && (
        <FormContainer
          formContext={firstShareFormContext}
          onSuccess={(data) => handleStepComplete("firstShare", data)}
        >
          <FirstShareStepContent
            availablePartitions={availablePartitions}
            hasAvailablePartitions={hasAvailablePartitions}
            isVolumesLoading={isVolumesLoading}
            selectedPartitionId={selectedPartitionId}
          />
          <SetupWizardActions
            allowSkip={allowSkip}
            showBack
            onBack={handleBack}
            onSkip={handleSkip}
            submitLabel="Next"
            submitDisabled={firstShareFormContext.formState.isSubmitting}
          />
        </FormContainer>
      )}

      {activeStep === 3 && (
        <FormContainer
          formContext={telemetryFormContext}
          onSuccess={(data) => handleStepComplete("telemetry", data)}
        >
          <TelemetryStepContent
            internetConnection={normalizedInternetConnection}
            isCheckingConnection={isCheckingConnection}
            control={telemetryFormContext.control}
          />
          <SetupWizardActions
            allowSkip={allowSkip}
            showBack
            onBack={handleBack}
            onSkip={handleSkip}
            submitLabel="Next"
            submitDisabled={
              telemetryFormContext.formState.isSubmitting ||
              isCheckingConnection
            }
          />
        </FormContainer>
      )}

      {activeStep === 4 && (
        <>
          <SummaryStepContent
            data={collectedData}
            selectedPartition={selectedSummaryPartition}
            isProcessing={isFinishing}
            dirtyTracking={health?.dirty_tracking}
          />
          <SetupWizardActions
            allowSkip={allowSkip && !isFinishing && !isWaitingForClean}
            showBack={!isFinishing && !isWaitingForClean}
            onBack={handleBack}
            onSkip={handleSkip}
            submitType="button"
            onSubmit={handleFinish}
            submitLabel={
              isFinishing
                ? "Applying..."
                : isWaitingForClean
                  ? "Waiting..."
                  : "Finish"
            }
            submitDisabled={isFinishing || isWaitingForClean}
          />
        </>
      )}
    </Dialog>
  );
}
