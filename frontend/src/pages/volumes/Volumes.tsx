import {
	Box,
	FormControlLabel,
	Grid,
	Paper,
	Stack,
	Switch,
	Typography,
} from "@mui/material";
import { useConfirm } from "material-ui-confirm";
import { useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router";
import { toast } from "react-toastify";
import { PreviewDialog } from "../../components/PreviewDialog";
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
import { VolumesTreeView, VolumeDetailsPanel, VolumeMountDialog } from "./components";
import { decodeEscapeSequence } from "./utils";
import { TourEvents, TourEventTypes } from "../../utils/TourEvents";
import { useGetServerEventsQuery } from "../../store/sseApi";

export function Volumes({ initialDisks }: { initialDisks?: Disk[] } = {}) {
	const { data: evdata, isLoading: is_evLoading } = useGetServerEventsQuery();
	const [showPreview, setShowPreview] = useState<boolean>(false);
	const [showMount, setShowMount] = useState<boolean>(false);
	const location = useLocation();

	const navigate = useNavigate();
	const [hideSystemPartitions, setHideSystemPartitions] = useState<boolean>(localStorage.getItem("volumes.hideSystemPartitions") === "true"); // Default to hide system partitions
	const volumeHook = useVolume();
	const disks = initialDisks ?? volumeHook.disks;
	const isLoading = initialDisks ? false : volumeHook.isLoading;
	const error = initialDisks ? null : volumeHook.error;
	const [selectedDisk, setSelectedDisk] = useState<Disk | undefined>(undefined);
	const [selectedPartition, setSelectedPartition] = useState<Partition | undefined>(undefined);
	const [selectedPartitionId, setSelectedPartitionId] = useState<string | undefined>(() => localStorage.getItem("volumes.selectedPartitionId") || undefined);
	const [expandedDisks, setExpandedDisks] = useState<string[]>(() => {
		try {
			const savedExpanded = localStorage.getItem("volumes.expandedDisks");
			if (savedExpanded) {
				const parsed = JSON.parse(savedExpanded);
				if (Array.isArray(parsed)) return parsed as string[];
			}
		} catch { }
		return [];
	});
	const confirm = useConfirm();
	const [mountVolume, _mountVolumeResult] = usePostApiVolumeByMountPathHashMountMutation();
	const [umountVolume, _umountVolumeResult] = useDeleteApiVolumeByMountPathHashMountMutation();
	const [patchMountSettings] = usePatchApiVolumeByMountPathHashSettingsMutation();

	// Handle partition selection
	const handlePartitionSelect = (disk: Disk, partition: Partition) => {
		setSelectedDisk(disk);
		setSelectedPartition(partition);
		const diskIdx = disks?.indexOf(disk) || 0;
		const partIdx = disk.partitions?.indexOf(partition) || 0;
		const partitionId = partition.id || `${disk.id || `disk-${diskIdx}`}-part-${partIdx}`;
		setSelectedPartitionId(partitionId);
		// Ensure the containing disk is expanded and persisted
		const diskIdentifier = disk.id || `disk-${diskIdx}`;
		setExpandedDisks((prev) => {
			if (prev.includes(diskIdentifier)) return prev;
			return [...prev, diskIdentifier];
		});
	};

	// Persist selection and expanded disks to localStorage
	useEffect(() => {
		try {
			if (selectedPartitionId) {
				localStorage.setItem("volumes.selectedPartitionId", selectedPartitionId);
			} else {
				localStorage.removeItem("volumes.selectedPartitionId");
			}
		} catch (err) {
			console.warn("Could not persist selectedPartitionId", err);
		}
	}, [selectedPartitionId]);

	useEffect(() => {
		try {
			if (expandedDisks.length > 0) {
				localStorage.setItem("volumes.expandedDisks", JSON.stringify(expandedDisks));
			} else {
				localStorage.removeItem("volumes.expandedDisks");
			}
		} catch (err) {
			console.warn("Could not persist expandedDisks", err);
		}
	}, [expandedDisks]);

	useEffect(() => {
		try {
			localStorage.setItem("volumes.hideSystemPartitions", hideSystemPartitions ? "true" : "false");
		} catch (err) {
			console.warn("Could not persist hideSystemPartitions", err);
		}
	}, [hideSystemPartitions]);

	// Effect to handle navigation state for opening mount settings for a specific volume
	useEffect(() => {
		const state = location.state as LocationState | undefined;
		const mountPathHashFromState = state?.mountPathHashToView;

		if (
			mountPathHashFromState &&
			Array.isArray(disks) &&
			disks.length > 0
		) {
			let foundPartition: Partition | undefined;
			let foundDisk: Disk | undefined;

			for (const disk of disks) {
				if (disk.partitions) {
					for (const partition of disk.partitions) {
						if (
							partition.mount_point_data?.some(
								(mpd) => mpd.path_hash === mountPathHashFromState,
							)
						) {
							foundPartition = partition;
							foundDisk = disk;
							break;
						}
					}
				}
				if (foundPartition) break;
			}

			if (foundPartition && foundDisk) {
				handlePartitionSelect(foundDisk, foundPartition);
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
		location.pathname,
	]);

	// When disks data is available and there's a selectedPartitionId (restored or new), find and select it so details show
	useEffect(() => {
		if (!disks || disks.length === 0) return;
		if (!selectedPartitionId) return;

		// Try to locate the partition corresponding to selectedPartitionId
		for (const disk of disks) {
			if (!disk.partitions) continue;
			for (const partition of disk.partitions) {
				const diskIdx = disks.indexOf(disk);
				const partIdx = disk.partitions.indexOf(partition);
				const partitionIdentifier = partition.id || `${disk.id || `disk-${diskIdx}`}-part-${partIdx}`;
				if (partitionIdentifier === selectedPartitionId) {
					setSelectedDisk(disk);
					setSelectedPartition(partition);
					return;
				}
			}
		}

		// If not found, clear selection
		setSelectedPartition(undefined);
		setSelectedDisk(undefined);
		setSelectedPartitionId(undefined);
	}, [disks, selectedPartitionId]);

	function onSubmitMountVolume(data?: MountPointData) {
		console.trace("Mount Request Data:", data);

		if (!selectedPartition || !data || !data.path) {
			toast.error("Cannot mount: Invalid selection or missing data.");
			console.error("Mount validation failed:", {
				selectedPartition,
				data,
			});
			return;
		}

		// Ensure device is included in submitData if required by API
		const submitData: MountPointData = {
			...data,
			device_id: selectedPartition.id,
		};

		mountVolume({
			mountPathHash: data.path_hash || "",
			mountPointData: submitData,
		})
			.unwrap()
			.then((res) => {
				toast.info(
					`Volume ${(res as MountPointData).path || selectedPartition.name} mounted successfully.`,
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
				setSelectedPartition(undefined); // Clear selection after successful mount
				setSelectedDisk(undefined);
				setSelectedPartitionId(undefined);
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
			description: `Do you really want to ${force ? "forcefully " : ""}unmount the Volume ${displayName} (${partition.legacy_device_name}) mounted at ${mountData.path}?`,
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
						if (selectedPartition?.id === partition.id) {
							setSelectedPartition(undefined);
							setSelectedDisk(undefined);
							setSelectedPartitionId(undefined);
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
		if (evdata?.hello?.read_only) return;

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

	useEffect(() => {
		const handleVolumesStep3 = () => {
			// find first disk and expand it - this will be handled by the tree view default expansion
		};

		const handleVolumesStep4 = () => {
			// find first disk and expand it - this will be handled by the tree view default expansion
		};

		const handleVolumesStep5 = () => {
			// find first disk and expand it - this will be handled by the tree view default expansion
		};

		TourEvents.on(TourEventTypes.VOLUMES_STEP_3, handleVolumesStep3);
		TourEvents.on(TourEventTypes.VOLUMES_STEP_4, handleVolumesStep4);
		TourEvents.on(TourEventTypes.VOLUMES_STEP_5, handleVolumesStep5);

		return () => {
			TourEvents.off(TourEventTypes.VOLUMES_STEP_3, handleVolumesStep3);
			TourEvents.off(TourEventTypes.VOLUMES_STEP_4, handleVolumesStep4);
			TourEvents.off(TourEventTypes.VOLUMES_STEP_5, handleVolumesStep5);
		};
	}, []);

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

	// Get the related share for the selected partition
	const selectedShare = selectedPartition?.mount_point_data?.[0]?.shares?.[0];

	return (
		<>
			<VolumeMountDialog
				objectToEdit={selectedPartition}
				open={showMount}
				readOnlyView={false}
				onClose={(data) => {
					if (showMount) {
						// If it was open for mounting
						if (data) {
							onSubmitMountVolume(data);
						} else {
							// Cancelled mount dialog or no data returned
							setSelectedPartition(undefined);
							setSelectedDisk(undefined);
							setSelectedPartitionId(undefined);
							setShowMount(false);
						}
					}
				}}
			/>
			{/* PreviewDialog can show details for both disks and partitions */}
			<PreviewDialog
				title={
					selectedDisk && selectedPartition
						? `Partition: ${decodeEscapeSequence(selectedPartition.name || selectedPartition.id || "Unknown")}`
						: selectedDisk
							? `Disk: ${selectedDisk.model}`
							: "Details"
				}
				objectToDisplay={selectedPartition || selectedDisk}
				open={showPreview}
				onClose={() => {
					setSelectedPartition(undefined);
					setSelectedDisk(undefined);
					setSelectedPartitionId(undefined);
					setShowPreview(false);
				}}
			/>

			{/* Main Layout Grid */}
			<Grid container spacing={2} sx={{ minHeight: "calc(100vh - 200px)" }}>
				{/* Left Panel - Tree View */}
				<Grid size={{ xs: 12, md: 4, lg: 3 }}>
					<Paper sx={{ height: "100%", p: 1 }} data-tutor={`reactour__tab${TabIDs.VOLUMES}__step3`}>
						<Stack
							direction="row"
							justifyContent="space-between"
							alignItems="center"
							sx={{ mb: 2, px: 2 }}
						>
							<Typography variant="h6">
								Volumes
							</Typography>
						</Stack>

						<Stack direction="row" justifyContent="flex-start" sx={{ pl: 2, mb: 1 }} data-tutor={`reactour__tab${TabIDs.VOLUMES}__step2`}>
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

						{isLoading ? (
							<Typography>Loading volumes...</Typography>
						) : error ? (
							<Typography color="error">
								Error loading volume information. Please try again later.
							</Typography>
						) : (
							<VolumesTreeView
								disks={disks}
								selectedPartitionId={selectedPartitionId}
								expandedItems={expandedDisks}
								onExpandedItemsChange={setExpandedDisks}
								hideSystemPartitions={hideSystemPartitions}
								onPartitionSelect={handlePartitionSelect}
								onToggleAutomount={handleToggleAutomount}
								onMount={(partition) => {
									setSelectedPartition(partition);
									setShowMount(true);
								}}
								onUnmount={onSubmitUmountVolume}
								onCreateShare={handleCreateShare}
								onGoToShare={handleGoToShare}
								protectedMode={evdata?.hello?.protected_mode === true}
								readOnly={evdata?.hello?.read_only === true}
							/>
						)}
					</Paper>
				</Grid>

				{/* Right Panel - Details */}
				<Grid size={{ xs: 12, md: 8, lg: 9 }}>
					<Paper sx={{ height: "100%", overflow: "hidden" }}>
						<VolumeDetailsPanel
							disk={selectedDisk}
							partition={selectedPartition}
							share={selectedShare}
						/>
					</Paper>
				</Grid>
			</Grid>
		</>
	);
}
