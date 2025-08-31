import AddIcon from "@mui/icons-material/Add";
import BackupIcon from "@mui/icons-material/Backup";
import EditIcon from "@mui/icons-material/Edit";
import ExpandMore from "@mui/icons-material/ExpandMore";
import FolderSharedIcon from "@mui/icons-material/FolderShared";
import FolderSpecialIcon from "@mui/icons-material/FolderSpecial";
import VisibilityIcon from "@mui/icons-material/Visibility";
import {
	Accordion,
	AccordionDetails,
	AccordionSummary,
	Box,
	Chip,
	Fab,
	Stack,
	Tooltip,
	Typography,
} from "@mui/material";
import Avatar from "@mui/material/Avatar";
import Divider from "@mui/material/Divider";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import ListItemAvatar from "@mui/material/ListItemAvatar";
import ListItemButton from "@mui/material/ListItemButton";
import ListItemText from "@mui/material/ListItemText";
import { useConfirm } from "material-ui-confirm";
import { Fragment, useEffect, useMemo, useRef, useState } from "react";
import { InView } from "react-intersection-observer";
import { useLocation, useNavigate } from "react-router";
import { toast } from "react-toastify";
import { PreviewDialog } from "../../components/PreviewDialog";
import { useShare } from "../../hooks/shareHook";
import { useVolume } from "../../hooks/volumeHook";
import { addMessage } from "../../store/errorSlice";
import { type LocationState, TabIDs } from "../../store/locationState";
import {
	type SharedResource,
	Usage,
	useDeleteApiShareByShareNameMutation,
	usePostApiShareMutation,
	usePutApiShareByShareNameDisableMutation,
	usePutApiShareByShareNameEnableMutation,
	usePutApiShareByShareNameMutation,
} from "../../store/sratApi";
import { useAppDispatch } from "../../store/store";
import { TourEvents, TourEventTypes } from "../../utils/TourEvents";
import { ShareActions } from "./ShareActions";
import { ShareEditDialog } from "./ShareEditDialog";
import type { ShareEditProps } from "./types";
import { getPathBaseName, sanitizeAndUppercaseShareName } from "./utils";
import { filesize } from "filesize";
import StorageIcon from "@mui/icons-material/Storage";
import { useGetServerEventsQuery } from "../../store/sseApi";


export function Shares() {
	const { data: evdata, isLoading: is_evLoading } = useGetServerEventsQuery();
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
	>(
		undefined,
	);
	const confirm = useConfirm();
	const [updateShare, _updateShareResult] = usePutApiShareByShareNameMutation();
	const [deleteShare, _updateDeleteShareResult] =
		useDeleteApiShareByShareNameMutation();
	const [createShare, _createShareResult] = usePostApiShareMutation();
	const [enableShare, _enableShareResult] =
		usePutApiShareByShareNameEnableMutation();
	const [disableShare, _disableShareResult] =
		usePutApiShareByShareNameDisableMutation();

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
					//if (partition.system) continue; // Skip system partitions
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
						`Share ${(res as SharedResource).name || data.name
						} created successfully.`,
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

	TourEvents.on(TourEventTypes.SHARES_STEP_3, (elem) => {
		setExpandedAccordion(false);
		//console.debug("Tour Step 3:", elem);
	});
	TourEvents.on(TourEventTypes.SHARES_STEP_4, (elem) => {
		setExpandedAccordion(groupedAndSortedShares?.[0]?.[0]);
		//console.debug("Tour Step 4:", elem, groupedAndSortedShares);
	});

	return (
		<InView data-tutor={`reactour__tab${TabIDs.SHARES}__step0`}>
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
						evdata?.hello?.read_only
							? "Cannot create share in read-only mode"
							: !canCreateNewShare
								? "No unshared mount points available to create a new share."
								: "Create new share"
					}
					data-tutor={`reactour__tab${TabIDs.SHARES}__step2`}
				>
					<span>
						<Fab
							id="create_new_share"
							color="primary"
							aria-label={
								evdata?.hello?.read_only
									? "Cannot create share in read-only mode"
									: !canCreateNewShare
										? "No unshared mount points available to create a new share."
										: "Create new share"
							}
							// sx removed: float, top, margin - FAB is now in normal flow within Stack
							size="small"
							onClick={() => {
								if (!evdata?.hello?.read_only && canCreateNewShare) {
									setSelected(null);
									setShowEdit(true);
								}
							}}
							disabled={evdata?.hello?.read_only || !canCreateNewShare}
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
							data-tutor={`reactour__tab${TabIDs.SHARES}__step3`}
						>
							<AccordionSummary
								expandIcon={<ExpandMore />}
								aria-controls={`${usageGroup}-content`}
								id={`${usageGroup}-header`}
								sx={{
									minHeight: 48, // Adjust as needed
									"&.Mui-expanded": { minHeight: 48 }, // Ensure consistent height
									"& .MuiAccordionSummary-content": {
										margin: "12px 0",
									}, // Adjust content margin
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
															read_only={evdata?.hello?.read_only || true}
															onEdit={(shareKey, shareProps) => {
																setSelected([shareKey, shareProps]);
																setShowEdit(true);
															}}
															onViewVolumeSettings={(shareProps) => {
																if (
																	shareProps.mount_point_data?.path_hash
																) {
																	navigate("/", {
																		state: {
																			tabId: TabIDs.VOLUMES,
																			mountPathHashToView:
																				shareProps.mount_point_data
																					.path_hash,
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
														<Avatar
															sx={{ width: 32, height: 32 }}
															onClick={() => {
																setSelected([share, props]);
																setShowPreview(true);
															}}
														>
															{(props.mount_point_data?.invalid && (
																<Tooltip
																	title={
																		props.mount_point_data?.invalid_error
																	}
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
														secondary={
															<Typography variant="body2" component="div">
																{props.mount_point_data?.disk_label && (
																	<Chip
																		size="small"
																		icon={<StorageIcon />}
																		variant="outlined"
																		label={`Volume: ${props.mount_point_data.disk_label}`}
																	/>
																)}
																{!props.mount_point_data?.is_write_supported && (
																	<Chip
																		label="Read-Only"
																		size="small"
																		variant="outlined"
																		color="secondary"
																	/>
																)}
																{props.mount_point_data?.disk_size && (
																	<Chip
																		label={`Size: ${filesize(props.mount_point_data.disk_size, { round: 1 })}`}
																		size="small"
																		variant="outlined"
																	/>
																)}
																{props.mount_point_data?.warnings &&
																	props.usage !== Usage.Internal && (
																		<Box
																			component="span"
																			sx={{
																				display: "block",
																				color: "orange",
																			}}
																		>
																			Warning:{" "}
																			{props.mount_point_data.warnings}
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
																									props.users?.[(
																										props.users
																											?.length || 0
																									) - 1] && ", "}
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
																										props.ro_users?.[(
																											props.ro_users
																												?.length || 0
																										) - 1] && ", "}
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
																				title={`Share Usage: ${props.is_ha_mounted
																					? "HA Mounted"
																					: "Not HA Mounted"
																					}`}
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