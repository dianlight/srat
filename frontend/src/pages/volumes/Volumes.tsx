import ComputerIcon from "@mui/icons-material/Computer";
import CreditScoreIcon from "@mui/icons-material/CreditScore";
import EjectIcon from "@mui/icons-material/Eject";
import ExpandMore from "@mui/icons-material/ExpandMore";
import SdStorageIcon from "@mui/icons-material/SdStorage";
import SettingsSuggestIcon from "@mui/icons-material/SettingsSuggest";
import StorageIcon from "@mui/icons-material/Storage";
import UsbIcon from "@mui/icons-material/Usb";
import {
	Accordion,
	AccordionDetails,
	AccordionSummary,
	Avatar,
	Chip,
	Divider,
	FormControlLabel,
	ListItem,
	ListItemAvatar,
	ListItemButton,
	ListItemText,
	Stack,
	Switch,
	Typography,
} from "@mui/material";
import List from "@mui/material/List";
import { filesize } from "filesize";
import { useConfirm } from "material-ui-confirm";
import { Fragment, useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router";
import { toast } from "react-toastify";
import { PreviewDialog } from "../../components/PreviewDialog";
import { useReadOnly } from "../../hooks/readonlyHook";
import { useVolume } from "../../hooks/volumeHook";
import { type LocationState, TabIDs } from "../../store/locationState";
import {
	type Disk,
	type MountPointData,
	type Partition,
	useDeleteApiVolumeByMountPathHashMountMutation,
	usePatchApiVolumeByMountPathHashSettingsMutation,
	usePostApiVolumeByMountPathHashMountMutation,
} from "../../store/sratApi";
import { PartitionActions } from "./components/PartitionActions";
import { VolumeMountDialog } from "./components/VolumeMountDialog";
import { useVolumeAccordion } from "./hooks/useVolumeAccordion";
import { decodeEscapeSequence } from "./utils";

export function Volumes() {
	const read_only = useReadOnly();
	const [showPreview, setShowPreview] = useState<boolean>(false);
	const [showMount, setShowMount] = useState<boolean>(false);
	const [showMountSettings, setShowMountSettings] = useState<boolean>(false); // For viewing mount settings
	const location = useLocation();

	const navigate = useNavigate();
	const [hideSystemPartitions, setHideSystemPartitions] =
		useState<boolean>(true); // Default to hide system partitions
	const { disks, isLoading, error } = useVolume();
	const [selected, setSelected] = useState<Partition | Disk | undefined>(
		undefined,
	); // Can hold a disk or partition
	const confirm = useConfirm();
	const [mountVolume, _mountVolumeResult] =
		usePostApiVolumeByMountPathHashMountMutation();
	const [umountVolume, _umountVolumeResult] =
		useDeleteApiVolumeByMountPathHashMountMutation();
	const [patchMountSettings] = usePatchApiVolumeByMountPathHashSettingsMutation();

	const {
		expandedAccordion,
		handleAccordionChange,
		isDiskRendered,
		setExpandedAccordion,
	} = useVolumeAccordion(disks, isLoading, hideSystemPartitions);

	// Effect to handle navigation state for opening mount settings for a specific volume
	useEffect(() => {
		const state = location.state as LocationState | undefined;
		const mountPathHashFromState = state?.mountPathHashToView;
		const shouldOpenMountSettings = state?.openMountSettings;

		if (
			mountPathHashFromState &&
			shouldOpenMountSettings &&
			Array.isArray(disks) &&
			disks.length > 0
		) {
			let foundPartition: Partition | undefined;
			let foundDiskIdentifier: string | undefined;

			for (const disk of disks) {
				if (disk.partitions) {
					for (const partition of disk.partitions) {
						const diskIdx = disks.indexOf(disk);
						if (
							partition.mount_point_data?.some(
								(mpd) => mpd.path_hash === mountPathHashFromState,
							)
						) {
							foundPartition = partition;
							foundDiskIdentifier = disk.id || `disk-${diskIdx}`;
							break;
						}
					}
				}
				if (foundPartition) break;
			}

			if (foundPartition && foundDiskIdentifier) {
				setSelected(foundPartition);
				setShowMountSettings(true);
				const targetDisk = disks.find(
					(d, idx) => (d.id || `disk-${idx}`) === foundDiskIdentifier,
				);
				if (targetDisk && isDiskRendered(targetDisk, hideSystemPartitions)) {
					setExpandedAccordion(foundDiskIdentifier);
				}
				navigate(location.pathname, { replace: true, state: {} });
			} else {
				console.warn(
					`Volume with mountPathHash ${mountPathHashFromState} not found.`,
				);
				navigate(location.pathname, { replace: true, state: {} });
			}
		}
	}, [
		disks,
		location.state,
		navigate,
		hideSystemPartitions,
		isDiskRendered,
		location.pathname,
		setExpandedAccordion,
	]);

	function onSubmitMountVolume(data?: MountPointData) {
		console.trace("Mount Request Data:", data);
		// Type guard to check if selected is a Partition
		const isPartition = (item: any): item is Partition =>
			item && !(item as Disk).partitions;

		if (!selected || !isPartition(selected) || !data || !data.path) {
			toast.error("Cannot mount: Invalid selection or missing data.");
			console.error("Mount validation failed:", {
				selected,
				isPartition: isPartition(selected),
				data,
			});
			return;
		}

		// Ensure device is included in submitData if required by API
		const submitData: MountPointData = {
			...data,
			device: selected.device,
		};
		//console.log("Submitting Mount Data:", submitData);

		mountVolume({
			mountPathHash: data.path_hash || "",
			mountPointData: submitData,
		})
			.unwrap()
			.then((res) => {
				toast.info(
					`Volume ${(res as MountPointData).path || selected.name} mounted successfully.`,
				);
			})
			.catch((err) => {
				console.error("Mount Error:", err);
				const errorData = err?.data || {};
				const errorMsg =
					errorData?.detail ||
					errorData?.message ||
					err?.status ||
					"Unknown mount error";
				const errorCode = errorData?.status || "Error";
				toast.error(`${errorCode}: ${errorMsg}`, {
					data: { error: errorData || err },
				});
			})
			.finally(() => {
				setSelected(undefined); // Clear selection after successful mount
				setShowMount(false); // Close the mount dialog
			});
	}

	function handleCreateShare(partition: Partition) {
		const firstMountPointData = partition.mount_point_data?.[0];
		if (firstMountPointData?.path) {
			// Ensure path exists for preselection
			navigate("/", {
				state: {
					tabId: TabIDs.SHARES,
					newShareData: firstMountPointData,
				} as LocationState,
			});
		} else {
			toast.warn(
				"Cannot create share: Partition is not mounted or has no mount path.",
			);
		}
	}

	function handleGoToShare(partition: Partition) {
		//console.log("Go to share for:", partition);
		const mountData = partition.mount_point_data?.[0];
		const share = mountData?.shares?.[0]; // Get the first share associated with this mount point

		if (share?.name) {
			// Navigate to the shares page and pass the share name as state
			navigate("/", {
				state: { tabId: TabIDs.SHARES, shareName: share.name } as LocationState,
			}); // Navigate to root, NavBar handles tab
		}
	}

	function onSubmitUmountVolume(partition: Partition, force = false) {
		console.log("Umount Request", partition, "Force:", force);
		// Ensure mount_point_data exists and has at least one entry with a path
		const mountData = partition.mount_point_data?.[0];
		if (!mountData?.path) {
			toast.error("Cannot unmount: Missing mount point path.");
			console.error("Missing mount path for partition:", partition);
			return;
		}

		// Use partition label or name for confirmation dialog
		const displayName = decodeEscapeSequence(partition.name || "this volume");

		confirm({
			title: `Unmount ${displayName}?`,
			description: `Do you really want to ${force ? "forcefully " : ""}unmount the Volume ${displayName} (${partition.device}) mounted at ${mountData.path}?`,
			confirmationText: force ? "Force Unmount" : "Unmount",
			cancellationText: "Cancel",
			confirmationButtonProps: { color: force ? "error" : "primary" },
			acknowledgement: `Please confirm this action carefully. Unmounting may lead to data loss or corruption if the volume is in use. ${force ? "NOTE:Configured shares will be disabled!" : ""}`,
		}).then(({ reason }) => {
			// Only proceed if confirmed
			if (reason === "confirm") {
				console.log(
					`Proceeding with ${force ? "forced " : ""}unmount for:`,
					mountData.path,
				);
				umountVolume({
					mountPathHash: mountData.path_hash || "", // Use the extracted path
					force: force,
					lazy: true, // Consider if lazy unmount is always desired
				})
					.unwrap()
					.then(() => {
						toast.info(`Volume ${displayName} unmounted successfully.`);
						// Optionally clear selection if the unmounted item was selected
						if (selected?.id === partition.id) {
							setSelected(undefined);
						}
					})
					.catch((err) => {
						console.error("Unmount Error:", err);
						const errorData = err?.data || {};
						const errorMsg =
							errorData?.message || err?.status || "Unknown error";
						toast.error(`Error unmounting ${displayName}: ${errorMsg}`, {
							data: { error: err },
						});
					});
			}
		});
	}

	function handleToggleAutomount(partition: Partition) {
		if (read_only) return;

		const mountData = partition.mount_point_data?.[0];
		if (!mountData || !mountData.path_hash) {
			toast.error("Cannot toggle automount: Missing mount point data.");
			console.error("Missing mount data for partition:", partition);
			return;
		}

		const newAutomountState = !mountData.is_to_mount_at_startup;
		const actionText = newAutomountState ? "enable" : "disable";
		const partitionName = decodeEscapeSequence(partition.name || "this volume");

		console.log(partition);

		patchMountSettings({
			mountPathHash: mountData.path_hash,
			mountPointData: {
				...mountData,
				is_to_mount_at_startup: newAutomountState,
			},
		})
			.unwrap()
			.then(() => {
				toast.info(`Automount ${actionText}d for ${partitionName}.`);
			})
			.catch((err: any) => {
				console.error(`Error toggling automount for ${partitionName}:`, err);
				toast.error(
					`Failed to ${actionText} automount for ${partitionName}: ${err.data?.detail || err.message || "Unknown error"}`,
				);
			});
	}
	// Handle loading and error states
	if (isLoading) {
		return <Typography>Loading volumes...</Typography>;
	}

	if (error) {
		// Provide a more user-friendly error message
		console.error("Error loading volumes:", error);
		return (
			<Typography color="error">
				Error loading volume information. Please try again later.
			</Typography>
		);
	}

	// Helper function to render disk icon
	const renderDiskIcon = (disk: Disk) => {
		switch (disk.connection_bus?.toLowerCase()) {
			case "usb":
				return <UsbIcon />;
			case "sdio":
			case "mmc":
				return <SdStorageIcon />;
		}
		if (disk.removable) {
			return <EjectIcon />;
		}
		// Add more specific icons based on bus or type if needed
		// e.g., if (disk.type === 'nvme') return <MemoryIcon />;
		return <ComputerIcon />;
	};

	// Helper function to render partition icon
	const renderPartitionIcon = (partition: Partition) => {
		const isToMountAtStartup =
			partition.mount_point_data?.[0]?.is_to_mount_at_startup === true;
		const iconColorProp = isToMountAtStartup
			? { color: "primary" as const }
			: {};

		if (partition.name === "hassos-data") {
			return <CreditScoreIcon fontSize="small" {...iconColorProp} />;
		}
		if (
			partition.system ||
			partition.name?.startsWith("hassos-") ||
			(partition.host_mount_point_data &&
				partition.host_mount_point_data.length > 0)
		) {
			return <SettingsSuggestIcon fontSize="small" {...iconColorProp} />;
		}
		return <StorageIcon fontSize="small" {...iconColorProp} />;
	};

	return (
		<>
			<VolumeMountDialog
				// Type guard to ensure we only pass Partitions to the mount dialog
				objectToEdit={
					selected && !(selected as Disk).partitions
						? (selected as Partition)
						: undefined
				}
				open={showMount || showMountSettings}
				readOnlyView={showMountSettings}
				onClose={(data) => {
					if (showMountSettings) {
						// If it was open for viewing settings
						setSelected(undefined);
						setShowMountSettings(false);
					} else if (showMount) {
						// If it was open for mounting
						if (data) {
							onSubmitMountVolume(data);
						} else {
							// Cancelled mount dialog or no data returned
							setSelected(undefined);
							setShowMount(false);
						}
					}
				}}
			/>
			{/* PreviewDialog can show details for both disks and partitions */}
			<PreviewDialog
				// Improved title logic using type guards
				title={
					selected
						? (selected as Disk).model // If it has a model, it's likely a Disk
							? `Disk: ${(selected as Disk).model}`
							: `Partition: ${decodeEscapeSequence((selected as Partition).name || "Unknown")}` // Otherwise, assume Partition
						: "Details"
				}
				objectToDisplay={selected}
				open={showPreview}
				onClose={() => {
					setSelected(undefined);
					setShowPreview(false);
				}}
			/>
			<br />
			<Stack direction="row" justifyContent="flex-start" sx={{ pl: 2, mb: 1 }}>
				<FormControlLabel
					control={
						<Switch
							checked={hideSystemPartitions}
							onChange={(e) => setHideSystemPartitions(e.target.checked)}
							name="hideSystemPartitions"
							size="small"
						/>
					}
					label={
						<Typography variant="body2">Hide system partitions</Typography>
					}
				/>
			</Stack>
			<List dense={true}>
				<Divider />
				{/* Iterate over disks */}
				{disks &&
					disks.map((disk, diskIdx) => {
						const diskIdentifier = disk.id || `disk-${diskIdx}`;

						const filteredPartitions =
							disk.partitions?.filter(
								(partition) =>
									!(
										hideSystemPartitions &&
										(partition.system &&
											(partition.name?.startsWith("hassos-") ||
												(partition.host_mount_point_data &&
													partition.host_mount_point_data.length >
													0)))
									),
							) || [];

						if (!isDiskRendered(disk, hideSystemPartitions)) {
							return null; // Don't render this disk if all its partitions are hidden by the toggle
						}

						return (
							<Fragment key={diskIdentifier}>
								<Accordion
									expanded={expandedAccordion === diskIdentifier}
									onChange={handleAccordionChange(diskIdentifier)}
									sx={{
										boxShadow: "none",
										"&:before": { display: "none" },
										"&.Mui-expanded": { margin: "0" },
										backgroundColor: "transparent",
									}}
									disableGutters
								>
									<AccordionSummary
										expandIcon={<ExpandMore />}
										aria-controls={`${diskIdentifier}-content`}
										id={`${diskIdentifier}-header`}
										sx={{
											"& .MuiAccordionSummary-content": {
												alignItems: "flex-start",
												width: "calc(100% - 32px)",
											}, // Adjust width for expand icon
											py: 0, // Remove padding if ListItem handles it
										}}
									>
										<ListItem
											sx={{ pl: 0, py: 1, width: "100%" }}
											disablePadding
											component="div"
										>
											<ListItemAvatar sx={{ pt: 1, cursor: "pointer" }}>
												<Avatar
													onClick={(e) => {
														e.stopPropagation();
														setSelected(disk);
														setShowPreview(true);
													}}
												>
													{renderDiskIcon(disk)}
												</Avatar>
											</ListItemAvatar>
											<ListItemText
												sx={{ cursor: "pointer", overflowWrap: "break-word" }}
												primary={`Disk: ${disk.model?.toUpperCase() || `Disk ${diskIdx + 1}`}`}
												disableTypography
												secondary={
													<Stack spacing={0.5} sx={{ pt: 0.5 }}>
														<Typography variant="caption" component="div">
															{`${disk.partitions?.length || 0} partition(s)`}
														</Typography>
														<Stack
															direction="row"
															spacing={1}
															flexWrap="wrap"
															alignItems="center"
															sx={{ display: { xs: "none", sm: "flex" } }}
														>
															{disk.size != null && (
																<Chip
																	label={`Size: ${filesize(disk.size, { round: 1 })}`}
																	size="small"
																	variant="outlined"
																/>
															)}
															{disk.vendor && (
																<Chip
																	label={`Vendor: ${disk.vendor}`}
																	size="small"
																	variant="outlined"
																/>
															)}
															{disk.serial && (
																<Chip
																	label={`Serial: ${disk.serial}`}
																	size="small"
																	variant="outlined"
																/>
															)}
															{disk.connection_bus && (
																<Chip
																	label={`Bus: ${disk.connection_bus}`}
																	size="small"
																	variant="outlined"
																/>
															)}
															{disk.revision && (
																<Chip
																	label={`Rev: ${disk.revision}`}
																	size="small"
																	variant="outlined"
																/>
															)}
														</Stack>
													</Stack>
												}
											/>
										</ListItem>
									</AccordionSummary>
									<AccordionDetails sx={{ p: 0 }}>
										{/* Collapsible Section for Partitions */}
										{disk.partitions && disk.partitions.length > 0 && (
											<List
												component="div"
												disablePadding
												dense={true}
												sx={{ pl: 4 }}
											>
												{filteredPartitions.map((partition, partIdx) => {
													const partitionIdentifier =
														partition.id || `${diskIdentifier}-part-${partIdx}`;
													const isMounted =
														partition.mount_point_data &&
														partition.mount_point_data.length > 0 &&
														partition.mount_point_data.some(
															(mpd) => mpd.is_mounted,
														);

													const partitionNameDecoded = decodeEscapeSequence(
														partition.name || "Unnamed Partition",
													);

													return (
														<Fragment key={partitionIdentifier}>
															<ListItemButton
																sx={{ pl: 1, alignItems: "flex-start" }} // Align items top
															>
																<ListItem
																	disablePadding
																	secondaryAction={
																		<PartitionActions
																			partition={partition}
																			read_only={read_only}
																			onToggleAutomount={
																				handleToggleAutomount
																			}
																			onMount={(p) => {
																				setSelected(p);
																				setShowMount(true);
																			}}
																			onViewSettings={(p) => {
																				setSelected(p);
																				setShowMountSettings(true);
																			}}
																			onUnmount={onSubmitUmountVolume}
																			onCreateShare={handleCreateShare}
																			onGoToShare={handleGoToShare}
																		/>
																	}
																>
																	<ListItemAvatar
																		sx={{ minWidth: "auto", pr: 1.5, pt: 0.5 }}
																	>
																		{" "}
																		{/* Align avatar */}
																		<Avatar
																			sx={{ width: 32, height: 32 }}
																			onClick={() => {
																				setSelected(partition);
																				setShowPreview(true);
																			}}
																		>
																			{renderPartitionIcon(partition)}
																		</Avatar>
																	</ListItemAvatar>
																	<ListItemText
																		primary={partitionNameDecoded}
																		disableTypography
																		secondary={
																			<Stack
																				spacing={1}
																				direction="row"
																				flexWrap="wrap"
																				alignItems="center"
																				sx={{
																					pt: 0.5,
																					display: {
																						xs: "none",
																						sm: "flex",
																					},
																			}}
																			>
																				{partition.size != null && (
																					<Chip
																						label={`Size: ${filesize(partition.size, { round: 0 })}`}
																						size="small"
																						variant="outlined"
																					/>
																			)}
																			{partition.mount_point_data?.[0]
																				?.fstype && (
																					<Chip
																						label={`Type: ${partition.mount_point_data[0].fstype}`}
																						size="small"
																						variant="outlined"
																					/>
																			)}
																			{isMounted && (
																					<Chip
																						label={`Mount: ${partition.mount_point_data?.map((mpd) => mpd.path).join(" ")}`}
																						size="small"
																						variant="outlined"
																					/>
																			)}
																			{partition.host_mount_point_data &&
																				partition.host_mount_point_data
																					.length > 0 && (
																					<Chip
																						label={`Host: ${partition.host_mount_point_data.map((mpd) => mpd.path).join(" ")}`}
																						size="small"
																						variant="outlined"
																					/>
																			)}
																			{partition.id && (
																					<Chip
																						label={`UUID: ${partition.id}`}
																						size="small"
																						variant="outlined"
																					/>
																			)}
																			{partition.device && (
																					<Chip
																						label={`Dev: ${partition.device}`}
																						size="small"
																						variant="outlined"
																					/>
																			)}
																			</Stack>
																		}
																	/>
																</ListItem>
															</ListItemButton>
															{partIdx < filteredPartitions.length - 1 && (
																<Divider
																	variant="inset"
																	component="li"
																	sx={{ ml: 4 }}
																/>
															)}
														</Fragment>
													);
												})}
												{expandedAccordion === diskIdentifier &&
													disk.partitions &&
													disk.partitions.length > 0 &&
													filteredPartitions.length === 0 &&
													hideSystemPartitions && (
														<ListItem dense sx={{ pl: 1 }}>
															<ListItemText
																secondary="System partitions are hidden."
																slotProps={{
																	secondary: {
																		variant: "caption",
																		fontStyle: "italic",
																	},
																}}
															/>
														</ListItem>
													)}
											</List>
										)}
									</AccordionDetails>
								</Accordion>
								<Divider /> {/* This Divider separates the Accordions */}
							</Fragment>
						);
					})}
			</List>
		</>
	);
}
