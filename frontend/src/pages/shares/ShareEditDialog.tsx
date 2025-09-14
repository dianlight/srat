import ModeEditIcon from "@mui/icons-material/ModeEdit";
import PlaylistAddIcon from "@mui/icons-material/PlaylistAdd";
import {
	Box,
	Button,
	Chip,
	Dialog,
	DialogActions,
	DialogContent,
	DialogContentText,
	DialogTitle,
	Grid,
	IconButton,
	InputAdornment,
	Stack,
	Tooltip,
	Typography,
} from "@mui/material";
import { MuiChipsInput } from "mui-chips-input";
import { Fragment, useEffect, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import {
	AutocompleteElement,
	CheckboxElement,
	SelectElement,
	SwitchElement,
	TextFieldElement,
} from "react-hook-form-mui";
import { useVolume } from "../../hooks/volumeHook";
import default_json from "../../json/default_config.json";
import {
	type MountPointData,
	type SharedResource,
	type User,
	Time_machine_support,
} from "../../store/sratApi";
import {
	Usage,
	useGetApiUsersQuery,
} from "../../store/sratApi";
import type { ShareEditProps } from "./types";
import {
	casingCycleOrder,
	getCasingIcon,
	getPathBaseName,
	isValidVetoFileEntry,
	sanitizeAndUppercaseShareName,
	toCamelCase,
	toKebabCase,
} from "./utils";
import { color } from "bun";
import { filesize } from "filesize";

interface ShareEditDialogProps {
	open: boolean;
	onClose: (data?: ShareEditProps) => void;
	objectToEdit?: ShareEditProps;
	shares?: SharedResource[]; // Added to receive shares data
	onDeleteSubmit?: (shareName: string, shareData: SharedResource) => void; // Added for delete action
	onToggleEnabled?: (enabled: boolean) => void; // Added for enable/disable toggle
}

export function ShareEditDialog(props: ShareEditDialogProps) {
	const {
		data: users,
		isLoading: usLoading,
		error: usError,
	} = useGetApiUsersQuery();
	const { disks: volumes, isLoading: vlLoading, error: vlError } = useVolume();
	const [editName, setEditName] = useState(false);
	// Casing cycle state should be managed here if it's reset by volume selection
	const [activeCasingIndex, setActiveCasingIndex] = useState(0);
	const {
		control,
		handleSubmit,
		watch,
		formState: { errors },
		reset,
		setValue,
		getValues,
	} = useForm<ShareEditProps>(
			// Removed initial values from here, will be handled by useEffect + reset
		);
	const isDisabled = watch("disabled");
	const [availablePartitions, setAvailablePartition] = useState<MountPointData[]>([]);

	useEffect(() => {
		if (volumes) {
			const newAvailablePartitions = volumes
				?.flatMap((disk) => disk.partitions)
				?.filter(Boolean)
				.filter(
					(partition) =>
						!(partition?.system && partition?.host_mount_point_data && partition?.host_mount_point_data.length > 0)
				)
				.filter((partition) => partition?.mount_point_data)
				.flatMap(
					(partition) => partition?.mount_point_data,
				)
				.filter(
					(mp) => mp?.path !== "",
				) as MountPointData[] || [];
			setAvailablePartition(newAvailablePartitions);
		}
	}, [volumes]);

	useEffect(() => {
		if (props.open) {
			const adminUser = Array.isArray(users)
				? users.find((u) => u.is_admin)
				: undefined;
			if (props.objectToEdit) {
				// Covers editing existing share OR new share with prefill
				const isNewShareCreation = props.objectToEdit.org_name === undefined;
				reset({
					org_name: props.objectToEdit.org_name, // Key to determine if new/edit
					name: props.objectToEdit.name || "",
					mount_point_data: props.objectToEdit.mount_point_data, // This is the preselection
					// If it's a new share creation and no users are pre-filled, default to admin.
					// Otherwise, use the users from objectToEdit (could be empty for new, or populated for existing).
					users:
						props.objectToEdit.mount_point_data?.is_write_supported ?
							(isNewShareCreation &&
								(!props.objectToEdit.users ||
									props.objectToEdit.users.length === 0) &&
								adminUser
								? [adminUser]
								: props.objectToEdit.users || []) : [],
					ro_users: props.objectToEdit.mount_point_data?.is_write_supported ?
						(props.objectToEdit.ro_users || []) : (isNewShareCreation &&
							(!props.objectToEdit.ro_users ||
								props.objectToEdit.ro_users.length === 0) &&
							adminUser
							? [adminUser]
							: props.objectToEdit.ro_users || []),
					timemachine: props.objectToEdit.mount_point_data?.time_machine_support === Time_machine_support.Unsupported ? false : (props.objectToEdit.timemachine || false),
					recycle_bin_enabled: (props.objectToEdit.recycle_bin_enabled || false),
					guest_ok: props.objectToEdit.guest_ok || false,
					timemachine_max_size: props.objectToEdit.timemachine_max_size ||
						(props.objectToEdit.mount_point_data?.disk_size ? filesize(props.objectToEdit.mount_point_data?.disk_size) : "MAX"),
					usage: props.objectToEdit.usage || Usage.None,
					veto_files: props.objectToEdit.veto_files || [],
					disabled: props.objectToEdit.disabled,

					// any other fields from ShareEditProps that might be in objectToEdit
				});
				setEditName(isNewShareCreation); // Enable name edit for new shares
				setActiveCasingIndex(0); // Reset casing cycle state
			} else {
				// Completely new share, no prefill (e.g., user clicked "+" button directly)
				reset({
					org_name: undefined,
					name: "",
					users: adminUser ? [adminUser] : [], // Default to admin user if available
					ro_users: [],
					timemachine: false,
					usage: Usage.None,
					veto_files: [],
					disabled: false,
					// mount_point_data will be undefined, user must select
				});
				setEditName(true);
				setActiveCasingIndex(0); // Reset casing cycle state
			}
		} else {
			reset({
				// Reset to a clean state when dialog is not open
				org_name: undefined,
				name: "",
				users: [],
				ro_users: [],
				timemachine: false,
				usage: Usage.None,
				veto_files: [],
				disabled: false,
			}); // Reset to default values when closing or not open
		}
	}, [props.open, reset, users, props.objectToEdit]);

	// Effect to auto-populate share name if empty when a volume is selected
	const selectedMountPointData = watch("mount_point_data");
	const currentShareName = watch("name");

	useEffect(() => {
		if (
			props.open &&
			(!currentShareName || currentShareName.trim() === "") &&
			selectedMountPointData &&
			selectedMountPointData.path
		) {
			const baseName = getPathBaseName(selectedMountPointData.path);
			if (baseName) {
				const suggestedName = sanitizeAndUppercaseShareName(baseName);
				// Only update if the name is truly empty or different from the suggestion
				// to avoid unnecessary re-renders or dirtying the form.
				if (currentShareName !== suggestedName) {
					setValue("name", suggestedName, {
						shouldValidate: true,
						shouldDirty: true,
					});
					setActiveCasingIndex(0); // Reset casing cycle when name is auto-populated
				}
			}
		}
	}, [props.open, selectedMountPointData, currentShareName, setValue]);

	function handleCloseSubmit(data?: ShareEditProps) {
		setEditName(false);
		if (!data) {
			props.onClose();
			return;
		}
		console.log(data);
		props.onClose(data);
	}

	const handleCycleCasing = () => {
		const currentName = watch("name");
		if (typeof currentName !== "string") return;

		const styleToApply = casingCycleOrder[activeCasingIndex];
		let transformedName = currentName;

		switch (styleToApply) {
			case "UPPERCASE":
				transformedName = currentName.toUpperCase();
				break;
			case "lowercase":
				transformedName = currentName.toLowerCase();
				break;
			case "camelCase":
				transformedName = toCamelCase(currentName);
				break;
			case "kebab-case":
				transformedName = toKebabCase(currentName);
				break;
		}
		setValue("name", transformedName, {
			shouldValidate: true,
			shouldDirty: true,
		});
		setActiveCasingIndex(
			(prevIndex) => (prevIndex + 1) % casingCycleOrder.length,
		);
	};

	const nextCasingStyleName = casingCycleOrder[activeCasingIndex];
	const cycleCasingTooltipTitle = `Cycle casing (Next: ${nextCasingStyleName.charAt(0).toUpperCase() + nextCasingStyleName.slice(1)
		})`;
	const CasingIconToDisplay = getCasingIcon(nextCasingStyleName);

	return (
		<Fragment>
			<Dialog
				open={props.open}
				onClose={(_event, reason) => {
					if (reason && reason === "backdropClick") {
						return; // Prevent dialog from closing on backdrop click
					}
					handleCloseSubmit(); // Proceed with closing for other reasons (e.g., explicit button calls)
				}}
			>
				<DialogTitle
					sx={{
						display: "flex",
						alignItems: "center",
						justifyContent: "space-between",
					}}
				>
					<Stack direction="row" spacing={2} alignItems="center" sx={{ flex: 'auto' }}>
						{!(editName || props.objectToEdit?.org_name === undefined) && (
							<Box sx={{ display: "flex", alignItems: "center", flexGrow: 'inherit' }}>
								<>
									<IconButton onClick={() => setEditName(true)}>
										<ModeEditIcon fontSize="small" />
									</IconButton>
									{props.objectToEdit?.name}
								</>
							</Box>
						)}
						{(editName || props.objectToEdit?.org_name === undefined) && (
							<TextFieldElement
								sx={{ display: "flex", flexGrow: 'inherit' }}
								name="name"
								label="Share Name"
								required
								size="small"
								disabled={isDisabled}
								rules={{
									required: "Share name is required",
									pattern: {
										// Allows letters, numbers, and underscores
										value: /^[a-zA-Z0-9_]+$/,
										message:
											"Share name can only contain letters, numbers, and underscores (_)",
									},
									maxLength: {
										value: 80, // A common practical limit, adjust if your backend has a different rule
										message: "Share name cannot exceed 80 characters",
									},
								}}
								control={control}
								slotProps={{
									input: {
										endAdornment: (
											<InputAdornment position="end">
												<Tooltip title={cycleCasingTooltipTitle}>
													<IconButton
														aria-label="cycle share name casing"
														onClick={handleCycleCasing}
														edge="end"
													>
														<CasingIconToDisplay />
													</IconButton>
												</Tooltip>
											</InputAdornment>
										),
									},
								}}
							/>
						)}

						{/* Show the enable/disable switch only if it's an existing share */}
						{props.objectToEdit?.org_name !== undefined && (
							<>
								<SwitchElement
									switchProps={{
										size: "small",
									}}
									slotProps={{
										typography: {
											fontSize: "0.875rem",
										},
									}}
									control={control}

									name="disabled"
									color="primary"
									label={isDisabled ? "Disabled" : "Enabled"}
									sx={{ mr: 0 }}
								/>
							</>
						)}
					</Stack>
				</DialogTitle>
				<DialogContent>
					<Stack spacing={2}>
						<DialogContentText>
							Please enter or modify share properties.
						</DialogContentText>
						<form
							id="editshareform"
							onSubmit={handleSubmit(handleCloseSubmit)}
							noValidate
						>
							<Grid container spacing={2}>
								<Grid size={8}>
									{availablePartitions.length > 0 && (
										<>
											<AutocompleteElement
												label="Volume"
												name="mount_point_data"
												options={availablePartitions}
												control={control}
												required
												loading={vlLoading}
												autocompleteProps={{
													disabled: isDisabled,
													size: "small",
													renderValue: (value: MountPointData) => {
														//return ((value as MountPointData).path) || "--";
														return <Typography variant="body2">
															{value.disk_label || value.device_id} <sup>{value.is_write_supported ? "" : (<Typography variant="supper" color="error">Read-Only</Typography>)}</sup>
														</Typography>;

													},
													getOptionLabel: (option) =>
														(option as MountPointData)?.disk_label || "",
													getOptionKey: (option) =>
														(option as MountPointData)?.path_hash || "",
													renderOption: (props, option) => (
														<li {...props}>
															<Typography variant="body2">
																{option.disk_label || option.device_id} <sup>{option.is_write_supported ? "" : (<Typography variant="supper" color="error">Read-Only</Typography>)}</sup>
															</Typography>
														</li>
													),
													isOptionEqualToValue(option, value) {
														//console.log("Comparing", option, value);
														if (!value || !option) return false;
														return option.path_hash === value?.path_hash;
													},
													getOptionDisabled: (option) => {
														if (!props.shares || !option.path_hash) {
															return false; // Cannot determine, so don't disable
														}

														const currentEditingShareName =
															props.objectToEdit?.org_name;

														for (const existingShare of Object.values(
															props.shares,
														)) {
															if (
																existingShare.mount_point_data?.path_hash ===
																option.path_hash
															) {
																// This mount point is used by 'existingShare'.
																// If we are editing 'existingShare' itself, then this option should NOT be disabled.
																if (
																	currentEditingShareName &&
																	existingShare.name ===
																	currentEditingShareName
																) {
																	return false; // It's the current share's mount point, allow selection
																}
																return true; // Disable it, as it's used by another share or we are creating a new share
															}
														}
														return false; // Not used by any other share
													},
												}}
											/>
										</>
									)}
								</Grid>
								{props.objectToEdit?.usage !== Usage.Internal && (
									<Grid size={4}>
										<SelectElement
											sx={{ display: "flex" }}
											size="small"
											label="Usage"
											name="usage"
											disabled={isDisabled}
											options={Object.keys(Usage)
												.filter(
													(usage) =>
														usage.toLowerCase() !== Usage.Internal,
												)
												.map((usage) => {
													return { id: usage.toLowerCase(), label: usage };
												})}
											required
											control={control}
										/>
									</Grid>
								)}

								<Grid size={12}>
									<Controller
										name="veto_files"
										control={control}
										defaultValue={[]}
										rules={{
											validate: (chips: string[] | undefined) => {
												if (
													!chips ||
													chips == null ||
													chips.length === 0
												)
													return true; // Allow empty list
												for (const chip of chips) {
													if (!isValidVetoFileEntry(chip)) {
														return `Invalid entry: "${chip}". Veto file entries cannot be empty, contain '/' or null characters.`;
													}
												}
												return true;
											},
										}}
										render={({ field, fieldState: { error } }) => (
											<MuiChipsInput
												{...field}
												disabled={isDisabled}
												size="small"
												fullWidth
												hideClearAll
												label="Veto Files"
												validate={(chipValue) =>
													typeof chipValue === "string" &&
													isValidVetoFileEntry(chipValue)
												}
												error={!!error}
												helperText={
													error
														? error.message
														: "List of files/patterns to hide (e.g., ._* Thumbs.db). Entries cannot contain '/'."
												}
												renderChip={(Component, key, props) => {
													const isDefault =
														default_json.veto_files?.includes(
															props.label as string,
														);
													return (
														<Component
															key={key}
															{...props}
															sx={{
																color: isDefault
																	? "text.secondary"
																	: "text.primary",
															}}
															size="small"
														/>
													);
												}}
												slotProps={{
													input: {
														endAdornment: (
															<InputAdornment
																position="end"
																sx={{ pr: 1 }}
															>
																<Tooltip title="Add suggested default Veto files">
																	<span>
																		<IconButton
																			disabled={isDisabled}
																			aria-label="add suggested default veto files"
																			onClick={() => {
																				const currentVetoFiles:
																					string[] =
																					getValues(
																						"veto_files",
																					) || [];
																				const defaultVetoFiles:
																					string[] =
																					default_json.veto_files ||
																					[];
																				const newVetoFilesToAdd =
																					defaultVetoFiles.filter(
																						(defaultFile) =>
																							!currentVetoFiles.includes(
																								defaultFile,
																							),
																					);
																				setValue(
																					"veto_files",
																					[
																						...currentVetoFiles,
																						...newVetoFilesToAdd,
																					],
																					{
																						shouldDirty: true,
																						shouldValidate: true,
																					},
																				);
																			}}
																			edge="end"
																		>
																			<PlaylistAddIcon />
																		</IconButton>
																	</span>
																</Tooltip>
															</InputAdornment>
														),
													},
												}}
											/>
										)}
									/>
								</Grid>
								{watch("mount_point_data")?.is_write_supported && (
									<Grid size={6}>
										<Tooltip
											title={`Time Machine is ${watch("mount_point_data")?.time_machine_support} for the current volume!`}
										>
											<span>
												<SwitchElement
													switchProps={{
														size: "small",
														color: watch("mount_point_data")?.time_machine_support !== Time_machine_support.Supported ? "error" : "primary"
													}}
													label="Support Timemachine Backups"
													slotProps={{
														typography: {
															fontSize: "0.875rem",
															color: watch("mount_point_data")?.time_machine_support !== Time_machine_support.Supported ? "error" : "default"
														},
													}}
													name="timemachine"
													disabled={isDisabled || watch("mount_point_data")?.time_machine_support === Time_machine_support.Unsupported}
													control={control}
												/>
											</span>
										</Tooltip>
									</Grid>
								)}
								{watch("timemachine") && (
									<Grid size={6}>
										<TextFieldElement
											size="small"
											label="Time Machine Max Size (e.g., 100G, 5T, MAX)"
											name="timemachine_max_size"
											sx={{ display: "flex" }}
											disabled={isDisabled}
											control={control}
											rules={{
												pattern: {
													value: /^(MAX|\d+[KMGTP]?)$/i,
													message: "Invalid format. Use MAX or a number followed by K, M, G, T, P (e.g., 100G, 5T).",
												},
											}}
										/>
									</Grid>
								)}
								{watch("mount_point_data")?.is_write_supported && (
									<Grid size={6}>
										<SwitchElement
											switchProps={{
												size: "small",
											}}
											slotProps={{
												typography: {
													fontSize: "0.875rem",
												},
											}}
											label="Support Recycle Bin"
											name="recycle_bin_enabled"
											disabled={isDisabled}
											control={control}
										/>
									</Grid>
								)}
								<Grid size={12}>
									<SwitchElement
										switchProps={{
											size: "small",
										}}
										slotProps={{
											typography: {
												fontSize: "0.875rem",
											},
										}}
										label="Guest Access"
										name="guest_ok"
										disabled={isDisabled}
										control={control}
									/>
								</Grid>
								{!watch("guest_ok") && (
									<Grid size={6}>
										{!usLoading && ((users as User[]) || []).length > 0 && (
											<AutocompleteElement
												multiple
												name="users"
												label="Read and Write users"
												options={usLoading ? [] : (users as User[]) || []} // Use string keys for options
												control={control}
												loading={usLoading}
												autocompleteProps={{
													disabled: isDisabled || watch("mount_point_data")?.is_write_supported === false,
													size: "small",
													limitTags: 5,
													getOptionKey: (option) =>
														(option as User).username || "",
													getOptionLabel: (option) =>
														(option as User).username || "",
													renderOption: (props, option) => (
														<li {...props}>
															<Typography
																variant="body2"
																color={option.is_admin ? "warning" : "default"}
															>
																{option.username}
															</Typography>
														</li>
													),
													getOptionDisabled: (option) => {
														if (
															watch("ro_users")?.find(
																(user) =>
																	user.username === option.username,
															)
														) {
															return true; // Disable if the user is already in the users list
														}
														return false;
													},
													isOptionEqualToValue(option, value) {
														return option.username === value.username;
													},
													renderValue: (values, getItemProps) =>
														values.map((option, index) => {
															const { key, ...itemProps } = getItemProps({
																index,
															});
															//console.log(values, option)
															return (
																<Chip
																	color={
																		(option as User).is_admin
																			? "warning"
																			: "default"
																	}
																	key={key}
																	variant="outlined"
																	label={
																		(option as User)?.username || "bobo"
																	}
																	size="small"
																	{...itemProps}
																/>
															);
														}),
												}}
												textFieldProps={{
													//helperText: fsError ? 'Error loading filesystems' : (fsLoading ? 'Loading...' : 'Leave blank to auto-detect'),
													//error: !!fsError,

													InputLabelProps: { shrink: true },
												}}
											/>
										)}
									</Grid>
								)}
								<Grid size={6}>
									{!usLoading && ((users as User[]) || []).length > 0 && !watch("guest_ok") && (
										<AutocompleteElement
											multiple
											name="ro_users"
											label="Read Only users"
											options={usLoading ? [] : (users as User[]) || []} // Use string keys for options
											control={control}
											loading={usLoading}
											autocompleteProps={{
												disabled: isDisabled,
												size: "small",
												limitTags: 5,
												getOptionKey: (option) =>
													(option as User).username || "",
												getOptionLabel: (option) =>
													(option as User).username || "",
												renderOption: (props, option) => (
													<li {...props}>
														<Typography
															variant="body2"
															color={option.is_admin ? "warning" : "default"}
														>
															{option.username}
														</Typography>
													</li>
												),
												getOptionDisabled: (option) => {
													if (
														watch("users")?.find(
															(user) =>
																user.username === option.username,
														)
													) {
														return true; // Disable if the user is already in the users list
													}
													return false;
												},
												isOptionEqualToValue(option, value) {
													return option.username === value.username;
												},
												renderValue: (values, getItemProps) =>
													values.map((option, index) => {
														const { key, ...itemProps } = getItemProps({
															index,
														});
														//console.log(values, option)
														return (
															<Chip
																color={
																	(option as User).is_admin
																		? "warning"
																		: "default"
																}
																key={key}
																variant="outlined"
																label={
																	(option as User)?.username || "bobo"
																}
																size="small"
																{...itemProps}
															/>
														);
													}),
											}}
											textFieldProps={{
												//helperText: fsError ? 'Error loading filesystems' : (fsLoading ? 'Loading...' : 'Leave blank to auto-detect'),
												//error: !!fsError,

												InputLabelProps: { shrink: true },
											}}
										/>
									)}
								</Grid>
							</Grid>
						</form>
					</Stack>
				</DialogContent>
				<DialogActions>
					{props.objectToEdit?.org_name && props.onDeleteSubmit && (
						<Button
							onClick={() => {
								// Ensure objectToEdit and org_name are valid before calling onDeleteSubmit
								if (props.objectToEdit?.org_name && props.onDeleteSubmit) {
									props.onDeleteSubmit(
										props.objectToEdit.org_name,
										props.objectToEdit,
									);
								}
								handleCloseSubmit(); // Close the dialog
							}}
							color="error"
							variant="outlined"
						>
							Delete
						</Button>
					)}
					<Button onClick={() => handleCloseSubmit()}>Cancel</Button>
					<Button type="submit" form="editshareform" variant="contained">
						{props.objectToEdit?.org_name === undefined ? "Create" : "Apply"}
					</Button>
				</DialogActions>
			</Dialog>
		</Fragment>
	);
}
