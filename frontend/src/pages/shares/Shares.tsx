import AddIcon from "@mui/icons-material/Add";
import {
	Box,
	Button,
	Grid,
	IconButton,
	Paper,
	Stack,
	Tooltip,
	Typography,
} from "@mui/material";
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
	usePutApiShareByShareNameMutation,
} from "../../store/sratApi";
import { useAppDispatch } from "../../store/store";
import { TourEvents, TourEventTypes } from "../../utils/TourEvents";
import { ShareEditDialog } from "./ShareEditDialog";
import type { ShareEditProps } from "./types";
import { getPathBaseName, sanitizeAndUppercaseShareName } from "./utils";
import { useGetServerEventsQuery } from "../../store/sseApi";
import { SharesTreeView, ShareDetailsPanel, ShareEditForm } from "./components";


export function Shares() {
	const { data: evdata, isLoading: is_evLoading } = useGetServerEventsQuery();
	const dispatch = useAppDispatch();
	const location = useLocation();
	const navigate = useNavigate();
	const { shares, isLoading, error } = useShare();
	const { disks: volumes, isLoading: vlLoading, error: vlError } = useVolume();
	const [selectedShareKey, setSelectedShareKey] = useState<string | undefined>(() => localStorage.getItem("shares.selectedShareKey") || undefined);
	const [selectedShare, setSelectedShare] = useState<SharedResource | null>(null);
	const [expandedGroups, setExpandedGroups] = useState<string[]>(() => {
		try {
			const savedExpanded = localStorage.getItem("shares.expandedGroups");
			if (savedExpanded) {
				const parsed = JSON.parse(savedExpanded);
				if (Array.isArray(parsed)) return parsed as string[];
			}
		} catch { }
		return [];
	});
	const [showPreview, setShowPreview] = useState<boolean>(false);
	const [showEdit, setShowEdit] = useState<boolean>(false);
	const [initialNewShareData, setInitialNewShareData] = useState<
		ShareEditProps | undefined
	>(undefined);
	const confirm = useConfirm();
	const [updateShare, _updateShareResult] = usePutApiShareByShareNameMutation();
	const [deleteShare, _updateDeleteShareResult] = useDeleteApiShareByShareNameMutation();
	const [createShare, _createShareResult] = usePostApiShareMutation();

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
			const partitions = Object.values(disk.partitions || {});
			if (partitions.length > 0) {
				for (const partition of partitions) {
					const mpds = Object.values(partition.mount_point_data || {});
					for (const mpd of mpds) {
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
		return false; // No suitable mount points found
	}, [volumes, shares, vlLoading, isLoading]);

	// Persist selection and expanded groups to localStorage
	useEffect(() => {
		try {
			if (selectedShareKey) {
				localStorage.setItem("shares.selectedShareKey", selectedShareKey);
			} else {
				localStorage.removeItem("shares.selectedShareKey");
			}
		} catch (err) {
			console.warn("Could not persist selectedShareKey", err);
		}
	}, [selectedShareKey]);

	useEffect(() => {
		try {
			if (expandedGroups.length > 0) {
				localStorage.setItem("shares.expandedGroups", JSON.stringify(expandedGroups));
			} else {
				localStorage.removeItem("shares.expandedGroups");
			}
		} catch (err) {
			console.warn("Could not persist expandedGroups", err);
		}
	}, [expandedGroups]);

	// When shares data is available and there's a selectedShareKey (restored or new), find and select it so details show
	useEffect(() => {
		if (!shares || Object.keys(shares).length === 0) return;
		if (!selectedShareKey) return;

		// Try to locate the share corresponding to selectedShareKey
		const shareEntry = Object.entries(shares).find(([key, _]) => key === selectedShareKey);

		if (shareEntry) {
			const [key, shareData] = shareEntry;
			setSelectedShare(shareData);

			// Ensure the containing group is expanded
			const usageGroup = shareData.usage || Usage.None;
			const groupId = `group-${usageGroup}`;
			setExpandedGroups((prev) => {
				if (prev.includes(groupId)) return prev;
				return [...prev, groupId];
			});
		} else {
			// If share not found, clear selection and remove from localStorage
			setSelectedShare(null);
			setSelectedShareKey(undefined);
		}
	}, [shares, selectedShareKey]);

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
				const [shareKey, shareData] = shareEntry;
				setSelectedShareKey(shareKey);
				setSelectedShare(shareData);
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
				// Default values for users, ro_users, timemachine, usage will be set by ShareEditForm's reset
			};
			setInitialNewShareData(prefilledData);
			setSelectedShareKey(undefined); // No existing share selected
			setSelectedShare(null);
			setShowEdit(true);
			// Clear the state from history
			navigate(location.pathname, { replace: true, state: {} });
		}
		// Dependencies: shares data and location.state
	}, [shares, location.state, navigate, location.pathname]);

	const handleShareSelect = (shareKey: string, share: SharedResource) => {
		setSelectedShareKey(shareKey);
		setSelectedShare(share);
		setShowEdit(false); // Reset edit mode when selecting a new share

		// Ensure the containing group is expanded and persisted
		const usageGroup = share.usage || Usage.None;
		const groupId = `group-${usageGroup}`;
		setExpandedGroups((prev) => {
			if (prev.includes(groupId)) return prev;
			return [...prev, groupId];
		});
	};

	function onSubmitDeleteShare(shareName: string, shareData: SharedResource) {
		console.log("Delete", shareName, shareData);
		if (!shareName || !shareData) return;
		confirm({
			title: `Delete ${shareData?.name}?`,
			description:
				"This action cannot be undone. Are you sure you want to delete this share?",
			acknowledgement:
				"I understand that deleting the share will remove it permanently.",
		}).then(({ confirmed, reason }) => {
			if (confirmed) {
				deleteShare({ shareName: shareData?.name || "" })
					.unwrap()
					.then(() => {
						// Clear selection if deleted share was selected
						if (selectedShareKey && selectedShare?.name === shareData.name) {
							setSelectedShareKey(undefined);
							setSelectedShare(null);
							setShowEdit(false);
						}
					})
					.catch((err) => {
						dispatch(addMessage(JSON.stringify(err)));
					});
			} else if (reason === "cancel") {
				console.log("cancel");
			}
		});
	}

	function onSubmitEditShare(data: ShareEditProps) {
		console.log("Edit Share", data, selectedShare);
		if (!data) return;
		if (!data.name || !data.mount_point_data?.path) {
			dispatch(addMessage("Unable to save share!"));
			return;
		}

		// Save Data
		console.log(data);
		if (data.org_name !== "" && data.org_name !== undefined) {
			// Existing share being updated
			updateShare({ shareName: data.org_name, sharedResource: data })
				.unwrap()
				.then((res) => {
					toast.info(
						`Share ${(res as SharedResource).name} modified successfully.`,
					);
					// Update local state with new data
					if (selectedShareKey) {
						setSelectedShare(res as SharedResource);
					}
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
					setSelectedShareKey(undefined);
					setSelectedShare(null);
					setShowEdit(false);
					setInitialNewShareData(undefined);
				})
				.catch((err) => {
					dispatch(addMessage(JSON.stringify(err)));
				});
		}
	}

	return (
		<>
			<PreviewDialog
				title={selectedShare?.name || ""}
				objectToDisplay={selectedShare}
				open={showPreview}
				onClose={() => {
					setSelectedShareKey(undefined);
					setSelectedShare(null);
					setShowPreview(false);
				}}
			/>
			<ShareEditDialog
				objectToEdit={
					selectedShare && selectedShareKey
						? { ...selectedShare, org_name: selectedShare.name || "" } // For editing existing share
						: initialNewShareData // For new share, potentially with prefilled data from Volumes
				}
				open={showEdit && !selectedShareKey} // Only show dialog for new shares without tree selection
				shares={shares ? Object.values(shares) : undefined} // Pass the shares data to the dialog
				onClose={(data) => {
					if (data) {
						onSubmitEditShare(data);
					}
					setShowEdit(false);
					setInitialNewShareData(undefined); // Clear prefill data after dialog closes
				}}
				onDeleteSubmit={onSubmitDeleteShare}
			/>

			{/* Main Layout Grid */}
			<Grid container spacing={2} sx={{ minHeight: "calc(100vh - 200px)" }} data-tutor={`reactour__tab${TabIDs.SHARES}__step0`}>
				{/* Left Panel - Tree View */}
				<Grid size={{ xs: 12, md: 4, lg: 3 }}>
					<Paper sx={{ height: "100%", p: 1 }} data-tutor={`reactour__tab${TabIDs.SHARES}__step3`}>
						<Stack
							direction="row"
							justifyContent="space-between"
							alignItems="center"
							sx={{ mb: 2, px: 2 }}
						>
							<Typography variant="h6">
								Shares
							</Typography>
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
									<IconButton
										id="create_new_share"
										color="primary"
										aria-label={
											evdata?.hello?.read_only
												? "Cannot create share in read-only mode"
												: !canCreateNewShare
													? "No unshared mount points available to create a new share."
													: "Create new share"
										}
										size="small"
										onClick={() => {
											if (!evdata?.hello?.read_only && canCreateNewShare) {
												setSelectedShareKey(undefined);
												setSelectedShare(null);
												setShowEdit(true);
												setInitialNewShareData(undefined);
											}
										}}
										disabled={evdata?.hello?.read_only || !canCreateNewShare}
									>
										<AddIcon />
									</IconButton>
								</span>
							</Tooltip>
						</Stack>
						<SharesTreeView
							shares={shares}
							selectedShareKey={selectedShareKey}
							onShareSelect={handleShareSelect}
							protectedMode={evdata?.hello?.protected_mode === true}
							readOnly={evdata?.hello?.read_only === true}
							expandedItems={expandedGroups}
							onExpandedItemsChange={setExpandedGroups}
						/>
					</Paper>
				</Grid>

				{/* Right Panel - Details and Edit Form */}
				<Grid size={{ xs: 12, md: 8, lg: 9 }}>
					<Paper sx={{ height: "100%", overflow: "hidden" }}>
						{selectedShare && selectedShareKey ? (
							<ShareDetailsPanel
								share={selectedShare}
								shareKey={selectedShareKey}
								onEdit={onSubmitEditShare}
								onDelete={onSubmitDeleteShare}
								onEditClick={() => setShowEdit(true)}
								onCancelEdit={() => setShowEdit(false)}
								isEditing={showEdit}
							>
								{/* Embedded Edit Form */}
								<ShareEditForm
									shareData={{
										...selectedShare,
										org_name: selectedShare.name || "",
									}}
									shares={shares}
									onSubmit={(data) => {
										onSubmitEditShare(data);
										setShowEdit(false);
									}}
									onDelete={onSubmitDeleteShare}
									disabled={evdata?.hello?.read_only === true}
								/>
							</ShareDetailsPanel>
						) : (
							<Box
								sx={{
									display: "flex",
									alignItems: "center",
									justifyContent: "center",
									height: "100%",
									color: "text.secondary",
								}}
							>
								<Typography variant="h6">
									Select a share from the tree to view details
								</Typography>
							</Box>
						)}
					</Paper>
				</Grid>
			</Grid>
		</>
	);
}