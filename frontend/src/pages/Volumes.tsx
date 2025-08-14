import {
	faPlug,
	faPlugCircleMinus,
	faPlugCircleXmark,
} from "@fortawesome/free-solid-svg-icons";
import { sha1 } from "js-sha1";
import AddIcon from "@mui/icons-material/Add";
// ... other icon imports ...
import ComputerIcon from "@mui/icons-material/Computer";
import CreditScoreIcon from "@mui/icons-material/CreditScore";
// Add EjectIcon to your imports
import EjectIcon from "@mui/icons-material/Eject";
import ExpandMore from "@mui/icons-material/ExpandMore"; // Import expand icons
import MoreVertIcon from "@mui/icons-material/MoreVert";
import SdStorageIcon from "@mui/icons-material/SdStorage";
import SettingsSuggestIcon from "@mui/icons-material/SettingsSuggest";
import ShareIcon from "@mui/icons-material/Share";
import StorageIcon from "@mui/icons-material/Storage";
import UpdateIcon from "@mui/icons-material/Update";
import UpdateDisabledIcon from "@mui/icons-material/UpdateDisabled";
import UsbIcon from "@mui/icons-material/Usb";
import VisibilityIcon from "@mui/icons-material/Visibility";
import {
	Accordion,
	AccordionDetails,
	AccordionSummary,
	Avatar,
	Button,
	Chip,
	Dialog,
	DialogActions,
	DialogContent,
	DialogContentText,
	DialogTitle,
	Divider,
	FormControlLabel,
	Grid,
	IconButton,
	ListItem,
	ListItemAvatar,
	ListItemButton,
	ListItemIcon,
	ListItemText,
	Menu,
	MenuItem,
	Stack,
	Switch,
	Tooltip,
	Typography,
	useMediaQuery,
	useTheme,
} from "@mui/material";
import List from "@mui/material/List";
import { filesize } from "filesize";
import { useConfirm } from "material-ui-confirm";
import { Fragment, useCallback, useEffect, useRef, useState } from "react";
import {
	AutocompleteElement,
	CheckboxElement,
	TextFieldElement,
	useFieldArray,
	useForm,
} from "react-hook-form-mui"; // Import TextFieldElement
import { useLocation, useNavigate } from "react-router";
import { toast } from "react-toastify";
import { FontAwesomeSvgIcon } from "../components/FontAwesomeSvgIcon";
import { PreviewDialog } from "../components/PreviewDialog";
import { useReadOnly } from "../hooks/readonlyHook";
import { useVolume } from "../hooks/volumeHook";
import { type LocationState, TabIDs } from "../store/locationState";
import {
	type Disk,
	type FilesystemType,
	type MountFlag,
	type MountPointData,
	type Partition,
	Type,
	useDeleteApiVolumeByMountPathHashMountMutation,
	useGetApiFilesystemsQuery,
	usePatchApiVolumeByMountPathHashSettingsMutation,
	usePostApiVolumeByMountPathHashMountMutation,
	usePostApiVolumeDiskByDiskIdEjectMutation,
} from "../store/sratApi";

// --- Helper functions (decodeEscapeSequence, onSubmitMountVolume, etc.) remain the same ---
function decodeEscapeSequence(source: string) {
	// Basic check to avoid errors if source is not a string
	if (typeof source !== "string") return "";
	return source.replace(/\\x([0-9A-Fa-f]{2})/g, (_match, group1) => {
		// Ensure group1 is treated as a string before parseInt
		return String.fromCharCode(parseInt(String(group1), 16));
	});
}

// Helper function to generate SHA-1 hash with fallback
async function generateSHA1Hash(input: string): Promise<string> {
	// Try to use crypto.subtle if available
	if (typeof crypto !== "undefined" && crypto.subtle) {
		try {
			const hashBuffer = await crypto.subtle.digest('SHA-1', new TextEncoder().encode(input));
			return Array.from(new Uint8Array(hashBuffer))
				.map(b => b.toString(16).padStart(2, '0'))
				.join('');
		} catch (error) {
			console.warn('crypto.subtle failed, falling back to js-sha1:', error);
		}
	}

	// Fallback to js-sha1
	return sha1(input);
}

