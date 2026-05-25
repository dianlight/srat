import ScienceOutlinedIcon from "@mui/icons-material/ScienceOutlined";
import { Alert, DialogContent, Stack, Typography } from "@mui/material";
import type { Control } from "react-hook-form";
import { SettingSwitchRow } from "../../../pages/settings/components/SettingSwitchRow";
import type { LabModeFormData } from "../types";

interface LabModeStepContentProps {
  control: Control<LabModeFormData>;
}

export function LabModeStepContent({ control }: LabModeStepContentProps) {
  return (
    <DialogContent>
      <Stack spacing={2}>
        <Typography variant="body2" color="text.secondary">
          Experimental Lab Mode reveals advanced features that are still under
          active development, including the `smb.conf` view and selected Home
          Assistant integration tools.
        </Typography>

        <Alert severity="warning">
          Lab features are intended for troubleshooting and advanced testing.
          They may change, move, or behave differently between releases.
        </Alert>

        <SettingSwitchRow<LabModeFormData>
          ariaLabel="Experimental Lab Mode"
          control={control}
          helperText="Enable unstable lab features such as smb.conf, Use NFS for HA, and the SRAT custom component tools."
          label={
            <Stack direction="row" spacing={1} sx={{ alignItems: "center" }}>
              <Typography component="span">Experimental Lab Mode</Typography>
              <ScienceOutlinedIcon color="warning" fontSize="small" />
            </Stack>
          }
          name="experimental_lab_mode"
          tooltip={
            <>
              <Typography variant="h6" component="div">
                Experimental Lab Mode
              </Typography>
              <Typography variant="body2">
                Shows unstable and experimental SRAT capabilities that are not
                part of the regular production experience yet.
              </Typography>
              <Typography variant="body2" sx={{ mt: 1 }}>
                Leave this off unless you want access to advanced
                troubleshooting or preview tools.
              </Typography>
            </>
          }
        />
      </Stack>
    </DialogContent>
  );
}
