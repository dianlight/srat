import {
	faPlug,
	faPlugCircleMinus,
	faPlugCircleXmark,
} from "@fortawesome/free-solid-svg-icons";
import AddIcon from "@mui/icons-material/Add";
import MoreVertIcon from "@mui/icons-material/MoreVert";
import ShareIcon from "@mui/icons-material/Share";
import UpdateIcon from "@mui/icons-material/Update";
import UpdateDisabledIcon from "@mui/icons-material/UpdateDisabled";
import VisibilityIcon from "@mui/icons-material/Visibility";
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
	onViewSettings: (partition: Partition) => void;
	onUnmount: (partition: Partition, force: boolean) => void;
	onCreateShare: (partition: Partition) => void;
	onGoToShare: (partition: Partition) => void;
}

export function PartitionActions({
	partition,
	protected_mode,
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
		protected_mode ||
		//partition.system ||
		partition.name?.startsWith("hassos-") ||
		(partition.host_mount_point_data &&
			partition.host_mount_point_data.length > 0)
	) {
		console.log("Partition is read-only or system partition", protected_mode, partition);
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