interface PartitionActionsProps {
	partition: Partition;
	read_only: boolean;
	onToggleAutomount: (partition: Partition) => void;
	onMount: (partition: Partition) => void;
	onViewSettings: (partition: Partition) => void;
	onUnmount: (partition: Partition, force: boolean) => void;
	onCreateShare: (partition: Partition) => void;
	onGoToShare: (partition: Partition) => void;
}

function PartitionActions({
	partition,
	read_only,
	onToggleAutomount,
	onMount,
	onViewSettings,
	onUnmount,
	onCreateShare,
	onGoToShare,
}: PartitionActionsProps) {
	const theme = useTheme();
	const isSmallScreen = useMediaQuery(theme.breakpoints.down("sm"));
	const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);

	const handleMenuOpen = (event: React.MouseEvent<HTMLElement>) => {
		event.stopPropagation();
		setAnchorEl(event.currentTarget);
	};

	const handleMenuClose = (
		e?: React.MouseEvent<HTMLElement> | {},
		_reason?: "backdropClick" | "escapeKeyDown",
	) => {
		(e as React.MouseEvent<HTMLElement>)?.stopPropagation();
		setAnchorEl(null);
	};

	const isMounted =
		partition.mount_point_data &&
		partition.mount_point_data.length > 0 &&
		partition.mount_point_data.some((mpd) => mpd.is_mounted);
	const hasShares =
		partition.mount_point_data &&
		partition.mount_point_data.length > 0 &&
		partition.mount_point_data.some((mpd) => {
			return (
				mpd.shares &&
				mpd.shares.length > 0 //&&
				//mpd.shares.some((share) => !share.disabled)
			);
		});
	const firstMountPath = partition.mount_point_data?.[0]?.path;
	const showShareActions = isMounted && firstMountPath?.startsWith("/mnt/");

	if (
		read_only ||
		//partition.system ||
		partition.name?.startsWith("hassos-") ||
		(partition.host_mount_point_data &&
			partition.host_mount_point_data.length > 0)
	) {
		return null;
	}

	const actionItems = [];

	// Automount Toggle Button
	if (!hasShares && partition.mount_point_data?.[0]?.path) {
		if (partition.mount_point_data?.[0]?.is_to_mount_at_startup) {
			actionItems.push({
				key: "disable-automount",
				title: "Disable mount at startup",
				icon: <UpdateDisabledIcon />,
				onClick: () => onToggleAutomount(partition),
			});
		} else {
			actionItems.push({
				key: "enable-automount",
				title: "Enable mount at startup",
				icon: <UpdateIcon />,
				onClick: () => onToggleAutomount(partition),
			});
		}
	}

	// Mount
	if (!isMounted) {
		actionItems.push({
			key: "mount",
			title: "Mount Partition",
			icon: <FontAwesomeSvgIcon icon={faPlug} />,
			onClick: () => onMount(partition),
		});
	}

	if (isMounted) {
		actionItems.push({
			key: "view-settings",
			title: "View Mount Settings",
			icon: <VisibilityIcon fontSize="small" />,
			onClick: () => onViewSettings(partition),
		});
		if (!hasShares) {
			actionItems.push({
				key: "unmount",
				title: "Unmount Partition",
				icon: <FontAwesomeSvgIcon icon={faPlugCircleMinus} />,
				onClick: () => onUnmount(partition, false),
			});
		}
		actionItems.push({
			key: "force-unmount",
			title: "Force Unmount Partition",
			icon: <FontAwesomeSvgIcon icon={faPlugCircleXmark} />,
			onClick: () => onUnmount(partition, true),
		});
		if (showShareActions) {
			if (!hasShares) {
				actionItems.push({
					key: "create-share",
					title: "Create Share",
					icon: <AddIcon fontSize="small" />,
					onClick: () => onCreateShare(partition),
				});
			} else {
				actionItems.push({
					key: "go-to-share",
					title: "Go to Share",
					icon: <ShareIcon fontSize="small" />,
					onClick: () => onGoToShare(partition),
				});
			}
		}
	}

	if (isSmallScreen) {
		return (
			<>
				<IconButton
					aria-label="more actions"
					aria-controls="partition-actions-menu"
					aria-haspopup="true"
					onClick={handleMenuOpen}
					edge="end"
					size="small"
				>
					<MoreVertIcon />
				</IconButton>
				<Menu
					id="partition-actions-menu"
					anchorEl={anchorEl}
					open={Boolean(anchorEl)}
					onClose={handleMenuClose}
					onClick={(e) => e.stopPropagation()}
				>
					{actionItems.map((action) => (
						<MenuItem
							key={action.key}
							onClick={(e) => {
								e.stopPropagation();
								action.onClick();
								handleMenuClose();
							}}
						>
							<ListItemIcon>{action.icon}</ListItemIcon>
							<ListItemText>{action.title}</ListItemText>
						</MenuItem>
					))}
				</Menu>
			</>
		);
	}

	return (
		<Stack direction="row" spacing={0} alignItems="center" sx={{ pr: 1 }}>
			{actionItems.map((action) => (
				<Tooltip title={action.title} key={action.key}>
					<IconButton
						onClick={(e) => {
							e.stopPropagation();
							action.onClick();
						}}
						edge="end"
						aria-label={action.title.toLowerCase()}
						size="small"
					>
						{action.icon}
					</IconButton>
				</Tooltip>
			))}
		</Stack>
	);
}

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
	const [ejectDiskMutation, _ejectDiskResult] =
		usePostApiVolumeDiskByDiskIdEjectMutation();
	const [patchMountSettings] = usePatchApiVolumeByMountPathHashSettingsMutation();

	const localStorageKey = "srat_volumes_expanded_accordion";

	const [expandedAccordion, setExpandedAccordion] = useState<string | false>(
		() => {
			const savedAccordionId = localStorage.getItem(localStorageKey);
			return savedAccordionId || false;
		},
	);
	const initialAutoOpenDone = useRef(false);
	const prevDisksRef = useRef<Disk[] | undefined>(undefined);
	const prevHideSystemPartitionsRef = useRef<boolean | undefined>(undefined);

	// Helper to determine if a disk would be rendered based on current filters
	const isDiskRendered = useCallback(
		(disk: Disk, hideSystem: boolean): boolean => {
			const filteredPartitions =
				disk.partitions?.filter(
					(p) =>
						!(
							hideSystem &&
							(p.system && (
								p.name?.startsWith("hassos-") ||
								(p.host_mount_point_data && p.host_mount_point_data.length > 0)))
						),
				) || [];
			const hasActualPartitions = disk.partitions && disk.partitions.length > 0;
			const allPartitionsAreHiddenByToggle =
				hasActualPartitions && filteredPartitions.length === 0 && hideSystem;
			return !allPartitionsAreHiddenByToggle;
		},
		[],
	);

	useEffect(() => {
		if (
			prevDisksRef.current !== disks ||
			prevHideSystemPartitionsRef.current !== hideSystemPartitions
		) {
			initialAutoOpenDone.current = false;
			prevDisksRef.current = disks;
			prevHideSystemPartitionsRef.current = hideSystemPartitions;
		}

		if (initialAutoOpenDone.current) {
			if (!disks || disks.length === 0) {
				if (expandedAccordion !== false) setExpandedAccordion(false);
			} else if (typeof expandedAccordion === "string") {
				const diskIndex = disks.findIndex(
					(d, idx) => (d.id || `disk-${idx}`) === expandedAccordion,
				);
				if (
					diskIndex === -1 ||
					!isDiskRendered(disks[diskIndex], hideSystemPartitions)
				) {
					setExpandedAccordion(false);
				}
			}
			return;
		}

		if (isLoading || !Array.isArray(disks)) {
			return;
		}

		if (disks.length === 0) {
			if (expandedAccordion !== false) setExpandedAccordion(false);
			initialAutoOpenDone.current = true;
			return;
		}

		let isValidLocalStorageValue = false;
		if (typeof expandedAccordion === "string") {
			const diskIndex = disks.findIndex(
				(d, idx) => (d.id || `disk-${idx}`) === expandedAccordion,
			);
			if (diskIndex !== -1) {
				isValidLocalStorageValue = isDiskRendered(
					disks[diskIndex],
					hideSystemPartitions,
				);
			}
		}

		if (!isValidLocalStorageValue) {
			let firstRenderedDiskIdentifier: string | null = null;
			for (let i = 0; i < disks.length; i++) {
				if (isDiskRendered(disks[i], hideSystemPartitions)) {
					firstRenderedDiskIdentifier = disks[i].id || `disk-${i}`;
					break;
				}
			}
			setExpandedAccordion(firstRenderedDiskIdentifier || false);
		}
		initialAutoOpenDone.current = true;
	}, [
		disks,
		isLoading,
		hideSystemPartitions,
		expandedAccordion,
		isDiskRendered,
	]);

	useEffect(() => {
		if (!initialAutoOpenDone.current) {
			return;
		}

		if (expandedAccordion === false) {
			localStorage.removeItem(localStorageKey);
		} else if (typeof expandedAccordion === "string") {
			let isValidToSave = false;
			if (Array.isArray(disks)) {
				const diskIndex = disks.findIndex(
					(d, idx) => (d.id || `disk-${idx}`) === expandedAccordion,
				);
				if (diskIndex !== -1) {
					isValidToSave = isDiskRendered(
						disks[diskIndex],
						hideSystemPartitions,
					);
				}
			}

			if (isValidToSave) {
				localStorage.setItem(localStorageKey, expandedAccordion);
			} else {
				localStorage.removeItem(localStorageKey);
			}
		}
	}, [expandedAccordion, disks, hideSystemPartitions, isDiskRendered]);

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
	]);

	const handleAccordionChange =
		(panel: string) => (_event: React.SyntheticEvent, isExpanded: boolean) => {
			setExpandedAccordion(isExpanded ? panel : false);
			if (!initialAutoOpenDone.current) {
				initialAutoOpenDone.current = true;
			}
		};

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

	/*
	function _onSubmitEjectDisk(disk: Disk) {
		if (read_only || !disk || !disk.removable) {
			toast.error("Disk is not ejectable or action is not permitted.");
			return;
		}

		const diskName = disk.model || disk.id || "this disk";

		const sharesExistOnDisk =
			disk.partitions?.some((partition) =>
				partition.mount_point_data?.some((mpd) =>
					mpd.shares?.some((share) => !share.disabled),
				),
			) || false;

		const description = `Do you really want to eject the disk ${diskName}? This will unmount all its partitions. ${sharesExistOnDisk ? "Any configured shares on this disk will be disabled." : ""}`;
		const acknowledgement = `I understand that ejecting the disk will make it inaccessible ${sharesExistOnDisk ? "and disable related shares" : ""}.`;

		confirm({
			title: `Eject ${diskName}?`,
			description: description,
			confirmationText: "Eject",
			cancellationText: "Cancel",
			confirmationButtonProps: { color: "error" },
			acknowledgement: acknowledgement,
		}).then(({ reason }) => {
			if (reason === "confirm" && disk.id) {
				ejectDiskMutation({ diskId: disk.id })
					.unwrap()
					.then(() => toast.info(`Disk ${diskName} ejected successfully.`))
					.catch((err) => {
						console.error("Eject Error:", err);
						toast.error(
							`Error ejecting disk ${diskName}: ${err.data?.detail || err.data?.message || err.message || "Unknown error"}`,
						);
					});
			}
		});
	}
	*/

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
					selected &&
						!(selected as Disk).partitions
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
				{disks.map((disk, diskIdx) => {
					const diskIdentifier = disk.id || `disk-${diskIdx}`;

					const filteredPartitions =
						disk.partitions?.filter(
							(partition) =>
								!(
									hideSystemPartitions &&
									(partition.system &&
										(partition.name?.startsWith("hassos-") ||
											(partition.host_mount_point_data &&
												partition.host_mount_point_data.length > 0)))
								),
						) || [];

					// Determine if the disk itself should be hidden
					// A disk is hidden if:
					// 1. The "hideSystemPartitions" toggle is on.
					// 2. The disk actually has partitions.
					// 3. All of its partitions are system partitions (or hassos-) (meaning filteredPartitions would be empty).
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
										<ListItemAvatar
											sx={{ pt: 1, cursor: "pointer" }}
										>
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
										{/*
                                        <Stack direction="row" spacing={0.5} alignItems="center" sx={{ ml: 1, mt: 0.5, alignSelf: 'center' }}>
                                            {!read_only && disk.removable && (
                                                <Tooltip title={`Eject disk ${disk.model || disk.id}`}>
                                                    <IconButton
                                                        size="small"
                                                        onClick={(e) => { e.stopPropagation(); onSubmitEjectDisk(disk); }}
                                                        aria-label="Eject disk"
                                                    >
                                                        <EjectIcon />
                                                    </IconButton>
                                                </Tooltip>
                                            )}
                                        </Stack>
                                        */}
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
												const _hasShares =
													partition.mount_point_data &&
													partition.mount_point_data.length > 0 &&
													partition.mount_point_data.some((mpd) => {
														return (
															mpd.shares &&
															mpd.shares.length > 0 // &&
															//	mpd.shares.some((share) => !share.disabled)
														);
													});

												const firstMountPath =
													partition.mount_point_data?.[0]?.path;
												const _showShareActions =
													isMounted && firstMountPath?.startsWith("/mnt/");
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
																		onToggleAutomount={handleToggleAutomount}
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
																				display: { xs: "none", sm: "flex" },
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
																				partition.host_mount_point_data.length >
																				0 && (
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

interface xMountPointData extends MountPointData {
	custom_flags_values: MountFlag[]; // Array of custom flags (enum) for the TextField
}

interface VolumeMountDialogProps {
	open: boolean;
	onClose: (data?: MountPointData) => void;
	objectToEdit?: Partition;
	readOnlyView?: boolean;
}

function VolumeMountDialog(props: VolumeMountDialogProps) {
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
	const { fields, append, prepend, remove, swap, move, insert, replace } =
		useFieldArray({
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
									{" "}
									{/* Corrected Grid usage & responsiveness */}
									<TextFieldElement
										size="small"
										name="path"
										label="Mount Path"
										control={control}
										required
										fullWidth
										disabled={props.readOnlyView}
										slotProps={{ inputLabel: { shrink: true } }} // Ensure label is always shrunk
										helperText="Path must start with /mnt/"
									/>
								</Grid>
								<Grid size={{ xs: 12, sm: 6 }}>
									{" "}
									{/* FS Type & responsiveness */}
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
											helperText: fsError
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
									{" "}
									{/* FS Flags & responsiveness */}
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
													getOptionKey: (option) => (option as MountFlag).name,
													getOptionLabel: (option) =>
														(option as MountFlag).name,
													renderOption: (props, option) => (
														<li  {...props} key={props.key}>
															<Tooltip title={option.description || ""}>
																<span>
																	{option.name}{" "}
																	{option.needsValue ? (
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
									{" "}
									{/* FS CustomFlags & responsiveness */}
									{!fsLoading &&
										(
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
													getOptionKey: (option) => (option as MountFlag).name,
													getOptionLabel: (option) =>
														(option as MountFlag).name, // Ensure label is just the name
													renderOption: (props, option) => (
														<li {...props}>
															<Tooltip title={option.description || ""}>
																<span>
																	{option.name}{" "}
																	{option.needsValue ? (
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
													renderValue: (values, getItemProps) =>
														values.map((option, index) => {
															const { key, ...itemProps } = getItemProps({
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
																	label={(option as MountFlag)?.name || "error"}
																	size="small"
																	{...itemProps}
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
																		(fv) => fv.name === selectedFlag.name,
																	)
																) {
																	newFieldValues.push({
																		...selectedFlag,
																		value: selectedFlag.value || "",
																	}); // Use existing value or empty
																}
															});
															replace(newFieldValues);
															setValue("custom_flags", value as MountFlag[], {
																shouldDirty: true,
															}); // also update the custom_flags themselves
														},
												}}
												textFieldProps={{
													disabled: props.readOnlyView,
													InputLabelProps: { shrink: true },
												}}
											/>
										)}
								</Grid>
								{fields.map((field, index) => (
									<Grid size={{ xs: 12, sm: 6 }} key={field.id}>
										{" "}
										{/* FS CustomFlags Values & responsiveness */}
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
													value: RegExp(field.value_validation_regex || ".*"),
													message: `Invalid value for ${field.name}. ${field.value_description}`,
												},
											}}
											slotProps={{ inputLabel: { shrink: true } }}
											helperText={field.value_description}
										/>
									</Grid>
								))}
								<Grid size={12}>
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
									loading={mounting}
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

/*
// Helper to check if a value is a string key of the Flags enum (remains the same)
function isFlagsKey(key: string): key is keyof typeof Flags {
	// Ensure Flags is treated as an object for Object.keys
	return Object.keys(Flags as object).includes(key);
}
*/
