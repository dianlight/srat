import {
	Accordion,
	AccordionSummary,
	AccordionDetails,
	Grid,
	Typography,
	Tooltip,
	Box,
} from "@mui/material";
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import {
	AutocompleteElement,
	TextFieldElement,
	type Control,
} from "react-hook-form-mui";
import type { Disk } from "../../store/sratApi";

interface HDIdleDiskSettingsProps {
	disk: Disk;
	control: Control<any>;
	readOnly?: boolean;
}

export function HDIdleDiskSettings({ disk, control, readOnly = false }: HDIdleDiskSettingsProps) {
	const diskName = disk.name || disk.id || "Unknown";
	const fieldPrefix = `hdidle_disk_${diskName}`;

	return (
		<Accordion defaultExpanded={false}>
			<AccordionSummary
				expandIcon={<ExpandMoreIcon />}
				aria-controls={`${diskName}-hdidle-content`}
				id={`${diskName}-hdidle-header`}
			>
				<Typography variant="subtitle1">
					HDIdle Disk Spin-Down Settings
				</Typography>
			</AccordionSummary>
			<AccordionDetails>
				<Grid container spacing={2}>
					<Grid size={12}>
						<Typography variant="body2" color="text.secondary" gutterBottom>
							Configure specific spin-down settings for <strong>{disk.model || diskName}</strong>.
							Leave fields at 0 or empty to use default settings.
						</Typography>
					</Grid>

					<Grid size={{ xs: 12, md: 4 }}>
						<Tooltip
							title={
								<Typography variant="body2">
									Idle time before spinning down this specific disk. Set to 0 to use the default timeout.
								</Typography>
							}
						>
							<TextFieldElement
								name={`${fieldPrefix}_idle_time`}
								label="Idle Time (seconds)"
								type="number"
								control={control}
								disabled={readOnly}
								inputProps={{ min: 0 }}
								size="small"
								helperText="0 = use default"
							/>
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
							<AutocompleteElement
								name={`${fieldPrefix}_command_type`}
								label="Command Type"
								control={control}
								options={["", "scsi", "ata"]}
								autocompleteProps={{
									size: "small",
									disabled: readOnly,
								}}
								textFieldProps={{
									helperText: "Empty = use default",
								}}
							/>
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
							<TextFieldElement
								name={`${fieldPrefix}_power_condition`}
								label="Power Condition"
								type="number"
								control={control}
								disabled={readOnly}
								inputProps={{ min: 0, max: 15 }}
								size="small"
								helperText="0 = default"
							/>
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
				</Grid>
			</AccordionDetails>
		</Accordion>
	);
}
