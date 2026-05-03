import {
  Alert,
  DialogContent,
  Divider,
  Stack,
  Typography,
} from "@mui/material";
import { PasswordElement, TextFieldElement } from "react-hook-form-mui";
import type { User } from "../../../store/sratApi";

interface SecurityStepContentProps {
  adminUser?: User;
  rootError?: string;
}

export function SecurityStepContent({
  adminUser,
  rootError,
}: SecurityStepContentProps) {
  return (
    <DialogContent>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Set your administrator password and basic network identity for this
        Samba server.
      </Typography>
      {!adminUser?.password && (
        <Alert severity="warning" sx={{ mb: 2 }}>
          Your system is using the default password <strong>changeme!</strong>.
          Change it now.
        </Alert>
      )}
      {rootError && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {rootError}
        </Alert>
      )}
      <Stack spacing={2}>
        <TextFieldElement
          name="hostname"
          label="Hostname"
          fullWidth
          placeholder="e.g., samba-nas"
          helperText="Name of your server on the network"
          rules={{
            required: "Hostname is required",
            minLength: { value: 2, message: "At least 2 characters" },
          }}
        />
        <TextFieldElement
          name="workgroup"
          label="Workgroup"
          fullWidth
          placeholder="e.g., WORKGROUP"
          rules={{
            required: "Workgroup is required",
            minLength: { value: 2, message: "At least 2 characters" },
          }}
        />
        <Divider />
        <Typography variant="subtitle2">Administrator Password</Typography>
        <PasswordElement
          name="newPassword"
          label="New Password"
          fullWidth
          autoComplete="new-password"
          helperText="Leave blank to keep current password"
          rules={{
            validate: (value) => {
              if (!adminUser?.password && !value) return "Password is required";
              if (!value) return true;
              if (value === "changeme!")
                return "Cannot use the default password";
              if (value.length < 6) return "At least 6 characters";
              return true;
            },
          }}
        />
        <PasswordElement
          name="confirmPassword"
          label="Confirm Password"
          fullWidth
          autoComplete="new-password"
          rules={{
            validate: (value, formValues) =>
              !formValues.newPassword ||
              value === formValues.newPassword ||
              "Passwords do not match",
          }}
        />
      </Stack>
    </DialogContent>
  );
}
