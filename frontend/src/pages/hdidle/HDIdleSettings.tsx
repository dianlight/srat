import {
	Box,
	Button,
	Card,
	CardContent,
	CircularProgress,
	Divider,
	Grid,
	IconButton,
	Stack,
	Tooltip,
	Typography,
} from "@mui/material";
import DeleteIcon from "@mui/icons-material/Delete";
import AddIcon from "@mui/icons-material/Add";
import {
	AutocompleteElement,
	CheckboxElement,
	SwitchElement,
	TextFieldElement,
	useFieldArray,
	useForm,
} from "react-hook-form-mui";
import { useEffect } from "react";

// Types for HDIdle configuration
interface HDIdleDeviceConfig {
	id?: number;
	name: string;
	idle_time: number;
	command_type: string;
	power_condition: number;
}

interface HDIdleConfig {
	id?: number;
	enabled: boolean;
	default_idle_time: number;
	default_command_type: string;
	default_power_condition: number;
	debug: boolean;
	log_file: string;
	symlink_policy: number;
	ignore_spin_down_detection: boolean;
	devices: HDIdleDeviceConfig[];
}

interface HDIdleStatusDisk {
	name: string;
	given_name: string;
	spun_down: boolean;
	last_io_at: string;
	spin_down_at: string;
	spin_up_at: string;
	idle_time: number;
	command_type: string;
	power_condition: number;
}

interface HDIdleStatus {
	running: boolean;
	monitored_at?: string;
	disks?: HDIdleStatusDisk[];
}

// Mock hooks for API calls - these would be generated from OpenAPI
const useGetHDIdleConfigQuery = () => {
	// TODO: Replace with actual RTK Query hook
	return {
		data: null as HDIdleConfig | null,
		isLoading: false,
		error: null,
		refetch: () => {},
	};
};

const usePutHDIdleConfigMutation = () => {
	// TODO: Replace with actual RTK Query hook
	return [
		async (config: HDIdleConfig) => {
			console.log("Saving HDIdle config:", config);
			// Make actual API call here
		},
		{ isLoading: false },
	] as const;
};

const useGetHDIdleStatusQuery = () => {
	// TODO: Replace with actual RTK Query hook
	return {
		data: null as HDIdleStatus | null,
		isLoading: false,
		error: null,
	};
};

