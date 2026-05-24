import {
  Box,
  Stack,
  type SxProps,
  type Theme,
  Tooltip,
  Typography,
} from "@mui/material";
import type { ComponentProps, ReactNode } from "react";
import type { Control, FieldValues, Path } from "react-hook-form";
import { SwitchElement } from "react-hook-form-mui";

type SwitchElementProps = ComponentProps<typeof SwitchElement>;

type SettingSwitchRowProps<TFieldValues extends FieldValues = FieldValues> = {
  name: Path<TFieldValues>;
  label: ReactNode;
  ariaLabel: string;
  control?: Control<TFieldValues>;
  disabled?: boolean;
  id?: string;
  helperText?: ReactNode;
  tooltip?: ReactNode;
  sx?: SxProps<Theme>;
  switchElementProps?: Omit<
    SwitchElementProps,
    "name" | "label" | "labelPlacement" | "switchProps" | "control"
  >;
};

export function SettingSwitchRow<
  TFieldValues extends FieldValues = FieldValues,
>({
  name,
  label,
  ariaLabel,
  control,
  disabled,
  id,
  helperText,
  tooltip,
  sx,
  switchElementProps,
}: SettingSwitchRowProps<TFieldValues>) {
  const normalizedSwitchProps = {
    "aria-label": ariaLabel,
    size: "small" as const,
  };

  const switchControl = (
    <Box sx={{ display: "inline-block", width: "100%" }}>
      <SwitchElement
        {...switchElementProps}
        control={control}
        disabled={disabled}
        id={id}
        label={label}
        labelPlacement="start"
        name={name}
        switchProps={normalizedSwitchProps}
        sx={{ display: "flex", ...(sx as object) }}
      />
    </Box>
  );

  return (
    <Stack spacing={0.5}>
      {tooltip ? (
        <Tooltip title={tooltip}>{switchControl}</Tooltip>
      ) : (
        switchControl
      )}
      {helperText ? (
        <Typography
          variant="caption"
          sx={{
            color: "text.secondary",
          }}
        >
          {helperText}
        </Typography>
      ) : null}
    </Stack>
  );
}
