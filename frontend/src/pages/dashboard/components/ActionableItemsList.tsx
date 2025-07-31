import {
	Alert,
	Box,
	Button,
	ButtonGroup,
	CircularProgress,
	List,
	ListItem,
	ListItemText,
	Typography,
} from "@mui/material";
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import VisibilityIcon from '@mui/icons-material/Visibility';
import { useNavigate } from "react-router-dom";
import { type LocationState, TabIDs } from "../../../store/locationState";
import type { Partition } from "../../../store/sratApi";
import { decodeEscapeSequence } from "../metrics/utils";
import { useIgnoredIssues } from '../../../hooks/issueHooks';

interface ActionablePartitionItem {
	partition: Partition;
	action: "mount" | "share";
	id: string;
}

interface ActionableItemsListProps {
	actionablePartitions: { partition: Partition; action: "mount" | "share" }[];
	isLoading: boolean;
	error: Error | null | undefined | {};
	showIgnored?: boolean;
}

export function ActionableItemsList({
	actionablePartitions,
	isLoading,
	error,
	showIgnored = false,
}: ActionableItemsListProps) {
	const navigate = useNavigate();
	const { isIssueIgnored, ignoreIssue, unignoreIssue } = useIgnoredIssues();

	const itemsWithIds = actionablePartitions.map(({ partition, action }) => ({
		partition,
		action,
		id: `partition-${partition.id || partition.device}-${action}`,
	}));

	const handleMount = (_partition: Partition) => {
		navigate("/", { state: { tabId: TabIDs.VOLUMES } as LocationState });
	};

	const handleCreateShare = (partition: Partition) => {
		const firstMountPointData = partition.mount_point_data?.[0];
		if (firstMountPointData) {
			navigate("/", {
				state: {
					tabId: TabIDs.SHARES,
					newShareData: firstMountPointData,
				} as LocationState,
			});
		}
	};

	if (isLoading) {
		return (
			<Box
				sx={{ display: "flex", justifyContent: "center", alignItems: "center" }}
			>
				<CircularProgress />
			</Box>
		);
	}

	if (error) {
		return <Alert severity="error">Could not load volume information.</Alert>;
	}

	const visibleItems = itemsWithIds.filter(({ id }) =>
		showIgnored || !isIssueIgnored(id)
	);

	if (visibleItems.length === 0) {
		if (itemsWithIds.length === 0) {
			return <Typography>No actionable items at the moment.</Typography>;
		}
		return <Typography>No {showIgnored ? '' : 'visible '}actionable items at the moment.</Typography>;
	}

	return (
		<>
			<Typography variant="body2" sx={{ mb: 2 }}>
				You have volumes that are ready for use. Take action to make them
				available to the system.
			</Typography>
			<List dense>
				{visibleItems.map(({ partition, action, id }) => {
					const isIgnored = isIssueIgnored(id);
					// When showIgnored is false, show only non-ignored items
					// When showIgnored is true, show all items
					if (!showIgnored && isIgnored) {
						return null;
					}

					return (
						<ListItem
							key={id}
							secondaryAction={
								<ButtonGroup size="small">
									<Button
										variant="contained"
										onClick={() =>
											action === "mount"
												? handleMount(partition)
												: handleCreateShare(partition)
										}
									>
										{action === "mount" ? "Mount" : "Create Share"}
									</Button>
									<Button
										variant="outlined"
										onClick={() => isIgnored ? unignoreIssue(id) : ignoreIssue(id)}
										startIcon={isIgnored ? <VisibilityIcon /> : <VisibilityOffIcon />}
									>
										{isIgnored ? 'Show' : 'Hide'}
									</Button>
								</ButtonGroup>
							}
						>
							<ListItemText
								primary={decodeEscapeSequence(
									partition.name || partition.device || "Unknown Partition",
								)}
								secondary={
									action === "mount"
										? "This partition is not mounted."
										: "This partition is mounted but not shared."
								}
							/>
						</ListItem>
					);
				})}
			</List>
		</>
	);
}
