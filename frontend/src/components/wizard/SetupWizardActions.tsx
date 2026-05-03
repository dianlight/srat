import { Box, Button, DialogActions } from "@mui/material";

interface SetupWizardActionsProps {
  allowSkip: boolean;
  showBack: boolean;
  onBack?: () => void;
  onSkip: () => void;
  submitLabel: string;
  submitDisabled?: boolean;
  submitType?: "submit" | "button";
  onSubmit?: () => void;
}

export function SetupWizardActions({
  allowSkip,
  showBack,
  onBack,
  onSkip,
  submitLabel,
  submitDisabled = false,
  submitType = "submit",
  onSubmit,
}: SetupWizardActionsProps) {
  return (
    <DialogActions sx={{ px: 3, pb: 2 }}>
      {allowSkip && (
        <Button type="button" onClick={onSkip} color="inherit">
          Skip Setup
        </Button>
      )}
      <Box sx={{ flex: 1 }} />
      {showBack && (
        <Button type="button" onClick={onBack}>
          Back
        </Button>
      )}
      <Button
        type={submitType}
        variant="contained"
        disabled={submitDisabled}
        onClick={submitType === "button" ? onSubmit : undefined}
        loading={submitType === "button" && submitDisabled}
      >
        {submitLabel}
      </Button>
    </DialogActions>
  );
}