export function HDIdleSettings() {
	const { data: config, isLoading, error, refetch } = useGetHDIdleConfigQuery();
	const { data: status, isLoading: statusLoading } = useGetHDIdleStatusQuery();
	const [putConfig, { isLoading: isSaving }] = usePutHDIdleConfigMutation();

	const { control, handleSubmit, reset, watch } = useForm<HDIdleConfig>({
		defaultValues: {
			enabled: false,
			default_idle_time: 600,
			default_command_type: "scsi",
			default_power_condition: 0,
			debug: false,
			log_file: "",
			symlink_policy: 0,
			ignore_spin_down_detection: false,
			devices: [],
		},
	});

	const { fields, append, remove } = useFieldArray({
		control,
		name: "devices",
	});

	const isEnabled = watch("enabled");

	useEffect(() => {
		if (config) {
			reset(config);
		}
	}, [config, reset]);

	const onSubmit = async (data: HDIdleConfig) => {
		try {
			await putConfig(data);
			refetch();
		} catch (err) {
			console.error("Failed to save HDIdle configuration:", err);
		}
	};

	const handleAddDevice = () => {
		append({
			name: "",
			idle_time: 0,
			command_type: "",
			power_condition: 0,
		});
	};

	if (isLoading) {
		return (
			<Box display="flex" justifyContent="center" p={4}>
				<CircularProgress />
			</Box>
		);
	}

	if (error) {
		return (
			<Box p={2}>
				<Typography color="error">
					Failed to load HDIdle configuration
				</Typography>
			</Box>
		);
	}

	return (
		<Box sx={{ p: 2 }}>
			<Typography variant="h4" gutterBottom>
				HDIdle Disk Spin-Down Configuration
			</Typography>
			<Typography variant="body2" color="text.secondary" paragraph>
				Configure automatic disk spin-down to save power when disks are idle.
				This helps reduce power consumption and extend disk lifespan.
			</Typography>

			<form onSubmit={handleSubmit(onSubmit)} noValidate>
				<Card sx={{ mb: 3 }}>
					<CardContent>
						<Typography variant="h6" gutterBottom>
							General Settings
						</Typography>
						<Grid container spacing={2}>
							<Grid size={{ xs: 12 }}>
								<SwitchElement
									name="enabled"
									label="Enable HDIdle Service"
									control={control}
									switchProps={{
										"aria-label": "Enable HDIdle",
									}}
								/>
								<Typography variant="caption" color="text.secondary">
									Enable automatic disk spin-down monitoring
								</Typography>
							</Grid>

							<Grid size={{ xs: 12, md: 4 }}>
								<TextFieldElement
									name="default_idle_time"
									label="Default Idle Time (seconds)"
									type="number"
									control={control}
									required
									disabled={!isEnabled}
									inputProps={{ min: 60 }}
									size="small"
								/>
								<Typography variant="caption" color="text.secondary">
									Time in seconds before spinning down idle disks (minimum: 60)
								</Typography>
							</Grid>

							<Grid size={{ xs: 12, md: 4 }}>
								<AutocompleteElement
									name="default_command_type"
									label="Default Command Type"
									control={control}
									options={["scsi", "ata"]}
									autocompleteProps={{
										size: "small",
										disabled: !isEnabled,
										disableClearable: true,
									}}
								/>
								<Typography variant="caption" color="text.secondary">
									Type of command to use for spinning down disks
								</Typography>
							</Grid>

							<Grid size={{ xs: 12, md: 4 }}>
								<TextFieldElement
									name="default_power_condition"
									label="Default Power Condition"
									type="number"
									control={control}
									disabled={!isEnabled}
									inputProps={{ min: 0, max: 15 }}
									size="small"
								/>
								<Typography variant="caption" color="text.secondary">
									SCSI power condition (0-15)
								</Typography>
							</Grid>

							<Grid size={{ xs: 12, md: 6 }}>
								<AutocompleteElement
									name="symlink_policy"
									label="Symlink Resolution Policy"
									control={control}
									options={[
										{ label: "Resolve Once", value: 0 },
										{ label: "Retry Resolution", value: 1 },
									]}
									autocompleteProps={{
										size: "small",
										disabled: !isEnabled,
										disableClearable: true,
										getOptionLabel: (option: any) => option.label || String(option),
									}}
								/>
							</Grid>

							<Grid size={{ xs: 12, md: 6 }}>
								<TextFieldElement
									name="log_file"
									label="Log File Path (optional)"
									control={control}
									disabled={!isEnabled}
									size="small"
									placeholder="/var/log/hdidle.log"
								/>
							</Grid>

							<Grid size={{ xs: 12, md: 6 }}>
								<CheckboxElement
									name="debug"
									label="Enable Debug Logging"
									control={control}
									disabled={!isEnabled}
								/>
							</Grid>

							<Grid size={{ xs: 12, md: 6 }}>
								<CheckboxElement
									name="ignore_spin_down_detection"
									label="Ignore Spin Down Detection"
									control={control}
									disabled={!isEnabled}
								/>
							</Grid>
						</Grid>
					</CardContent>
				</Card>

				<Card sx={{ mb: 3 }}>
					<CardContent>
						<Stack
							direction="row"
							justifyContent="space-between"
							alignItems="center"
							mb={2}
						>
							<Typography variant="h6">Per-Device Configuration</Typography>
							<Button
								startIcon={<AddIcon />}
								onClick={handleAddDevice}
								disabled={!isEnabled}
								variant="outlined"
								size="small"
							>
								Add Device
							</Button>
						</Stack>

						{fields.length === 0 ? (
							<Typography variant="body2" color="text.secondary">
								No device-specific configurations. Default settings will apply to
								all disks.
							</Typography>
						) : (
							<Grid container spacing={2}>
								{fields.map((field, index) => (
									<Grid size={{ xs: 12 }} key={field.id}>
										<Card variant="outlined">
											<CardContent>
												<Stack
													direction="row"
													justifyContent="space-between"
													alignItems="center"
													mb={2}
												>
													<Typography variant="subtitle2">
														Device {index + 1}
													</Typography>
													<IconButton
														size="small"
														onClick={() => remove(index)}
														disabled={!isEnabled}
													>
														<DeleteIcon />
													</IconButton>
												</Stack>
												<Grid container spacing={2}>
													<Grid size={{ xs: 12, md: 3 }}>
														<TextFieldElement
															name={`devices.${index}.name`}
															label="Device Name"
															control={control}
															required
															disabled={!isEnabled}
															placeholder="sda"
															size="small"
														/>
													</Grid>
													<Grid size={{ xs: 12, md: 3 }}>
														<TextFieldElement
															name={`devices.${index}.idle_time`}
															label="Idle Time (seconds)"
															type="number"
															control={control}
															disabled={!isEnabled}
															inputProps={{ min: 0 }}
															size="small"
															helperText="0 = use default"
														/>
													</Grid>
													<Grid size={{ xs: 12, md: 3 }}>
														<AutocompleteElement
															name={`devices.${index}.command_type`}
															label="Command Type"
															control={control}
															options={["", "scsi", "ata"]}
															autocompleteProps={{
																size: "small",
																disabled: !isEnabled,
															}}
														/>
													</Grid>
													<Grid size={{ xs: 12, md: 3 }}>
														<TextFieldElement
															name={`devices.${index}.power_condition`}
															label="Power Condition"
															type="number"
															control={control}
															disabled={!isEnabled}
															inputProps={{ min: 0, max: 15 }}
															size="small"
														/>
													</Grid>
												</Grid>
											</CardContent>
										</Card>
									</Grid>
								))}
							</Grid>
						)}
					</CardContent>
				</Card>

				{status && status.running && (
					<Card sx={{ mb: 3 }}>
						<CardContent>
							<Typography variant="h6" gutterBottom>
								Service Status
							</Typography>
							<Typography variant="body2" color="success.main" gutterBottom>
								✓ Service is running
							</Typography>
							{status.monitored_at && (
								<Typography variant="caption" color="text.secondary">
									Last monitored: {new Date(status.monitored_at).toLocaleString()}
								</Typography>
							)}

							{status.disks && status.disks.length > 0 && (
								<Box mt={2}>
									<Typography variant="subtitle2" gutterBottom>
										Monitored Disks:
									</Typography>
									{status.disks.map((disk) => (
										<Box
											key={disk.name}
											sx={{
												p: 1,
												mb: 1,
												border: "1px solid",
												borderColor: "divider",
												borderRadius: 1,
											}}
										>
											<Typography variant="body2">
												<strong>{disk.given_name || disk.name}</strong>
												{disk.spun_down ? " (Spun Down)" : " (Active)"}
											</Typography>
											<Typography variant="caption" color="text.secondary">
												Last I/O: {new Date(disk.last_io_at).toLocaleString()}
											</Typography>
										</Box>
									))}
								</Box>
							)}
						</CardContent>
					</Card>
				)}

				<Stack direction="row" spacing={2}>
					<Button
						type="submit"
						variant="contained"
						disabled={isSaving || !isEnabled}
					>
						{isSaving ? <CircularProgress size={24} /> : "Save Configuration"}
					</Button>
					<Button
						variant="outlined"
						onClick={() => reset()}
						disabled={isSaving}
					>
						Reset
					</Button>
				</Stack>
			</form>
		</Box>
	);
}
