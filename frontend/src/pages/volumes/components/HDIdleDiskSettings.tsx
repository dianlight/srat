import {
	Card,
	CardContent,
	CardHeader,
	Collapse,
	Grid,
	Typography,
	Tooltip,
	Box,
	IconButton,
	ToggleButton,
	ToggleButtonGroup,
} from "@mui/material";
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import PowerIcon from "@mui/icons-material/Power";
import {
	AutocompleteElement,
	TextFieldElement,
	useForm,
	type Control,
} from "react-hook-form-mui";
import { useEffect, useMemo, useState } from "react";
import { Controller, useWatch } from "react-hook-form";
import type { Disk, Settings, HdIdleDeviceDto } from "../../../store/sratApi";
import {
	useGetApiSettingsQuery,
	useGetApiHdidleEffectiveConfigQuery,
	useGetApiDiskByDiskIdHdidleConfigQuery,
	useGetApiDiskByDiskIdHdidleSupportQuery,
	usePutApiDiskByDiskIdHdidleConfigMutation,
	Enabled,
} from "../../../store/sratApi";
import { getCurrentEnv } from "../../../macro/Environment" with {
	type: "macro",
};

interface HDIdleDiskSettingsProps {
	disk: Disk;
	readOnly?: boolean;
}

export function HDIdleDiskSettings({ disk, readOnly = false }: HDIdleDiskSettingsProps) {
	const { control, reset, formState, getValues } = useForm({
		defaultValues: {
			enabled: Enabled.Yes,
			...disk?.hdidle_status,
		},
	});
	const { data: settings, isLoading: isLoadingSettings } = useGetApiSettingsQuery();
	const { data: effectiveConfig } = useGetApiHdidleEffectiveConfigQuery();
	// disk_id must always be the stable disk ID (not a device path)
	const diskId = (disk as any)?.id || (disk as any)?.name || "";
	const { data: deviceConfig, isFetching: isFetchingDeviceConfig } = useGetApiDiskByDiskIdHdidleConfigQuery({ diskId }, { skip: !diskId });
	const { data: supportInfo, isFetching: isFetchingSupport } = useGetApiDiskByDiskIdHdidleSupportQuery({ diskId }, { skip: !diskId });
	const [saveConfig, { isLoading: isSaving }] = usePutApiDiskByDiskIdHdidleConfigMutation();
	const isTestEnv = (globalThis as any).__TEST__ === true;
	const [expanded, setExpanded] = useState(false);
	const [visible, setVisible] = useState(false);
	const diskName = (disk as any).model || (disk as any).id || "Unknown";

	// Watch the local enabled toggle to disable/enable the rest of the form
	const enabled = useWatch({ control, name: "enabled" }) as Enabled | undefined;
	const fieldsDisabled = enabled === Enabled.No || readOnly;
	const unsupported = (supportInfo as any)?.Supported === false;

	useEffect(() => {
		// Visibility rules: show when hdidle globally enabled or in tests/non-prod
		const globallyEnabled = (settings as Settings)?.hdidle_enabled || ((effectiveConfig as any)?.enabled ?? false);
		if (isTestEnv || globallyEnabled) {
			setVisible(true);
		} else {
			setVisible(getCurrentEnv() !== "production");
		}

		// When disk prop or API config changes, update form values
		const apiValues = (deviceConfig as HdIdleDeviceDto | undefined) ?? undefined;
		reset({
			enabled: (apiValues?.enabled as Enabled | undefined) ?? Enabled.Yes,
			idle_time: apiValues?.idle_time ?? (disk as any)?.hdidle_status?.idle_time ?? 0,
			command_type: apiValues?.command_type ?? (disk as any)?.hdidle_status?.command_type ?? "",
			power_condition: apiValues?.power_condition ?? (disk as any)?.hdidle_status?.power_condition ?? 0,
		});
	}, [disk, reset, settings, effectiveConfig, isTestEnv, deviceConfig]);

	// Close accordion if enabled is not Custom
	useEffect(() => {
		if (enabled !== Enabled.Custom) {
			setExpanded(false);
		}
	}, [enabled]);

	// Read HDIdle config snapshot from disk dto when available
	const hdidleStatus = useMemo(() => {
		const s = (disk as any)?.hdidle_status as
			| { idle_time?: number; command_type?: string; power_condition?: number; enabled?: Enabled }
			| undefined;
		return s;
	}, [disk]);

	const handleExpandChange = () => {
		setExpanded(!expanded);
	};

	const handleApply = async () => {
		if (!diskId) return;
		const values = getValues();
		const payload: HdIdleDeviceDto = {
			// The backend expects the full by-id device path in the payload
			device_path: `/dev/disk/by-id/${diskId}`,
			enabled: values.enabled as Enabled,
			idle_time: Number(values.idle_time ?? 0),
			command_type: (values.command_type as any) || undefined,
			power_condition: Number(values.power_condition ?? 0),
		};
		try {
			await saveConfig({ diskId, hdIdleDeviceDto: payload }).unwrap();
		} catch (e) {
			// No-op; errors should be surfaced by global error UI
		}
	};

	const handleCancel = () => {
		// Restore last loaded API values
		const apiValues = (deviceConfig as HdIdleDeviceDto | undefined) ?? undefined;
		reset({
			enabled: (apiValues?.enabled as Enabled | undefined) ?? Enabled.Yes,
			idle_time: apiValues?.idle_time ?? (disk as any)?.hdidle_status?.idle_time ?? 0,
			command_type: apiValues?.command_type ?? (disk as any)?.hdidle_status?.command_type ?? "",
			power_condition: apiValues?.power_condition ?? (disk as any)?.hdidle_status?.power_condition ?? 0,
		});
	};

	return visible && !isLoadingSettings && (
		<Card>
			<CardHeader
				title="Power Settings ( 🚧 Work In Progress )"
				avatar={
					<IconButton size="small" sx={{ pointerEvents: 'none' }}>
						<PowerIcon color="primary" />
					</IconButton>
				}
				action={
					<Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
						<Tooltip
							title={
								<Typography variant="body2">
									{unsupported
										? "Device not supported by hdidle. Override is disabled."
										: "Enable disk-specific override. When Off, fields are read-only."}
								</Typography>
							}
						>
							<span>
								<Controller
									name="enabled"
									control={control}
									render={({ field: { value, onChange } }) => (
										<ToggleButtonGroup
											value={value}
											exclusive
											size="small"
											color={value === Enabled.Yes ? "success" : "standard"}
											onChange={(_, newValue) => {
												if (newValue === null) return;
												if (unsupported) return; // prevent toggle when unsupported
												onChange(newValue as Enabled);
											}}
											aria-label="toggle disk override"
										>
											<ToggleButton value={Enabled.Yes}>{Enabled.Yes}</ToggleButton>
											<ToggleButton value={Enabled.Custom}>{Enabled.Custom}</ToggleButton>
											<ToggleButton value={Enabled.No}>{Enabled.No}</ToggleButton>
										</ToggleButtonGroup>
									)}
								/>
							</span>
						</Tooltip>



						<IconButton
							onClick={handleExpandChange}
							disabled={enabled !== Enabled.Custom || unsupported}
							aria-expanded={expanded}
							aria-label="show more"
							sx={{
								transform: expanded ? "rotate(180deg)" : "rotate(0deg)",
								transition: "transform 150ms cubic-bezier(0.4, 0, 0.2, 1)",
							}}
						>
							<ExpandMoreIcon />
						</IconButton>
					</Box>
				}
			/>
			<Collapse in={expanded} timeout="auto" unmountOnExit>
				<CardContent>
					<Grid container spacing={2}>
						<Grid size={12}>
							<Typography variant="body2" color="text.secondary" gutterBottom>
								Configure specific spin-down settings for <strong>{(disk as any).model || diskName}</strong>.
								Leave fields at 0 or empty to use default settings.
							</Typography>

							{hdidleStatus && (
								<Box sx={{ mt: 1, mb: 1, p: 1, backgroundColor: "info.lighter", borderRadius: 1 }}>
									<Typography variant="caption" color="text.secondary">
										Current config: idle time
										<strong> {hdidleStatus.idle_time ?? 0}s</strong>, command
										<strong> {hdidleStatus.command_type || "default"}</strong>, power condition
										<strong> {hdidleStatus.power_condition ?? 0}</strong>
										{hdidleStatus.enabled && (
											<span> — enabled: <strong>{hdidleStatus.enabled}</strong></span>
										)}
									</Typography>
								</Box>
							)}
						</Grid>

						<Grid size={{ xs: 12, md: 4 }}>
							<Tooltip
								title={
									<Typography variant="body2">
										Idle time before spinning down this specific disk. Set to 0 to use the default timeout.
									</Typography>
								}
							>
								<span style={{ display: "inline-block", width: "100%" }}>
									<TextFieldElement
										name={`idle_time`}
										label="Idle Time (seconds)"
										type="number"
										control={control}
										disabled={fieldsDisabled || unsupported}
										inputProps={{ min: 0 }}
										size="small"
										helperText="0 = use default"
									/>
								</span>
							</Tooltip>
						</Grid>

						<Grid size={{ xs: 12, md: 4 }}>
							<Tooltip
								title={
									<>
										<Typography variant="body2">
											Command type for this disk. Leave empty to use default.
										</Typography>
										<Typography variant="body2" sx={{ mt: 1 }}>
											<strong>SCSI:</strong> For most modern SATA/SAS drives
										</Typography>
										<Typography variant="body2">
											<strong>ATA:</strong> For legacy ATA/IDE drives
										</Typography>
									</>
								}
							>
								<span style={{ display: "inline-block", width: "100%" }}>
									<AutocompleteElement
										name={`command_type`}
										label="Command Type"
										control={control}
										options={["", "scsi", "ata"]}
										autocompleteProps={{
											size: "small",
											disabled: fieldsDisabled || unsupported,
										}}
										textFieldProps={{
											helperText: "Empty = use default",
										}}
									/>
								</span>
							</Tooltip>
						</Grid>

						<Grid size={{ xs: 12, md: 4 }}>
							<Tooltip
								title={
									<Typography variant="body2">
										SCSI power condition for this disk (0-15). Set to 0 for default behavior.
									</Typography>
								}
							>
								<span style={{ display: "inline-block", width: "100%" }}>
									<TextFieldElement
										name={`power_condition`}
										label="Power Condition"
										type="number"
										control={control}
										disabled={fieldsDisabled || unsupported}
										inputProps={{ min: 0, max: 15 }}
										size="small"
										helperText="0 = default"
									/>
								</span>
							</Tooltip>
						</Grid>

						<Grid size={12}>
							<Box
								sx={{
									mt: 1,
									p: 1,
									backgroundColor: "info.lighter",
									borderRadius: 1,
								}}
							>
								<Typography variant="caption" color="text.secondary">
									<strong>Note:</strong> Device-specific settings override global defaults.
									Changes take effect after the next service restart or configuration update.
								</Typography>
							</Box>
						</Grid>

						<Grid size={12}>
							<Box sx={{ display: "flex", gap: 1, justifyContent: "flex-end", mt: 2 }}>
								<Tooltip title={unsupported ? "Device unsupported" : formState.isDirty ? "Apply changes" : "No changes to apply"}>
									<span>
										<ToggleButton
											value="apply"
											disabled={unsupported || fieldsDisabled || !formState.isDirty || isSaving || isFetchingDeviceConfig || isFetchingSupport}
											onClick={handleApply}
											color={"success" as any}
											size="small"
										>
											Apply
										</ToggleButton>
									</span>
								</Tooltip>
								<Tooltip title="Restore last loaded values">
									<span>
										<ToggleButton
											value="cancel"
											disabled={isFetchingDeviceConfig || isSaving}
											onClick={handleCancel}
											size="small"
										>
											Cancel
										</ToggleButton>
									</span>
								</Tooltip>
							</Box>
						</Grid>
					</Grid>
				</CardContent>
			</Collapse>
		</Card>
	);
}
