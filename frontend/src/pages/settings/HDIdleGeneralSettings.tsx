import {
	Grid,
	Typography,
	Tooltip,
	Divider,
} from "@mui/material";
import {
	AutocompleteElement,
	CheckboxElement,
	SwitchElement,
	TextFieldElement,
	type Control,
} from "react-hook-form-mui";

interface HDIdleGeneralSettingsProps {
	control: Control<any>;
	isEnabled: boolean;
	readOnly?: boolean;
}

export function HDIdleGeneralSettings({ control, isEnabled, readOnly = false }: HDIdleGeneralSettingsProps) {
	return (
		<>
			<Grid size={12}>
				<Divider sx={{ my: 2 }} />
				<Typography variant="h6" gutterBottom>
					HDIdle Disk Spin-Down Settings
				</Typography>
				<Typography variant="body2" color="text.secondary" gutterBottom>
					Configure automatic disk spin-down to save power when disks are idle.
				</Typography>
			</Grid>

			<Grid size={{ xs: 12 }}>
				<Tooltip
					title={
						<>
							<Typography variant="h6" component="div">
								Enable HDIdle Service
							</Typography>
							<Typography variant="body2">
								Automatically spin down idle disks after a configured timeout to reduce
								power consumption and extend disk lifespan.
							</Typography>
						</>
					}
				>
					<SwitchElement
						name="hdidle_enabled"
						label="Enable Automatic Disk Spin-Down"
						control={control}
						disabled={readOnly}
						switchProps={{
							"aria-label": "Enable HDIdle",
							size: "small",
						}}
						labelPlacement="start"
					/>
				</Tooltip>
			</Grid>

			<Grid size={{ xs: 12, md: 4 }}>
				<TextFieldElement
					name="hdidle_default_idle_time"
					label="Default Idle Time (seconds)"
					type="number"
					control={control}
					required
					disabled={!isEnabled || readOnly}
					inputProps={{ min: 60 }}
					size="small"
				/>
				<Typography variant="caption" color="text.secondary">
					Time before spinning down idle disks (minimum: 60 seconds)
				</Typography>
			</Grid>

			<Grid size={{ xs: 12, md: 4 }}>
				<Tooltip
					title={
						<>
							<Typography variant="body2">
								<strong>SCSI:</strong> For most modern SATA/SAS drives
							</Typography>
							<Typography variant="body2">
								<strong>ATA:</strong> For legacy ATA/IDE drives
							</Typography>
						</>
					}
				>
					<AutocompleteElement
						name="hdidle_default_command_type"
						label="Default Command Type"
						control={control}
						options={["scsi", "ata"]}
						autocompleteProps={{
							size: "small",
							disabled: !isEnabled || readOnly,
							disableClearable: true,
						}}
					/>
				</Tooltip>
			</Grid>

			<Grid size={{ xs: 12, md: 4 }}>
				<TextFieldElement
					name="hdidle_log_file"
					label="Log File Path (optional)"
					control={control}
					disabled={!isEnabled || readOnly}
					size="small"
					placeholder="/var/log/hdidle.log"
				/>
			</Grid>

			<Grid size={{ xs: 12, md: 6 }}>
				<Tooltip
					title={
						<Typography variant="body2">
							Enable detailed logging for troubleshooting disk spin-down issues
						</Typography>
					}
				>
					<CheckboxElement
						name="hdidle_debug"
						label="Enable Debug Logging"
						control={control}
						disabled={!isEnabled || readOnly}
						size="small"
					/>
				</Tooltip>
			</Grid>

			<Grid size={{ xs: 12, md: 6 }}>
				<Tooltip
					title={
						<Typography variant="body2">
							Force spin down even if disk reports it's already spun down
						</Typography>
					}
				>
					<CheckboxElement
						name="hdidle_ignore_spin_down_detection"
						label="Ignore Spin Down Detection"
						control={control}
						disabled={!isEnabled || readOnly}
						size="small"
					/>
				</Tooltip>
			</Grid>
		</>
	);
}
