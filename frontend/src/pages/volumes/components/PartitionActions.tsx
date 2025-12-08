import {
	faPlug,
	faPlugCircleBolt,
	faPlugCircleExclamation,
	faPlugCircleMinus,
	faPlugCircleXmark,
} from "@fortawesome/free-solid-svg-icons";
import AddIcon from "@mui/icons-material/Add";
import MoreVertIcon from "@mui/icons-material/MoreVert";
import ScheduleOutlinedIcon from "@mui/icons-material/ScheduleOutlined";
import ScheduleIcon from "@mui/icons-material/Schedule";
import ShareIcon from "@mui/icons-material/Share";
import {
	IconButton,
	ListItemIcon,
	ListItemText,
	Menu,
	MenuItem,
	Stack,
	Tooltip,
	useMediaQuery,
	useTheme,
} from "@mui/material";
import { useState } from "react";
import { FontAwesomeSvgIcon } from "../../../components/FontAwesomeSvgIcon";
import { type Partition } from "../../../store/sratApi";

interface PartitionActionsProps {
	partition: Partition;
	protected_mode: boolean;
	onToggleAutomount: (partition: Partition) => void;
	onMount: (partition: Partition) => void;
	onUnmount: (partition: Partition, force: boolean) => void;
	onCreateShare: (partition: Partition) => void;
	onGoToShare: (partition: Partition) => void;
}

export function PartitionActions({
	partition,
	protected_mode,
	onToggleAutomount,
	onMount,
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

	// Check if partition is read-only or system partition
	if (
		protected_mode ||
		partition.name?.startsWith("hassos-") ||
		Object.values(partition.host_mount_point_data || {}).length > 0
	) {
		return null;
	}

	const mpds = Object.values(partition.mount_point_data || {});
	const mountPointCount = mpds.length;

	// Determine action items based on mount point rules
	const actionItems = [];

	// Rule 1: No mountpoint --> mount action
	if (mountPointCount === 0) {
		actionItems.push({
			key: "mount",
			title: "Mount Partition",
			icon: <FontAwesomeSvgIcon icon={faPlug} />,
			onClick: () => onMount(partition),
		});
	}
	// Rule 2: More than one mountpoint --> no actions
	else if (mountPointCount > 1) {
		return null;
	}
	// Single mountpoint: apply conditional rules
	else {
		const mpd = mpds[0];
		const isMounted = mpd?.is_mounted;
		const hasEnabledShare = mpd?.share && mpd?.share.disabled === false;
		const hasShare = mpd?.share !== null && mpd?.share !== undefined;

		// Add automount toggle if mountpoint exists (unless mounted with enabled share)
		const canShowAutomount = !(isMounted && hasEnabledShare);
		if (canShowAutomount) {
			if (mpd?.is_to_mount_at_startup) {
				actionItems.push({
					key: "disable-automount",
					title: "Disable automatic mount",
					icon: <FontAwesomeSvgIcon icon={faPlugCircleXmark} />,
					onClick: () => onToggleAutomount(partition),
				});
			} else {
				actionItems.push({
					key: "enable-automount",
					title: "Enable automatic mount",
					icon: <FontAwesomeSvgIcon icon={faPlugCircleBolt} />,
					onClick: () => onToggleAutomount(partition),
				});
			}
		}

		// Rule 3: Mountpoint but unmounted --> mount action
		if (!isMounted) {
			actionItems.push({
				key: "mount",
				title: "Mount Partition",
				icon: <FontAwesomeSvgIcon icon={faPlug} />,
				onClick: () => onMount(partition),
			});
		}
		// Mounted cases
		else {
			// Rule 7: Mounted with enabled share --> only go to share
			if (hasEnabledShare) {
				actionItems.push({
					key: "go-to-share",
					title: "Go to Share",
					icon: <ShareIcon fontSize="small" />,
					onClick: () => onGoToShare(partition),
				});
			}
			// Rule 6: Mounted with no share or disabled share --> unmount actions (if automount not enabled)
			else if (!mpd?.is_to_mount_at_startup) {
				actionItems.push({
					key: "unmount",
					title: "Unmount Partition",
					icon: <FontAwesomeSvgIcon icon={faPlugCircleMinus} />,
					onClick: () => onUnmount(partition, false),
				});
				actionItems.push({
					key: "force-unmount",
					title: "Force Unmount Partition",
					icon: <FontAwesomeSvgIcon icon={faPlugCircleExclamation} />,
					onClick: () => onUnmount(partition, true),
				});

				// Rule 5: Mountpoint hasn't a share --> add share
				if (!hasShare && mpd.path?.startsWith("/mnt/")) {
					actionItems.push({
						key: "create-share",
						title: "Create Share",
						icon: <AddIcon fontSize="small" />,
						onClick: () => onCreateShare(partition),
					});
				}
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
