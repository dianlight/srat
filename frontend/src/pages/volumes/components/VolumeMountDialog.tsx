import {
	Button,
	Chip,
	Dialog,
	DialogActions,
	DialogContent,
	DialogContentText,
	DialogTitle,
	Grid,
	Stack,
	Tooltip,
} from "@mui/material";
import { Fragment, useEffect, useState } from "react";
import {
	AutocompleteElement,
	CheckboxElement,
	TextFieldElement,
	useFieldArray,
	useForm,
} from "react-hook-form-mui";
import {
	type FilesystemType,
	type MountFlag,
	type MountPointData,
	type Partition,
	Type,
	useGetApiFilesystemsQuery,
} from "../../../store/sratApi";
import { decodeEscapeSequence, generateSHA1Hash } from "../utils";

interface xMountPointData extends MountPointData {
	custom_flags_values: MountFlag[]; // Array of custom flags (enum) for the TextField
}

interface VolumeMountDialogProps {
	open: boolean;
	onClose: (data?: MountPointData) => void;
	objectToEdit?: Partition;
	readOnlyView?: boolean;
}

export function VolumeMountDialog(props: VolumeMountDialogProps) {
	const {
		control,
		handleSubmit,
		watch,
		reset,
		formState: { errors, isDirty },
		register,
		setValue,
	} = useForm<xMountPointData>({
		defaultValues: {
			path: "",
			fstype: "",
			flags: [],
			custom_flags: [],
			custom_flags_values: [],
			is_to_mount_at_startup: false,
		}, // Default values for the form
	});
	const { fields, replace } = useFieldArray({
		control, // control props comes from useForm (optional: if you are using FormProvider)
		name: "custom_flags_values", // unique name for your Field Array
	});
	const {
		data: filesystems,
		isLoading: fsLoading,
		error: fsError,
	} = useGetApiFilesystemsQuery();
	const [mounting, setMounting] = useState(false);

	// Use useEffect to update form values when objectToEdit changes or dialog opens
	useEffect(() => {
		if (props.open && props.objectToEdit) {
			const suggestedName = decodeEscapeSequence(
				props.objectToEdit.name || props.objectToEdit.id || "new_mount",
			);
			const sanitizedName = suggestedName.replace(/[\s\\/:"*?<>|]+/g, "_");
			const existingMountData = props.objectToEdit.mount_point_data?.[0];

			reset({
				path: existingMountData?.path || `/mnt/${sanitizedName}`,
				fstype: existingMountData?.fstype || undefined, // Use existing or let backend detect
				flags: existingMountData?.flags || [], // Keep numeric flags if needed internally
				custom_flags: existingMountData?.custom_flags || [], // Keep numeric flags if needed internally
				custom_flags_values: [], // Will be populated by `replace` below
				is_to_mount_at_startup:
					existingMountData?.is_to_mount_at_startup || false, // Initialize the switch state
			});

			setMounting(false);

			const valueFlags = ([] as MountFlag[]).concat(
				existingMountData?.custom_flags || [],
				existingMountData?.flags || [],
			);
			replace(
				valueFlags.filter((v) => v.needsValue).map((flag) => ({ ...flag })),
			); // Ensure we pass new objects to replace
		} else if (!props.open) {
			reset({
				path: "",
				fstype: "",
				flags: [],
				custom_flags: [],
				custom_flags_values: [],
				is_to_mount_at_startup: false,
			}); // Reset to default values when closing
		}
	}, [props.open, props.objectToEdit, reset, replace]); // Added `replace` to dependencies

	async function handleCloseSubmit(formData: xMountPointData) {
		if (props.readOnlyView) {
			props.onClose();
			return;
		}
		if (!props.objectToEdit) {
			console.error("Mount dialog submitted without an objectToEdit.");
			props.onClose();
			return;
		}

		const custom_flags = (formData.custom_flags || []).map((flag) => {
			if (
				formData.custom_flags_values &&
				formData.custom_flags_values.length > 0
			) {
				const flagValue = formData.custom_flags_values.find(
					(fv) => fv.name === flag.name,
				);
				return {
					...flag,
					value: flagValue ? flagValue.value : "", // Use the value from custom_flags_values if available
				};
			}
			return flag; // Return the flag as is if no custom values are provided
		});
		//console.debug("Form Data:", formData,custom_flags);

		const submitData: MountPointData = {
			path: formData.path,
			path_hash: await generateSHA1Hash(formData.path),
			fstype: formData.fstype || undefined,
			flags: formData.flags,
			custom_flags: custom_flags,
			//device: props.objectToEdit.device, // Ensure device name is included
			is_to_mount_at_startup: formData.is_to_mount_at_startup, // Include the switch value in submitted data
			type: Type.Addon,
		};
		//console.debug("Submitting Mount Data:", submitData);
		setMounting(true);
		props.onClose(submitData);
	}

	function handleCancel() {
		props.onClose(); // Call onClose without data
	}

	const partitionNameDecoded = decodeEscapeSequence(
		props.objectToEdit?.name || "Unnamed Partition",
	);
	const partitionId = props.objectToEdit?.id || "N/A";

	return (
		<Fragment>
			<Dialog open={props.open} onClose={handleCancel} maxWidth="sm" fullWidth>
				<DialogTitle>
					{props.readOnlyView ? "View Mount Settings: " : "Mount Volume: "}{" "}
					{partitionNameDecoded} ({partitionId})
				</DialogTitle>
				<form
					id="mountvolumeform"
					onSubmit={handleSubmit(async (data) => await handleCloseSubmit(data))}
					noValidate
				>
					<DialogContent>
						<Stack spacing={2} sx={{ pt: 1 }}>
							<DialogContentText>
								Configure mount options for the volume. The suggested path is
								based on the volume name.
							</DialogContentText>
							<Grid container spacing={2}>
								<Grid size={{ xs: 12, sm: 6 }}>
									<TextFieldElement
										size="small"
										name="path"
										label="Mount Path"
										control={control}
										required
										fullWidth
										disabled={props.readOnlyView}
										InputLabelProps={{ shrink: true }} // Ensure label is always shrunk
										helperText="Path must start with /mnt/"
									/>
								</Grid>
								<Grid size={{ xs: 12, sm: 6 }}>
									<AutocompleteElement
										name="fstype"
										label="File System Type"
										control={control}
										options={
											fsLoading
												? []
												: ((filesystems as FilesystemType[]) || []).map(
													(fs) => fs.name,
												)
										}
										autocompleteProps={{
											freeSolo: true,
											disabled: props.readOnlyView,
											size: "small",
											onChange: (_event, value) => {
												if (props.readOnlyView) return;
												console.log("FS Type changed:", value);
												setValue("custom_flags", []); // Clear custom flags when FS type changes
												setValue("custom_flags_values", []); // Clear custom flags values when FS type changes
												replace([]); // Clear field array for custom flag values
											},
										}}
										textFieldProps={{
											disabled: props.readOnlyView,
											helperText:
												fsError
													? "Error loading filesystems"
													: fsLoading
														? "Loading..."
														: "Leave blank to auto-detect",
											error: !!fsError,
											InputLabelProps: { shrink: true },
										}}
									/>
								</Grid>
								<Grid size={{ xs: 12, sm: 6 }}>
									{!fsLoading &&
										((filesystems as FilesystemType[])[0]?.mountFlags || [])
											.length > 0 && (
											<AutocompleteElement
												multiple
												name="flags"
												label="Mount Flags"
												options={
													fsLoading
														? []
														: (filesystems as FilesystemType[])[0]
															?.mountFlags || []
												} // Use string keys for options
												control={control}
												autocompleteProps={{
													disabled: props.readOnlyView,
													size: "small",
													limitTags: 5,
													getOptionLabel: (option) =>
														(option as MountFlag).name,
													renderOption: (props, option) => (
														<li {...props} >
															<Tooltip
																title={(option as MountFlag).description || ""}
															>
																<span>
																	{(option as MountFlag).name}{" "}
																	{(option as MountFlag).needsValue ? (
																		<span
																			style={{
																				fontSize: "0.8em",
																				color: "#888",
																			}}
																		>
																			(Requires Value)
																		</span>
																	) : null}
																</span>
															</Tooltip>
														</li>
													),
													isOptionEqualToValue(option, value) {
														return option.name === value.name;
													},
												}}
												textFieldProps={{
													disabled: props.readOnlyView,
													InputLabelProps: { shrink: true },
												}}
											/>
										)}
								</Grid>
								<Grid size={{ xs: 12, sm: 6 }}>
									{!fsLoading &&
										((
											(filesystems as FilesystemType[]).find(
												(fs) => fs.name === watch("fstype"),
											)?.customMountFlags || []
										).length > 0 && (
												<AutocompleteElement
													multiple
													name="custom_flags"
													label="FileSystem specific Mount Flags"
													options={
														fsLoading
															? []
															: (filesystems as FilesystemType[]).find(
																(fs) => fs.name === watch("fstype"),
															)?.customMountFlags || []
													}
													control={control}
													autocompleteProps={{
														disabled: props.readOnlyView,
														size: "small",
														limitTags: 5,
														getOptionLabel: (option) =>
															(option as MountFlag).name, // Ensure label is just the name
														renderOption: (props, option) => (
															<li {...props} key={(option as MountFlag).name}>
																<Tooltip
																	title={(option as MountFlag).description || ""}
																>
																	<span>
																		{(option as MountFlag).name}{" "}
																		{(option as MountFlag).needsValue ? (
																			<span
																				style={{
																					fontSize: "0.8em",
																					color: "#888",
																				}}
																			>
																				(Requires Value)
																			</span>
																		) : null}
																	</span>
																</Tooltip>
															</li>
														),
														isOptionEqualToValue(option, value) {
															return option.name === value.name;
														},
														renderTags: (values, getTagProps) =>
															values.map((option, index) => {
																const { key, ...tagProps } = getTagProps({
																	index,
																});
																return (
																	<Chip
																		color={
																			(option as MountFlag).needsValue
																				? "warning"
																				: "default"
																		}
																		key={key}
																		variant="filled" // "outlined" or "filled"
																		label={
																			(option as MountFlag)?.name || "error"
																		}
																		size="small"
																		{...tagProps}
																	/>
																);
															}),
														onChange: props.readOnlyView
															? undefined
															: (_event, value) => {
																const flagsWithValue = (
																	value as MountFlag[]
																).filter((v) => v.needsValue);
																const currentFieldValues =
																	watch("custom_flags_values") || [];

																// Filter out existing values for flags that are no longer selected
																const newFieldValues =
																	currentFieldValues.filter((fv) =>
																		flagsWithValue.some(
																			(selectedFlag) =>
																				selectedFlag.name === fv.name,
																		),
																	);

																// Add new placeholders for newly selected flags that need values
																flagsWithValue.forEach((selectedFlag) => {
																	if (
																		!newFieldValues.some(
																			(fv) =>
																				fv.name === selectedFlag.name,
																		)
																	) {
																		newFieldValues.push({
																			...selectedFlag,
																			value: selectedFlag.value || "",
																		}); // Use existing value or empty
																	}
																});
																replace(newFieldValues);
																setValue(
																	"custom_flags",
																	value as MountFlag[],
																	{
																		shouldDirty: true,
																	},
																); // also update the custom_flags themselves
															},
													}}
													textFieldProps={{
														disabled: props.readOnlyView,
														InputLabelProps: { shrink: true },
													}}
												/>
											))}
								</Grid>
								{fields.map((field, index) => (
									<Grid size={{ xs: 12, sm: 6 }} key={field.id}>
										<TextFieldElement
											size="small"
											name={`custom_flags_values.${index}.value`}
											label={field.name} // This is MountFlag, so field.name is the flag name
											control={control}
											required
											fullWidth
											disabled={props.readOnlyView}
											variant="outlined"
											rules={{
												required: `Value for ${field.name} is required.`,
												pattern: {
													value: new RegExp(
														field.value_validation_regex || ".*",
													),
													message: `Invalid value for ${field.name}. ${field.value_description}`,
												},
											}}
											InputLabelProps={{ shrink: true }}
											helperText={field.value_description}
										/>
									</Grid>
								))}
								<Grid size={{ xs: 12 }}>
									<CheckboxElement
										name="is_to_mount_at_startup"
										label="Mount at startup"
										control={control}
										disabled={props.readOnlyView}
										size="small"
									/>
								</Grid>
							</Grid>
						</Stack>
					</DialogContent>
					<DialogActions>
						{props.readOnlyView ? (
							<Button
								onClick={handleCancel}
								color="primary"
								variant="contained"
							>
								Close
							</Button>
						) : (
							<>
								<Button onClick={handleCancel} color="secondary">
									Cancel
								</Button>
								<Button
									type="submit"
									form="mountvolumeform"
									disabled={mounting}
									variant="contained"
								>
									Mount
								</Button>{" "}
								{/* Corrected disabled prop */}
							</>
						)}
					</DialogActions>
				</form>
			</Dialog>
		</Fragment>
	);
}
