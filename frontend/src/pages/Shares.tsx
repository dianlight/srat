import { DriveFileMove } from "@mui/icons-material";
import AddIcon from "@mui/icons-material/Add";
import BackupIcon from "@mui/icons-material/Backup";
import BlockIcon from "@mui/icons-material/Block";
import CheckCircleIcon from "@mui/icons-material/CheckCircle";
import DataObjectIcon from "@mui/icons-material/DataObject"; // For camelCase
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import ExpandMore from "@mui/icons-material/ExpandMore";
import FolderSharedIcon from "@mui/icons-material/FolderShared";
import FolderSpecialIcon from "@mui/icons-material/FolderSpecial";
import KeyboardCapslockIcon from "@mui/icons-material/KeyboardCapslock"; // For UPPERCASE
import ModeEditIcon from "@mui/icons-material/ModeEdit";
import MoreVertIcon from "@mui/icons-material/MoreVert";
import PlaylistAddIcon from "@mui/icons-material/PlaylistAdd"; // Import an icon for the button
import RemoveIcon from "@mui/icons-material/Remove"; // Import RemoveIcon for kebab-case
import SettingsIcon from "@mui/icons-material/Settings";
import TextDecreaseIcon from "@mui/icons-material/TextDecrease"; // For lowercase
import VisibilityIcon from "@mui/icons-material/Visibility";
import {
	Accordion,
	AccordionDetails,
	AccordionSummary,
	Box,
	Chip,
	Fab,
	FormControlLabel,
	InputAdornment,
	ListItemIcon,
	Menu,
	MenuItem,
	Stack,
	type SvgIconTypeMap,
	Switch,
	Tooltip,
	Typography,
	useMediaQuery,
	useTheme,
} from "@mui/material";
import Avatar from "@mui/material/Avatar";
import Button from "@mui/material/Button";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogContentText from "@mui/material/DialogContentText";
import DialogTitle from "@mui/material/DialogTitle";
import Divider from "@mui/material/Divider";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import ListItemAvatar from "@mui/material/ListItemAvatar";
import ListItemButton from "@mui/material/ListItemButton";
import ListItemText from "@mui/material/ListItemText";
import type { OverridableComponent } from "@mui/material/OverridableComponent";
import { useConfirm } from "material-ui-confirm";
import { MuiChipsInput } from "mui-chips-input";
import { Fragment, useEffect, useMemo, useRef, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import {
	AutocompleteElement,
	CheckboxElement,
	SelectElement,
	SwitchElement,
	TextFieldElement,
} from "react-hook-form-mui";
import { InView } from "react-intersection-observer";
import { useLocation, useNavigate } from "react-router";
import { toast } from "react-toastify";
import { PreviewDialog } from "../components/PreviewDialog";
import { useReadOnly } from "../hooks/readonlyHook";
import { useShare } from "../hooks/shareHook";
import { useVolume } from "../hooks/volumeHook";
import default_json from "../json/default_config.json";
import { addMessage } from "../store/errorSlice";
import { type LocationState, TabIDs } from "../store/locationState";
import {
	type MountPointData,
	type SharedResource,
	Usage,
	type User,
	useDeleteShareByShareNameMutation,
	useGetUsersQuery,
	usePostShareMutation,
	usePutShareByShareNameDisableMutation,
	usePutShareByShareNameEnableMutation,
	usePutShareByShareNameMutation,
} from "../store/sratApi";
import { useAppDispatch } from "../store/store";

interface ShareEditProps extends SharedResource {
	org_name: string;
}

// Helper function to extract basename from a path
function getPathBaseName(path: string): string {
	if (!path) return "";
	// Remove trailing slashes to correctly get the last segment
	const p = path.replace(/\/+$/, "");
	const lastSegment = p.substring(p.lastIndexOf("/") + 1);
	// Return empty string if lastSegment is empty (e.g. path was just "/")
	return lastSegment === "" && p === "/" ? "" : lastSegment;
}

// Helper function to sanitize a string for use as a Windows share name and convert to uppercase
function sanitizeAndUppercaseShareName(name: string): string {
	if (!name) return "";
	// Replace invalid characters (/\:*?"<>|) and whitespace with an underscore, then convert to uppercase
	return name.replace(/[\\/:"*?<>|\s]+/g, "_").toUpperCase();
}

// --- Veto File Entry Validation Helper ---
// Matches a valid Samba veto file entry:
// - Not empty
// - Does not contain '/' (as it's a separator for the list in smb.conf)
// - Does not contain null byte '\0'
const VETO_FILE_ENTRY_REGEX = /^[^/\0]+$/;

function isValidVetoFileEntry(entry: string): boolean {
	if (typeof entry !== "string") return false;
	return VETO_FILE_ENTRY_REGEX.test(entry);
}

// --- Casing Styles and Helpers ---
enum CasingStyle {
	UPPERCASE = "UPPERCASE",
	LOWERCASE = "lowercase",
	CAMELCASE = "camelCase",
	KEBABCASE = "kebab-case",
}

const casingCycleOrder: CasingStyle[] = [
	CasingStyle.UPPERCASE,
	CasingStyle.LOWERCASE,
	CasingStyle.CAMELCASE,
	CasingStyle.KEBABCASE,
];

// Helper to split words based on common separators and camelCase transitions
const splitWords = (str: string): string[] => {
	if (!str) return [];
	const s1 = str.replace(/([a-z0-9])([A-Z])/g, "$1 $2"); // myWord -> my Word
	const s2 = s1.replace(/([A-Z])([A-Z][a-z])/g, "$1 $2"); // ABBRWord -> ABBR Word
	const s3 = s2.replace(/[_-]+/g, " "); // Replace _ and - with space
	return s3.split(/\s+/).filter(Boolean); // Split by space and remove empty parts
};

const toCamelCase = (str: string): string => {
	const words = splitWords(str);
	if (words.length === 0) return "";
	return words
		.map((word, index) =>
			index === 0
				? word.toLowerCase()
				: word.charAt(0).toUpperCase() + word.slice(1).toLowerCase(),
		)
		.join("");
};

const toKebabCase = (str: string): string => {
	const words = splitWords(str);
	if (words.length === 0) return "";
	return words.map((word) => word.toLowerCase()).join("-");
};

const casingStyleToIconMap: Record<
	CasingStyle,
	OverridableComponent<SvgIconTypeMap<{}, "svg">>
> = {
	[CasingStyle.UPPERCASE]: KeyboardCapslockIcon,
	[CasingStyle.LOWERCASE]: TextDecreaseIcon,
	[CasingStyle.CAMELCASE]: DataObjectIcon, // Assuming DataObjectIcon is suitable for camelCase
	[CasingStyle.KEBABCASE]: RemoveIcon,
};

const getCasingIcon = (
	style: CasingStyle,
): OverridableComponent<SvgIconTypeMap<{}, "svg">> => {
	return casingStyleToIconMap[style] || KeyboardCapslockIcon; // Default to UPPERCASE icon if not found
};

interface ShareActionsProps {
	shareKey: string;
	shareProps: SharedResource;
	read_only: boolean;
	onEdit: (shareKey: string, shareProps: SharedResource) => void;
	onViewVolumeSettings: (shareProps: SharedResource) => void;
	onDelete: (shareKey: string, shareProps: SharedResource) => void;
	onEnable: (shareKey: string, shareProps: SharedResource) => void;
	onDisable: (shareKey: string, shareProps: SharedResource) => void;
}

function ShareActions({
	shareKey,
	shareProps,
	read_only,
	onEdit,
	onViewVolumeSettings,
	onDelete,
	onEnable,
	onDisable,
}: ShareActionsProps) {
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

	if (read_only) {
		return null;
	}

	const actionItems = [];

	actionItems.push({
		key: "edit",
		title: "Settings",
		icon: <SettingsIcon />,
		onClick: () => onEdit(shareKey, shareProps),
	});

	if (
		!shareProps.mount_point_data?.invalid &&
		shareProps.usage !== Usage.Internal &&
		shareProps.mount_point_data?.path_hash
	) {
		actionItems.push({
			key: "view-volume",
			title: "View Volume Mount Settings",
			icon: <DriveFileMove />,
			onClick: () => onViewVolumeSettings(shareProps),
		});
	}

	if (shareProps.usage !== Usage.Internal) {
		actionItems.push({
			key: "delete",
			title: "Delete share",
			icon: <DeleteIcon color="error" />,
			onClick: () => onDelete(shareKey, shareProps),
		});
	}

	if (shareProps.disabled) {
		actionItems.push({
			key: "enable",
			title: "Enable share",
			icon: <CheckCircleIcon />,
			onClick: () => onEnable(shareKey, shareProps),
		});
	} else if (shareProps.usage !== Usage.Internal) {
		actionItems.push({
			key: "disable",
			title: "Disable share",
			icon: <BlockIcon />,
			onClick: () => onDisable(shareKey, shareProps),
		});
	}

	if (isSmallScreen) {
		return (
			<>
				<IconButton
					aria-label="more actions"
					aria-controls="share-actions-menu"
					aria-haspopup="true"
					onClick={handleMenuOpen}
					edge="end"
					size="small"
				>
					<MoreVertIcon />
				</IconButton>
				<Menu
					id="share-actions-menu"
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
		<Stack direction="row" spacing={0} alignItems="center">
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

export function Shares() {
	const read_only = useReadOnly();
	const dispatch = useAppDispatch();
	const location = useLocation();
	const navigate = useNavigate();
	const { shares, isLoading, error } = useShare();
	const { disks: volumes, isLoading: vlLoading, error: vlError } = useVolume();
	const [selected, setSelected] = useState<[string, SharedResource] | null>(
		null,
	);
	const [showPreview, setShowPreview] = useState<boolean>(false);
	const [showEdit, setShowEdit] = useState<boolean>(false);
	const [_showUserEdit, _setShowUserEdit] = useState<boolean>(false);
	//const formRef = useRef<HTMLFormElement>(null);
	const [initialNewShareData, setInitialNewShareData] = useState<
		ShareEditProps | undefined
	>(undefined);
	const confirm = useConfirm();
	const [updateShare, _updateShareResult] = usePutShareByShareNameMutation();
	const [deleteShare, _updateDeleteShareResult] =
		useDeleteShareByShareNameMutation();
	const [createShare, _createShareResult] = usePostShareMutation();
	const [enableShare, _enableShareResult] =
		usePutShareByShareNameEnableMutation();
	const [disableShare, _disableShareResult] =
		usePutShareByShareNameDisableMutation();

	// Calculate if a new share can be created
	const canCreateNewShare = useMemo(() => {
		if (vlLoading || isLoading || !volumes || !shares) {
			return false; // Disable if data is loading or not available
		}

		const usedPathHashes = new Set(
			Object.values(shares)
				.map((share) => share.mount_point_data?.path_hash)
				.filter(
					(hash): hash is string => typeof hash === "string" && hash !== "",
				),
		);

		for (const disk of volumes) {
			if (disk.partitions) {
				for (const partition of disk.partitions) {
					if (partition.system) continue; // Skip system partitions
					if (partition.mount_point_data) {
						for (const mpd of partition.mount_point_data) {
							if (
								mpd?.is_mounted &&
								mpd.path &&
								mpd.path_hash &&
								!usedPathHashes.has(mpd.path_hash)
							) {
								return true; // Found an available, unshared, mounted mount point
							}
						}
					}
				}
			}
		}
		return false; // No suitable mount points found
	}, [volumes, shares, vlLoading, isLoading]);

	// Effect to handle navigation state for opening a specific share dialog
	useEffect(() => {
		const state = location.state as LocationState | undefined;
		const shareNameFromState = state?.shareName;
		const newShareDataFromState = state?.newShareData;

		// Check if we have a share name from state and shares data is loaded
		if (shareNameFromState && shares) {
			// Logic for opening an existing share for editing
			const shareEntry = Object.entries(shares).find(
				([_key, value]) => value.name === shareNameFromState,
			);

			if (shareEntry) {
				// Found the share, set selected and show edit dialog
				setSelected(shareEntry);
				setInitialNewShareData(undefined); // Ensure no new share prefill
				setShowEdit(true);
				// Clear the state from history to prevent reopening on refresh/re-render
				navigate(location.pathname, { replace: true, state: {} });
			}
		} else if (newShareDataFromState) {
			let suggestedName = "";
			if (newShareDataFromState.path) {
				const baseName = getPathBaseName(newShareDataFromState.path);
				if (baseName) {
					// Only proceed if basename is not empty
					suggestedName = sanitizeAndUppercaseShareName(baseName);
				}
			}
			// Logic for opening a new share dialog with preselection
			const prefilledData: ShareEditProps = {
				org_name: "", // Signals a new share
				name: suggestedName, // Generated from path basename and normalized
				mount_point_data: newShareDataFromState,
				// Default values for users, ro_users, timemachine, usage will be set by ShareEditDialog's reset
			};
			setInitialNewShareData(prefilledData);
			setSelected(null); // No existing share selected
			setShowEdit(true);
			// Clear the state from history
			navigate(location.pathname, { replace: true, state: {} });
		}
		// Dependencies: shares data and location.state
	}, [shares, location.state, navigate, location.pathname]);

	const groupedAndSortedShares = useMemo(() => {
		if (!shares) {
			return [];
		}

		const groups: Record<string, Array<[string, SharedResource]>> = {};

		Object.entries(shares).forEach(([shareKey, shareProps]) => {
			const usageGroup = shareProps.usage || Usage.None; // Default to 'none' if usage is undefined
			if (!groups[usageGroup]) {
				groups[usageGroup] = [];
			}
			groups[usageGroup].push([shareKey, shareProps]);
		});

		// Sort shares within each group by name
		for (const usageGroup in groups) {
			groups[usageGroup].sort((a, b) =>
				(a[1].name || "").localeCompare(b[1].name || ""),
			);
		}

		// Sort the groups by usage type (key)
		return Object.entries(groups).sort((a, b) => a[0].localeCompare(b[0]));
	}, [shares]);

	// localStorage key for expanded accordion
	const localStorageKey = "srat_shares_expanded_accordion";

	// State to manage open/closed state of usage groups
	const [expandedAccordion, setExpandedAccordion] = useState<string | false>(
		() => {
			// Initialize from localStorage directly if possible, only on first render.
			// Validation against actual groups will happen in the useEffect.
			const savedAccordionId = localStorage.getItem(localStorageKey);
			return savedAccordionId || false;
		},
	);
	const initialSetupDone = useRef(false); // Tracks if initial setup (load/auto-open) is done

	// Store previous groupedAndSortedShares to detect changes for resetting setup
	const prevGroupedSharesRef = useRef<
		typeof groupedAndSortedShares | undefined
	>(undefined);

	useEffect(() => {
		if (prevGroupedSharesRef.current !== groupedAndSortedShares) {
			// Groups data has changed (e.g., loaded, added/removed), reset the setup flag
			// to allow re-validation and defaulting for the new set of groups.
			initialSetupDone.current = false;
			prevGroupedSharesRef.current = groupedAndSortedShares;
		}

		// If initial setup for the current set of groups is already done, nothing more to do here.
		if (initialSetupDone.current) {
			// However, if groups became empty *after* setup was done, ensure accordion is closed.
			if (
				(!groupedAndSortedShares || groupedAndSortedShares.length === 0) &&
				expandedAccordion !== false
			) {
				setExpandedAccordion(false);
			}
			return;
		}

		// At this point, initialSetupDone.current is false.

		// If groups are not yet loaded or are empty, wait.
		// Do not modify expandedAccordion, as it holds the optimistic value from localStorage.
		if (!groupedAndSortedShares || groupedAndSortedShares.length === 0) {
			return;
		}

		// Groups are NOW loaded, and initialSetupDone.current is false.
		// This is the first opportunity to validate expandedAccordion (from localStorage)
		// against the *actual* loaded groups.
		const isValidCurrentExpanded =
			typeof expandedAccordion === "string" &&
			groupedAndSortedShares.some(
				([groupName]) => groupName === expandedAccordion,
			);

		if (!isValidCurrentExpanded) {
			// The value in expandedAccordion (from localStorage or default 'false') is not valid for the current groups.
			// Default to the first available group.
			const firstGroupName = groupedAndSortedShares[0]?.[0];
			setExpandedAccordion(firstGroupName || false);
		}
		// If isValidCurrentExpanded is true, the value from localStorage was valid, so expandedAccordion remains as is.

		initialSetupDone.current = true;
	}, [groupedAndSortedShares, expandedAccordion]); // expandedAccordion is included because its current value is read and might trigger a set if invalid.
	// initialSetupDone.current prevents re-defaulting after user interaction.

	// Effect to save expanded accordion to localStorage
	useEffect(() => {
		if (expandedAccordion === false) {
			localStorage.removeItem(localStorageKey);
		} else if (typeof expandedAccordion === "string") {
			// Only save to localStorage if the initial setup/validation is done and groups are present.
			// This prevents saving an unvalidated localStorage value back to itself or clearing it prematurely.
			if (
				initialSetupDone.current &&
				groupedAndSortedShares &&
				groupedAndSortedShares.length > 0
			) {
				if (
					groupedAndSortedShares.some(
						([groupName]) => groupName === expandedAccordion,
					)
				) {
					localStorage.setItem(localStorageKey, expandedAccordion);
				} else {
					// expandedAccordion is a string, but not in the current valid groups (e.g., group was deleted).
					localStorage.removeItem(localStorageKey);
				}
			} else if (
				initialSetupDone.current &&
				groupedAndSortedShares &&
				groupedAndSortedShares.length === 0
			) {
				// Groups are confirmed empty after setup, so any string ID is invalid.
				localStorage.removeItem(localStorageKey);
			}
			// If initialSetupDone.current is false, groups are still loading/validating; don't touch localStorage for a string value yet.
		}
	}, [expandedAccordion, groupedAndSortedShares]); // initialSetupDone.current ensures we save based on a validated state.

	const handleAccordionChange =
		(panel: string) => (_event: React.SyntheticEvent, isExpanded: boolean) => {
			setExpandedAccordion(isExpanded ? panel : false);
		};

	function onSubmitDisableShare(cdata?: string, props?: SharedResource) {
		console.log("Disable", cdata, props);
		if (!cdata || !props) return;
		confirm({
			title: `Disable ${props?.name}?`,
			description:
				"If you disable this share, all of its configurations will be retained.",
			acknowledgement:
				"I understand that disabling the share will retain its configurations but prevent access to it.",
		}).then(({ confirmed, reason }) => {
			if (confirmed) {
				disableShare({ shareName: props?.name || "" })
					.unwrap()
					.then(() => {
						//                        setErrorInfo('');
					})
					.catch((err) => {
						dispatch(addMessage(JSON.stringify(err)));
					});
			} else if (reason === "cancel") {
				console.log("cancel");
			}
		});
	}

	function onSubmitEnableShare(cdata?: string, props?: SharedResource) {
		console.log("Enable", cdata, props);
		if (!cdata || !props) return;
		enableShare({ shareName: props?.name || "" })
			.unwrap()
			.then(() => {
				//            setErrorInfo('');
			})
			.catch((err) => {
				dispatch(addMessage(JSON.stringify(err)));
			});
	}

	function onSubmitDeleteShare(cdata?: string, props?: SharedResource) {
		console.log("Delete", cdata, props);
		if (!cdata || !props) return;
		confirm({
			title: `Delete ${props?.name}?`,
			description:
				"This action cannot be undone. Are you sure you want to delete this share?",
			acknowledgement:
				"I understand that deleting the share will remove it permanently.",
		}).then(({ confirmed, reason }) => {
			if (confirmed) {
				deleteShare({ shareName: props?.name || "" })
					.unwrap()
					.then(() => {
						//                        setErrorInfo('');
					})
					.catch((err) => {
						dispatch(addMessage(JSON.stringify(err)));
					});
			} else if (reason === "cancel") {
				console.log("cancel");
			}
		});
	}

	function onSubmitEditShare(data?: ShareEditProps) {
		console.log("Edit Share", data, selected);
		if (!data) return;
		if (!data.name || !data.mount_point_data?.path) {
			dispatch(addMessage("Unable to open share!"));
			return;
		}

		// Save Data
		console.log(data);
		if (data.org_name !== "") {
			// Existing share being updated
			updateShare({ shareName: data.org_name, sharedResource: data })
				.unwrap()
				.then((res) => {
					toast.info(
						`Share ${(res as SharedResource).name} modified successfully.`,
					);
					setSelected(null);
					setShowEdit(false);
				})
				.catch((err) => {
					dispatch(addMessage(JSON.stringify(err)));
				});
		} else {
			// New share being created
			createShare({ sharedResource: data })
				.unwrap()
				.then((res) => {
					toast.info(
						`Share ${(res as SharedResource).name || data.name} created successfully.`,
					);
					setSelected(null);
					setShowEdit(false);
				})
				.catch((err) => {
					dispatch(addMessage(JSON.stringify(err)));
				});
		}

		return false;
	}

	return (
		<InView>
			<PreviewDialog
				title={selected?.[1].name || ""}
				objectToDisplay={selected?.[1]}
				open={showPreview}
				onClose={() => {
					setSelected(null);
					setShowPreview(false);
				}}
			/>
			<ShareEditDialog
				objectToEdit={
					selected
						? { ...selected[1], org_name: selected[1].name || "" } // For editing existing share
						: initialNewShareData // For new share, potentially with prefilled data from Volumes
				}
				open={showEdit}
				shares={shares} // Pass the shares data to the dialog
				onClose={(data) => {
					onSubmitEditShare(data);
					setSelected(null);
					setShowEdit(false);
					setInitialNewShareData(undefined); // Clear prefill data after dialog closes
				}}
				onDeleteSubmit={onSubmitDeleteShare}
			/>
			<br />
			<Stack
				direction="row"
				justifyContent="flex-end"
				sx={{ px: 2, mb: 1, alignItems: "center" }}
			>
				<Tooltip
					title={
						read_only
							? "Cannot create share in read-only mode"
							: !canCreateNewShare
								? "No unshared mount points available to create a new share."
								: "Create new share"
					}
				>
					<span>
						{" "}
						{/* Wrapper for Tooltip when Fab might be disabled */}
						<Fab
							id="create_new_share"
							color="primary"
							aria-label={
								read_only
									? "Cannot create share in read-only mode"
									: !canCreateNewShare
										? "No unshared mount points available to create a new share."
										: "Create new share"
							}
							// sx removed: float, top, margin - FAB is now in normal flow within Stack
							size="small"
							onClick={() => {
								if (!read_only && canCreateNewShare) {
									setSelected(null);
									setShowEdit(true);
								}
							}}
							disabled={read_only || !canCreateNewShare}
						>
							<AddIcon />
						</Fab>
					</span>
				</Tooltip>
			</Stack>
			<List dense={true}>
				<Divider />
				{groupedAndSortedShares.map(
					([usageGroup, sharesInGroup], _groupIndex) => (
						<Accordion
							key={usageGroup}
							expanded={expandedAccordion === usageGroup}
							onChange={handleAccordionChange(usageGroup)}
							sx={{
								boxShadow: "none", // Remove default shadow for a flatter look if desired
								"&:before": { display: "none" }, // Remove the top border line of the accordion
								"&.Mui-expanded": { margin: "auto 0" }, // Control margin when expanded
								backgroundColor: "transparent", // Remove accordion background
							}}
							disableGutters // Removes left/right padding from Accordion itself
						>
							<AccordionSummary
								expandIcon={<ExpandMore />}
								aria-controls={`${usageGroup}-content`}
								id={`${usageGroup}-header`}
								sx={{
									minHeight: 48, // Adjust as needed
									"&.Mui-expanded": { minHeight: 48 }, // Ensure consistent height
									"& .MuiAccordionSummary-content": { margin: "12px 0" }, // Adjust content margin
									backgroundColor: "transparent", // Remove accordion background
								}}
							>
								<Typography
									variant="subtitle2"
									color="text.primary"
									sx={{ textTransform: "capitalize", pl: 1 }}
								>
									{usageGroup} Shares ({sharesInGroup.length})
								</Typography>
							</AccordionSummary>
							<AccordionDetails sx={{ p: 0 }}>
								{" "}
								{/* Remove padding from details to allow List to control it */}
								<List component="div" disablePadding dense={true}>
									{sharesInGroup.map(([share, props]) => (
										<Fragment key={share}>
											<ListItemButton
												sx={{
													opacity: props.disabled ? 0.5 : 1,
													"&:hover": {
														opacity: 1,
													},
												}}
											>
												<ListItem
													secondaryAction={
														<ShareActions
															shareKey={share}
															shareProps={props}
															read_only={read_only}
															onEdit={(shareKey, shareProps) => {
																setSelected([shareKey, shareProps]);
																setShowEdit(true);
															}}
															onViewVolumeSettings={(shareProps) => {
																if (shareProps.mount_point_data?.path_hash) {
																	navigate("/", {
																		state: {
																			tabId: TabIDs.VOLUMES,
																			mountPathHashToView:
																				shareProps.mount_point_data.path_hash,
																			openMountSettings: true,
																		} as LocationState,
																	});
																}
															}}
															onDelete={onSubmitDeleteShare}
															onEnable={onSubmitEnableShare}
															onDisable={onSubmitDisableShare}
														/>
													}
												>
													<ListItemAvatar>
														<Avatar>
															{(props.mount_point_data?.invalid && (
																<Tooltip
																	title={props.mount_point_data?.invalid_error}
																	arrow
																>
																	<FolderSharedIcon color="error" />
																</Tooltip>
															)) || (
																	<Tooltip
																		title={props.mount_point_data?.warnings}
																		arrow
																	>
																		<FolderSharedIcon />
																	</Tooltip>
																)}
														</Avatar>
													</ListItemAvatar>
													<ListItemText
														primary={
															<Box
																sx={{
																	display: "flex",
																	alignItems: "center",
																	gap: 1,
																}}
															>
																{props.name}
															</Box>
														}
														onClick={() => {
															setSelected([share, props]);
															setShowPreview(true);
														}}
														secondary={
															<Typography variant="body2" component="div">
																{props.mount_point_data?.path && (
																	<Box
																		component="span"
																		sx={{ display: "block" }}
																	>
																		Path: {props.mount_point_data.path}
																	</Box>
																)}
																{props.mount_point_data?.warnings &&
																	props.usage !== Usage.Internal && (
																		<Box
																			component="span"
																			sx={{ display: "block", color: "orange" }}
																		>
																			Warning: {props.mount_point_data.warnings}
																		</Box>
																	)}
																<Stack
																	direction="row"
																	spacing={1}
																	flexWrap="wrap"
																	alignItems="center"
																	sx={{
																		mt: 1,
																		display: { xs: "none", sm: "flex" },
																	}}
																>
																	{props.users && props.users.length > 0 && (
																		<Tooltip title="Users with write access">
																			<Chip
																				onClick={(e) => {
																					e.stopPropagation();
																					setSelected([share, props]);
																					setShowEdit(true);
																				}}
																				size="small"
																				icon={<EditIcon />}
																				variant="outlined"
																				label={
																					<Typography
																						variant="body2"
																						component="span"
																					>
																						Users:{" "}
																						{props.users.map((u) => (
																							<Typography
																								variant="body2"
																								component="span"
																								key={u.username}
																								color={
																									u.is_admin
																										? "warning"
																										: "inherit"
																								}
																							>
																								{u.username}
																								{u !==
																									props.users?.[
																									props.users?.length - 1
																									] && ", "}
																							</Typography>
																						))}
																					</Typography>
																				}
																				sx={{ my: 0.5 }}
																			/>
																		</Tooltip>
																	)}
																	{props.ro_users &&
																		props.ro_users.length > 0 && (
																			<Tooltip title="Users with read-only access">
																				<Chip
																					onClick={(e) => {
																						e.stopPropagation();
																						setSelected([share, props]);
																						setShowEdit(true);
																					}}
																					size="small"
																					icon={<VisibilityIcon />}
																					variant="outlined"
																					label={
																						<span>
																							Read-only Users:{" "}
																							{props.ro_users.map((u) => (
																								<span
																									key={u.username}
																									style={{
																										color: u.is_admin
																											? "warning"
																											: "inherit",
																									}}
																								>
																									{u.username}
																									{u !==
																										props.ro_users?.[
																										props.ro_users?.length - 1
																										] && ", "}
																								</span>
																							))}
																						</span>
																					}
																					sx={{ my: 0.5 }}
																				/>
																			</Tooltip>
																		)}
																	{props.usage &&
																		props.usage !== Usage.Internal && (
																			<Tooltip
																				title={`Share Usage: ${props.is_ha_mounted ? "HA Mounted" : "Not HA Mounted"}`}
																			>
																				<Chip
																					onClick={(e) => {
																						e.stopPropagation();
																						setSelected([share, props]);
																						setShowEdit(true);
																					}}
																					size="small"
																					variant="outlined"
																					icon={<FolderSpecialIcon />}
																					disabled={!props.is_ha_mounted}
																					label={`Usage: ${props.usage}`}
																					sx={{ my: 0.5 }}
																				/>
																			</Tooltip>
																		)}
																	{props.timemachine && (
																		<Tooltip title="TimeMachine Enabled">
																			<Chip
																				onClick={(e) => {
																					e.stopPropagation();
																					setSelected([share, props]);
																					setShowEdit(true);
																				}}
																				size="small"
																				variant="outlined"
																				icon={<BackupIcon />}
																				label="TimeMachine"
																				color="secondary"
																				sx={{ my: 0.5 }}
																			/>
																		</Tooltip>
																	)}{" "}
																</Stack>
															</Typography>
														}
													/>
												</ListItem>
											</ListItemButton>
											<Divider component="li" />
										</Fragment>
									))}
								</List>
							</AccordionDetails>
							{/* Divider is implicitly handled by Accordion borders, or can be added if a stronger visual separation is needed */}
						</Accordion>
					),
				)}
			</List>
		</InView>
	);
}

interface ShareEditDialogProps {
	open: boolean;
	onClose: (data?: ShareEditProps) => void;
	objectToEdit?: ShareEditProps;
	shares?: SharedResource[]; // Added to receive shares data
	onDeleteSubmit?: (shareName: string, shareData: SharedResource) => void; // Added for delete action
	onToggleEnabled?: (enabled: boolean) => void; // Added for enable/disable toggle
}
function ShareEditDialog(props: ShareEditDialogProps) {
	const {
		data: users,
		isLoading: usLoading,
		error: usError,
	} = useGetUsersQuery();
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
	const isDisabled = watch('disabled');

	useEffect(() => {
		if (props.open) {
			const adminUser = Array.isArray(users)
				? users.find((u) => u.is_admin)
				: undefined;

			if (props.objectToEdit) {
				// Covers editing existing share OR new share with prefill
				const isNewShareCreation = props.objectToEdit.org_name === "";
				reset({
					org_name: props.objectToEdit.org_name ?? "", // Key to determine if new/edit
					name: props.objectToEdit.name || "",
					mount_point_data: props.objectToEdit.mount_point_data, // This is the preselection
					// If it's a new share creation and no users are pre-filled, default to admin.
					// Otherwise, use the users from objectToEdit (could be empty for new, or populated for existing).
					users:
						isNewShareCreation &&
							(!props.objectToEdit.users ||
								props.objectToEdit.users.length === 0) &&
							adminUser
							? [adminUser]
							: props.objectToEdit.users || [],
					ro_users: props.objectToEdit.ro_users || [],
					timemachine: props.objectToEdit.timemachine || false,
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
					org_name: "",
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
				org_name: "",
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
			case CasingStyle.UPPERCASE:
				transformedName = currentName.toUpperCase();
				break;
			case CasingStyle.LOWERCASE:
				transformedName = currentName.toLowerCase();
				break;
			case CasingStyle.CAMELCASE:
				transformedName = toCamelCase(currentName);
				break;
			case CasingStyle.KEBABCASE:
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
	const cycleCasingTooltipTitle = `Cycle casing (Next: ${nextCasingStyleName.charAt(0).toUpperCase() + nextCasingStyleName.slice(1)})`;
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
				<DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
					<Box sx={{ display: 'flex', alignItems: 'center', flex: 1 }}>
						{!(editName || props.objectToEdit?.org_name === "") && (
							<>
								<IconButton onClick={() => setEditName(true)}>
									<ModeEditIcon fontSize="small" />
								</IconButton>
								{props.objectToEdit?.name}
							</>
						)}
					</Box>
					{props.objectToEdit?.org_name !== "" && (

						<SwitchElement
							control={control}
							name="disabled"
							color="primary"
							label={isDisabled ? "Disabled" : "Enabled"}
							sx={{ mr: 0 }}
						/>
					)}
					{(editName || props.objectToEdit?.org_name === "") && (
						<TextFieldElement
							sx={{ display: "flex" }}
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
								{props.objectToEdit?.usage !== Usage.Internal && (
									<Grid size={6}>
										<SelectElement
											sx={{ display: "flex" }}
											size="small"
											label="Usage"
											name="usage"
											disabled={isDisabled}
											options={Object.keys(Usage)
												.filter(
													(usage) => usage.toLowerCase() !== Usage.Internal,
												)
												.map((usage) => {
													return { id: usage.toLowerCase(), label: usage };
												})}
											required
											control={control}
										/>
									</Grid>
								)}
								{props.objectToEdit?.usage !== Usage.Internal && (
									<>
										<Grid size={6}>
											<AutocompleteElement
												label="Volume"
												name="mount_point_data"
												options={
													(volumes
														?.flatMap((disk) => disk.partitions)
														?.filter(Boolean)
														.filter((partition) => !partition?.system)
														.flatMap((partition) => partition?.mount_point_data)
														.filter(
															(mp) => mp?.path !== "",
														) as MountPointData[]) || ([] as MountPointData[])
												}
												control={control}
												required
												loading={vlLoading}
												autocompleteProps={{
													disabled: isDisabled,
													size: "small",
													renderValue: (value) => {
														return (value as MountPointData).path || "--";
													},
													getOptionLabel: (option) =>
														(option as MountPointData)?.path || "",
													getOptionKey: (option) =>
														(option as MountPointData)?.path_hash || "",
													renderOption: (props, option) => (
														<li {...props} key={props.key}>
															<Typography variant="body2">
																{option.path}
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
																	existingShare.name === currentEditingShareName
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
										</Grid>
										<Grid size={12}>
											<Controller
												name="veto_files"
												control={control}
												defaultValue={[]}
												rules={{
													validate: (chips: string[] | undefined) => {
														if (!chips || chips == null || chips.length === 0)
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
																	{...props}
																	sx={{
																		color: isDefault
																			? "text.secondary"
																			: "text.primary",
																	}}
																	size="small"
																	key={key}
																/>
															);
														}}
														slotProps={{
															input: {
																endAdornment: (
																	<InputAdornment position="end" sx={{ pr: 1 }}>
																		<Tooltip title="Add suggested default Veto files">
																			<span>
																				<IconButton
																					disabled={isDisabled}
																					aria-label="add suggested default veto files"
																					onClick={() => {
																						const currentVetoFiles: string[] =
																							getValues("veto_files") || [];
																						const defaultVetoFiles: string[] =
																							default_json.veto_files || [];
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
										<Grid size={6}>
											<CheckboxElement
												size="small"
												label="Support Timemachine Backups"
												name="timemachine"
												disabled={isDisabled}
												control={control}
											/>
										</Grid>
										<Grid size={6}>
											<CheckboxElement
												size="small"
												label="Support Recycle Bin"
												name="recycle_bin_enabled"
												disabled={isDisabled}
												control={control}
											/>
										</Grid>
									</>
								)}
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
												disabled: isDisabled,
												size: "small",
												limitTags: 5,
												getOptionKey: (option) =>
													(option as User).username || "",
												getOptionLabel: (option) =>
													(option as User).username || "",
												renderOption: (props, option) => (
													<li {...props} key={props.key}>
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
															(user) => user.username === option.username,
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
																label={(option as User)?.username || "bobo"}
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
								<Grid size={6}>
									{!usLoading && ((users as User[]) || []).length > 0 && (
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
													<li {...props} key={props.key}>
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
															(user) => user.username === option.username,
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
																label={(option as User)?.username || "bobo"}
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
						{props.objectToEdit?.org_name === "" ? "Create" : "Apply"}
					</Button>
				</DialogActions>
			</Dialog>
		</Fragment>
	);
}
