import {
    faPlug,
    faPlugCircleBolt,
    faPlugCircleExclamation,
    faPlugCircleMinus,
    faPlugCircleXmark,
} from "@fortawesome/free-solid-svg-icons";
import AddIcon from "@mui/icons-material/Add";
import MoreVertIcon from "@mui/icons-material/MoreVert";
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
import { useState, type ReactElement } from "react";
import { FontAwesomeSvgIcon } from "../../../components/FontAwesomeSvgIcon";
import {
	type Partition
} from "../../../store/sratApi";
import {
	getPartitionActionItems,
	type PartitionActionKey,
} from "./partition-action-items";

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
	const isSmallScreen = useMediaQuery(theme.breakpoints.between("sm", "md"));
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

	const actionItems = getPartitionActionItems({
		partition,
		protectedMode: protected_mode,
		onToggleAutomount,
		onMount,
		onUnmount,
		onCreateShare,
		onGoToShare,
	});

	if (!actionItems || actionItems.length === 0) {
		return null;
	}

	const actionIcons: Record<PartitionActionKey, ReactElement | null> = {
		"mount": <FontAwesomeSvgIcon icon={faPlug} />,
		"enable-automount": <FontAwesomeSvgIcon icon={faPlugCircleBolt} />,
		"disable-automount": <FontAwesomeSvgIcon icon={faPlugCircleXmark} />,
		"unmount": <FontAwesomeSvgIcon icon={faPlugCircleMinus} />,
		"force-unmount": <FontAwesomeSvgIcon icon={faPlugCircleExclamation} />,
		"create-share": <AddIcon fontSize="small" />,
		"go-to-share": <ShareIcon fontSize="small" />,
		"check-filesystem": null,
		"set-label": null,
		"format": null,
	};

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
							<ListItemIcon>{actionIcons[action.key]}</ListItemIcon>
							<ListItemText>{action.title}</ListItemText>
						</MenuItem>
					))}
				</Menu>
			</>
		);
	}

	return (
		<Stack direction="row" spacing={0} alignItems="center" sx={{ pr: 1 }}>
			{actionItems.
				filter((action) => actionIcons[action.key]).
				map((action) => (
					<Tooltip title={action.title} key={action.key}>
						<IconButton
							onClick={(e) => {
								e.stopPropagation();
								action.onClick();
							}}
							edge="end"
							aria-label={action.title.toLowerCase()}
							size="small"
							color={action.color}
						>
							{actionIcons[action.key]}
						</IconButton>
					</Tooltip>
				))}
		</Stack>
	);
}
