import {
    Button,
    Chip,
    Dialog,
    DialogActions,
    DialogContent,
    DialogContentText,
    DialogTitle,
    Divider,
    Grid,
    Stack,
    Tooltip,
    Typography
} from "@mui/material";
import { Fragment, useEffect, useMemo, useState } from "react";
import {
	AutocompleteElement,
	SwitchElement,
	TextFieldElement,
	useFieldArray,
	useForm
} from "react-hook-form-mui";
import {
	type FilesystemInfo,
	type MountFlag,
	type MountPointData,
	type Partition,
	Type,
	useGetApiFilesystemsQuery,
} from "../../../store/sratApi";
import { decodeEscapeSequence } from "../utils";

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
		formState: { errors },
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
	} = useGetApiFilesystemsQuery(undefined, { skip: !props.open });
	// Ensure we always have an array to avoid runtime errors when the query is skipped
	const fsList = Array.isArray(filesystems)
		? (filesystems as FilesystemInfo[])
		: [];
	const [mounting, setMounting] = useState(false);

	const [unsupported_flags, setUnsupportedFlags] = useState<MountFlag[]>([]); // Array of unsupported flags (string) for display only
	const [unsupported_custom_flags, setUnsupportedCustomFlags] = useState<MountFlag[]>([]); // Array of unsupported custom flags (string) for display only


	useMemo(() => { }, [errors]);

	// Use useEffect to update form values when objectToEdit changes or dialog opens
	useEffect(() => {
		if (props.open && props.objectToEdit) {
			const suggestedName = decodeEscapeSequence(
				props.objectToEdit.name || props.objectToEdit.id || "new_mount",
			);
			const sanitizedName = suggestedName.replace(/[\s\\/:"*?<>|]+/g, "_");
			const existingMountData = Object.values(props.objectToEdit.mount_point_data || {})[0];

			if (existingMountData?.fstype) {
				// If existing fstype is set, ensure it's in the filesystems list
				const fsCurrent = fsList.find((fs) => fs.name === existingMountData.fstype);
				if (fsCurrent) {
					setUnsupportedFlags([]); // Reset before checking
					setUnsupportedCustomFlags([]); // Reset before checking
					// Check existing flags against supported flags for this FS
					existingMountData?.flags?.forEach((flag) => {
						console.log("Checking flag:", flag, fsCurrent.mountFlags);
						if (!fsCurrent.mountFlags?.find((flagItem) => flagItem.name === flag.name)) {
							setUnsupportedFlags((prev) => [...prev, flag]);
						}
					});
					existingMountData?.custom_flags?.forEach((flag) => {
						console.log("Checking custom flag:", flag, fsCurrent.customMountFlags);
						if (!fsCurrent.customMountFlags?.find((flagItem) => flagItem.name === flag.name)) {
							setUnsupportedCustomFlags((prev) => [...prev, flag]);
						}
					});
				} else {
					// FSType not found in current list, consider all flags unsupported
					setUnsupportedFlags(existingMountData?.flags || []);
					setUnsupportedCustomFlags(existingMountData?.custom_flags || []);
				}
			}

			reset({
				path: existingMountData?.path || `/mnt/${sanitizedName}`,
				fstype: existingMountData?.fstype || props.objectToEdit?.fs_type || undefined, // Use existing or let backend detect
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
			setUnsupportedCustomFlags([]);
			setUnsupportedFlags([]);
			setMounting(false);
			reset({
				path: "",
				fstype: "",
				flags: [],
				custom_flags: [],
				custom_flags_values: [],
				is_to_mount_at_startup: false,
			}); // Reset to default values when closing
		}
	}, [props.open, props.objectToEdit, reset, replace]);

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
			root: "/",
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
		props.objectToEdit?.name || props.objectToEdit?.legacy_device_name || "Unnamed Partition",
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
								Configure mount options for the volume.
							</DialogContentText>
							<Grid container spacing={2}>
								{/*
								<Grid size={{ xs: 12, sm: 6 }}>
									<TextFieldElement
										hidden={true}
										size="small"
										name="path"
										label="Mount Path"
										control={control}
										required
										fullWidth
										disabled={props.readOnlyView}
										slotProps={{
											inputLabel: {
												shrink: true,
											},
										}}
										helperText="Path must start with /mnt/"
									/>
								</Grid>
									*/}
								<Grid size={{ xs: 12, sm: 12 }}>
									<AutocompleteElement
										name="fstype"
										label="File System Type"
										control={control}
										options={
											fsLoading
												? []
												: fsList.map((fs) => fs.name)
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
									{!fsLoading && (fsList.find((fs) => fs.name === watch("fstype"))?.mountFlags || []).length > 0 && (
										<AutocompleteElement<MountFlag, true, false, false, "div", xMountPointData, "flags">
											multiple
											name="flags"
											label="Mount Flags"
											options={fsLoading ? [] : fsList.find((fs) => fs.name === watch("fstype"))?.mountFlags || []} // Use string keys for options
											control={control}
											autocompleteProps={{
												disabled: props.readOnlyView,
												size: "small",
												limitTags: 7,
												getOptionLabel: (option) =>
													(option as MountFlag).name, // Ensure label is just the name
												renderOption: ({ key, ...restProps }, option) => (
													<li key={key} {...restProps}  >
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
												isOptionEqualToValue: (option, value) => option.name === value.name,
												renderTags: (values, getTagProps) =>
													values.filter((option) => option != null).map((option, index) => {
														const { key, ...tagProps } = getTagProps({
															index,
														});
														return (
															<Chip
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
											}}
											textFieldProps={{
												disabled: props.readOnlyView,
												InputLabelProps: { shrink: true },
											}}
										/>
									)}
									{unsupported_flags.length > 0 && (
										<Typography fontSize="0.8em" color="error">Unknown Flags: {unsupported_flags?.map(flag => flag.name).join(", ")}</Typography>
									)}
								</Grid>
								<Grid size={{ xs: 12, sm: 6 }}>
									{!fsLoading && ((fsList.find((fs) => fs.name === watch("fstype"))?.customMountFlags || [])).length > 0 && (
										<AutocompleteElement<MountFlag, true, false, false, "div", xMountPointData, "custom_flags">
											multiple
											name="custom_flags"
											label="FileSystem specific Mount Flags"
											options={fsLoading ? [] : fsList.find((fs) => fs.name === watch("fstype"))?.customMountFlags || []}
											control={control}
											autocompleteProps={{
												disabled: props.readOnlyView,
												size: "small",
												limitTags: 7,
												getOptionLabel: (option) =>
													(option as MountFlag).name, // Ensure label is just the name
												renderOption: ({ key, ...props }, option) => (
													<li key={(option as MountFlag).name}{...props} >
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
												isOptionEqualToValue: (option, value) => option.name === value.name,
												renderTags: (values, getTagProps) =>
													values.filter((option) => option != null).map((option, index) => {
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
									)}
									{unsupported_custom_flags.length > 0 && (
										<Typography fontSize="0.8em" color="error">
											Unknown Flags: {unsupported_custom_flags?.map(flag => flag.name).join(", ")}
										</Typography>
									)}
								</Grid>
								<Grid size={{ xs: 12 }}>
									<Divider />
								</Grid>
								{fields.map((field, index) => (
									<Grid size={{ xs: 12, sm: 6 }} key={field.id + "_edit_"}>
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
									<SwitchElement
										switchProps={{
											size: "small",
										}}
										slotProps={{
											typography: {
												fontSize: "0.875rem",
											},
										}}
										name="is_to_mount_at_startup"
										label="Automatic mount"
										control={control}
										disabled={props.readOnlyView}
									/>
								</Grid>
							</Grid>
						</Stack>
					</DialogContent>
					<DialogActions>
						{props.readOnlyView ? (
							<Button
								onClick={handleCancel}
								color="secondary"
								variant="outlined"
							>
								Close
							</Button>
						) : (
							<>
								<Button onClick={handleCancel} color="secondary" variant="outlined">
									Cancel
								</Button>
								<Button
									type="submit"
									form="mountvolumeform"
									disabled={mounting}
									variant="outlined"
									color="success"
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
