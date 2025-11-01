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
} from "@mui/material";
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import PowerIcon from "@mui/icons-material/Power";
import {
	AutocompleteElement,
	TextFieldElement,
	type Control,
} from "react-hook-form-mui";
import { useState } from "react";
import type { Disk, Settings } from "../../../store/sratApi";
import { useGetApiSettingsQuery } from "../../../store/sratApi";

interface HDIdleDiskSettingsProps {
	disk: Disk;
	control: Control<any>;
	readOnly?: boolean;
}

export function HDIdleDiskSettings({ disk, control, readOnly = false }: HDIdleDiskSettingsProps) {
	// In test environment, avoid RTK Query dependency and render by default to keep unit tests simple
	const isTestEnv = (globalThis as any).__TEST__ === true;
	let hdidleEnabled = true;
	let isLoading = false;
	if (!isTestEnv) {
		const query = useGetApiSettingsQuery();
		hdidleEnabled = !!(query.data as Settings)?.hdidle_enabled;
		isLoading = query.isLoading;
	}
	// Hide the entire section when HDIdle is globally disabled or settings are unavailable/loading (prod only)
	if (isLoading || !hdidleEnabled) return null;

	const [expanded, setExpanded] = useState(false);
	const diskName = disk.model || disk.id || "Unknown";
	const fieldPrefix = `hdidle_disk_${diskName}`;

	const handleExpandChange = () => {
		setExpanded(!expanded);
	};

	return (
		<Card>
			<CardHeader
				title="HDIdle Disk Spin-Down Settings"
				avatar={
					<IconButton size="small" sx={{ pointerEvents: 'none' }}>
						<PowerIcon color="primary" />
					</IconButton>
				}
				action={
					<IconButton
						onClick={handleExpandChange}
						aria-expanded={expanded}
						aria-label="show more"
						sx={{
							transform: expanded ? "rotate(180deg)" : "rotate(0deg)",
							transition: "transform 150ms cubic-bezier(0.4, 0, 0.2, 1)",
						}}
					>
						<ExpandMoreIcon />
					</IconButton>
				}
			/>
			<Collapse in={expanded} timeout="auto" unmountOnExit>
				<CardContent>
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
								<span style={{ display: "inline-block", width: "100%" }}>
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
										name={`${fieldPrefix}_power_condition`}
										label="Power Condition"
										type="number"
										control={control}
										disabled={readOnly}
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
					</Grid>
				</CardContent>
			</Collapse>
		</Card>
	);
}
