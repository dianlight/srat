import { DialogContent, Stack, Typography } from "@mui/material";
import { AutocompleteElement, CheckboxElement } from "react-hook-form-mui";
import type { InterfaceStat } from "../../../store/sratApi";

interface NetworkStepContentProps {
  bindAllInterfaces: boolean;
  nics?: InterfaceStat[];
  isNicLoading: boolean;
}

export function NetworkStepContent({
  bindAllInterfaces,
  nics,
  isNicLoading,
}: NetworkStepContentProps) {
  return (
    <DialogContent>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Choose which network interfaces Samba should listen on.
      </Typography>
      <Stack spacing={2}>
        <CheckboxElement
          name="bind_all_interfaces"
          label="Bind to all interfaces"
        />
        <AutocompleteElement
          multiple
          label="Interfaces"
          name="interfaces"
          options={
            nics
              ?.map((nc) => nc.name)
              .filter((name) => name !== "lo" && name !== "hassio") ?? []
          }
          loading={isNicLoading}
          autocompleteProps={{
            size: "small",
            disabled: bindAllInterfaces,
          }}
        />
      </Stack>
    </DialogContent>
  );
}
